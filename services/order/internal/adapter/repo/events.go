package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/repo/sqlcdb"
)

// Outbox cài đặt outbox.Store (cho relay) và hàm ghi phong bì (cho usecase).
type Outbox struct{}

// Add ghi phong bì vào outbox trong giao dịch nghiệp vụ đang mở.
func (Outbox) Add(ctx context.Context, tx pgx.Tx, env outbox.Envelope) error {
	payload, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("repo: mã hóa phong bì: %w", err)
	}
	err = sqlcdb.New(tx).InsertOutbox(ctx, sqlcdb.InsertOutboxParams{ID: env.ID, Type: env.Type, Payload: payload})
	if err != nil {
		return fmt.Errorf("repo: insert outbox: %w", err)
	}
	return nil
}

// Claim lấy một lô outbox chưa gửi, khóa SKIP LOCKED.
func (Outbox) Claim(ctx context.Context, tx pgx.Tx, limit int32) ([]outbox.Message, error) {
	rows, err := sqlcdb.New(tx).ClaimOutbox(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("repo: claim outbox: %w", err)
	}
	msgs := make([]outbox.Message, 0, len(rows))
	for _, r := range rows {
		msgs = append(msgs, outbox.Message{ID: r.ID, Type: r.Type, Payload: r.Payload})
	}
	return msgs, nil
}

// MarkPublished đánh dấu các dòng đã gửi.
func (Outbox) MarkPublished(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) error {
	if err := sqlcdb.New(tx).MarkOutboxPublished(ctx, ids); err != nil {
		return fmt.Errorf("repo: mark outbox published: %w", err)
	}
	return nil
}

// Inbox cài đặt inbox.Store.
type Inbox struct{}

// Claim chèn dòng inbox; false nghĩa là sự kiện đã xử lý rồi.
func (Inbox) Claim(ctx context.Context, tx pgx.Tx, consumer string, eventID uuid.UUID) (bool, error) {
	n, err := sqlcdb.New(tx).ClaimInbox(ctx, sqlcdb.ClaimInboxParams{Consumer: consumer, EventID: eventID})
	if err != nil {
		return false, fmt.Errorf("repo: claim inbox: %w", err)
	}
	return n == 1, nil
}
