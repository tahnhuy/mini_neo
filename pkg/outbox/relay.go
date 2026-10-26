package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Message là một dòng outbox chưa gửi.
type Message struct {
	ID      uuid.UUID
	Type    string
	Payload []byte
}

// Store là truy vấn outbox của một service (sqlc), chạy trong giao dịch của relay.
type Store interface {
	// Claim: SELECT ... WHERE published_at IS NULL ORDER BY id LIMIT n FOR UPDATE SKIP LOCKED.
	Claim(ctx context.Context, tx pgx.Tx, limit int32) ([]Message, error)
	// MarkPublished: UPDATE outbox SET published_at = now() WHERE id = ANY(ids).
	MarkPublished(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) error
}

// Publisher gửi một tin lên JetStream với header Nats-Msg-Id = msgID.
type Publisher interface {
	Publish(ctx context.Context, subject, msgID string, payload []byte) error
}

// Beginner mở giao dịch; bản thật là pool của vai trò mini_relay (BYPASSRLS),
// vì relay đọc outbox của mọi tenant.
type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Relay đọc outbox mỗi Interval và gửi lên JetStream.
type Relay struct {
	DB       Beginner
	Store    Store
	Pub      Publisher
	Log      *slog.Logger
	Interval time.Duration // mặc định 500 ms
	Batch    int32         // mặc định 100
}

// Run lặp tới khi ctx bị hủy (server tắt) thì trả nil.
func (r *Relay) Run(ctx context.Context) error {
	interval := r.Interval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := r.RunOnce(ctx); err != nil && ctx.Err() == nil {
				r.Log.ErrorContext(ctx, "relay outbox lỗi", "err", err)
			}
		}
	}
}

// RunOnce gửi một lô; trả số tin đã gửi và đánh dấu.
// Gửi xong mà chết trước khi commit thì lần sau gửi lại: sự kiện có thể đến hai lần,
// bên nhận chống trùng bằng inbox.
func (r *Relay) RunOnce(ctx context.Context) (n int, err error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("relay: begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()
	msgs, err := r.Store.Claim(ctx, tx, r.batch())
	if err != nil {
		return 0, fmt.Errorf("relay: claim: %w", err)
	}
	sent, pubErr := r.publishAll(ctx, msgs)
	if len(sent) > 0 {
		if err = r.Store.MarkPublished(ctx, tx, sent); err != nil {
			return 0, fmt.Errorf("relay: mark published: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("relay: commit: %w", err)
	}
	return len(sent), pubErr
}

// publishAll gửi theo thứ tự id; dừng ở tin lỗi đầu tiên để giữ thứ tự,
// các tin đã gửi trước đó vẫn được đánh dấu.
func (r *Relay) publishAll(ctx context.Context, msgs []Message) ([]uuid.UUID, error) {
	sent := make([]uuid.UUID, 0, len(msgs))
	for _, m := range msgs {
		if err := r.Pub.Publish(ctx, m.Type, m.ID.String(), m.Payload); err != nil {
			return sent, fmt.Errorf("relay: publish %s (%s): %w", m.ID, m.Type, err)
		}
		sent = append(sent, m.ID)
	}
	return sent, nil
}

func (r *Relay) batch() int32 {
	if r.Batch <= 0 {
		return 100
	}
	return r.Batch
}
