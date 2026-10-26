package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/services/stock/internal/adapter/repo/sqlcdb"
	"github.com/tahnhuy/mini_neo/services/stock/internal/domain"
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

// Reservations đọc ghi tồn và phiếu giữ hàng.
type Reservations struct{}

// LockLevels khóa dòng stock_items (FOR UPDATE, theo thứ tự sku) rồi mới đọc tồn.
func (Reservations) LockLevels(ctx context.Context, tx pgx.Tx, skus []string) ([]domain.Level, error) {
	rows, err := sqlcdb.New(tx).LockStockLevels(ctx, skus)
	if err != nil {
		return nil, fmt.Errorf("repo: lock stock levels: %w", err)
	}
	levels := make([]domain.Level, 0, len(rows))
	for _, r := range rows {
		levels = append(levels, domain.Level{SKU: r.Sku, OnHand: r.OnHand, Reserved: r.Reserved})
	}
	return levels, nil
}

// Exists báo đơn đã có phiếu giữ hàng chưa (UNIQUE (tenant_id, order_id)).
func (Reservations) Exists(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (bool, error) {
	ok, err := sqlcdb.New(tx).ReservationExists(ctx, orderID)
	if err != nil {
		return false, fmt.Errorf("repo: reservation exists: %w", err)
	}
	return ok, nil
}

// Hold ghi reserved mới cho từng mặt hàng và phiếu giữ hàng HELD kèm các dòng.
func (Reservations) Hold(ctx context.Context, tx pgx.Tx, id, orderID uuid.UUID, plan domain.Plan) error {
	q := sqlcdb.New(tx)
	for _, l := range plan.NewLevels {
		if err := q.SetReserved(ctx, sqlcdb.SetReservedParams{Sku: l.SKU, Reserved: l.Reserved}); err != nil {
			return fmt.Errorf("repo: set reserved %s: %w", l.SKU, err)
		}
	}
	err := q.InsertReservation(ctx, sqlcdb.InsertReservationParams{ID: id, OrderID: orderID, Status: domain.ReservationHeld})
	if err != nil {
		return fmt.Errorf("repo: insert reservation: %w", err)
	}
	for _, w := range plan.Lines {
		err := q.InsertReservationLine(ctx, sqlcdb.InsertReservationLineParams{ReservationID: id, Sku: w.SKU, Quantity: w.Quantity})
		if err != nil {
			return fmt.Errorf("repo: insert reservation line %s: %w", w.SKU, err)
		}
	}
	return nil
}
