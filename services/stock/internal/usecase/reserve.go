package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/services/stock/internal/domain"
)

// Tên sự kiện (docs/specs/events.md, mục 2) và tên consumer inbox (Q12).
const (
	EventOrderApproved     = "mini.ord.order.approved"
	EventReservationHeld   = "mini.stk.reservation.held"
	EventReservationFailed = "mini.stk.reservation.failed"
	ConsumerReserve        = "stock.reserve"
	source                 = "stock"
)

// approvedData là data của mini.ord.order.approved.
type approvedData struct {
	OrderID uuid.UUID `json:"orderId"`
	Lines   []struct {
		SKU      string `json:"sku"`
		Quantity int32  `json:"quantity"`
	} `json:"lines"`
}

type heldData struct {
	OrderID       uuid.UUID `json:"orderId"`
	ReservationID uuid.UUID `json:"reservationId"`
}

type failedData struct {
	OrderID   uuid.UUID `json:"orderId"`
	ShortSKUs []string  `json:"shortSkus"`
}

// ReservationRepo khóa tồn và ghi phiếu giữ hàng, trong giao dịch đang mở.
type ReservationRepo interface {
	LockLevels(ctx context.Context, tx pgx.Tx, skus []string) ([]domain.Level, error)
	Exists(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (bool, error)
	Hold(ctx context.Context, tx pgx.Tx, id, orderID uuid.UUID, plan domain.Plan) error
}

// OutboxWriter ghi phong bì vào outbox trong giao dịch đang mở.
type OutboxWriter interface {
	Add(ctx context.Context, tx pgx.Tx, env outbox.Envelope) error
}

// Reserver giữ hàng khi nhận "đơn đã duyệt" (consumer stock.reserve).
type Reserver struct {
	Repo   ReservationRepo
	Outbox OutboxWriter
	Now    func() time.Time
}

// Handle là inbox.Handler: giữ hàng cho cả đơn rồi ghi outbox sự kiện kết quả,
// tất cả trong giao dịch đã có dòng inbox.
func (r *Reserver) Handle(ctx context.Context, tx pgx.Tx, env outbox.Envelope) error {
	data, wants, err := decodeApproved(env)
	if err != nil {
		return err
	}
	done, err := r.Repo.Exists(ctx, tx, data.OrderID)
	if err != nil || done {
		return err // đã có phiếu cho đơn này: không giữ lần hai
	}
	skus := make([]string, 0, len(wants))
	for _, w := range wants {
		skus = append(skus, w.SKU)
	}
	levels, err := r.Repo.LockLevels(ctx, tx, skus)
	if err != nil {
		return err
	}
	plan := domain.PlanReservation(levels, wants)
	if !plan.OK {
		return r.emit(ctx, tx, r.resultMeta(env, EventReservationFailed, data.OrderID),
			failedData{OrderID: data.OrderID, ShortSKUs: plan.ShortSKUs})
	}
	resID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("reserve: sinh id phiếu: %w", err)
	}
	if err := r.Repo.Hold(ctx, tx, resID, data.OrderID, plan); err != nil {
		return err
	}
	return r.emit(ctx, tx, r.resultMeta(env, EventReservationHeld, data.OrderID),
		heldData{OrderID: data.OrderID, ReservationID: resID})
}

// decodeApproved đọc data của "đơn đã duyệt" và kiểm hợp đồng: có orderId, có ít
// nhất một dòng, mỗi dòng có mã và số lượng > 0. Sai thì là phong bì hỏng (Term),
// vì gửi lại cũng không sửa được. Không kiểm thì số lượng âm sẽ làm giảm reserved,
// còn số lượng 0 vướng CHECK của reservation_lines và bị gửi lại mãi.
func decodeApproved(env outbox.Envelope) (approvedData, []domain.Want, error) {
	var data approvedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return data, nil, fmt.Errorf("%w: data của %s: %v", outbox.ErrMalformed, env.Type, err)
	}
	if data.OrderID == uuid.Nil || len(data.Lines) == 0 {
		return data, nil, fmt.Errorf("%w: %s thiếu orderId hoặc lines", outbox.ErrMalformed, env.Type)
	}
	wants := make([]domain.Want, 0, len(data.Lines))
	for _, l := range data.Lines {
		if l.SKU == "" || l.Quantity <= 0 {
			return data, nil, fmt.Errorf("%w: %s có dòng sai (sku %q, quantity %d)", outbox.ErrMalformed, env.Type, l.SKU, l.Quantity)
		}
		wants = append(wants, domain.Want{SKU: l.SKU, Quantity: l.Quantity})
	}
	return data, wants, nil
}

// resultMeta: correlationId = orderId cho cả hành trình (events.md, mục 3), lấy từ
// data chứ không chép correlationId của bên gửi; causationId = id sự kiện gây ra.
func (r *Reserver) resultMeta(cause outbox.Envelope, typ string, orderID uuid.UUID) outbox.Meta {
	return outbox.Meta{
		Type: typ, Source: source, TenantID: cause.TenantID,
		CorrelationID: orderID.String(), CausationID: &cause.ID, OccurredAt: r.now(),
	}
}

// emit ghi sự kiện kết quả vào outbox, trong giao dịch đang mở.
func (r *Reserver) emit(ctx context.Context, tx pgx.Tx, meta outbox.Meta, data any) error {
	env, err := outbox.NewEnvelope(meta, data)
	if err != nil {
		return err
	}
	return r.Outbox.Add(ctx, tx, env)
}

func (r *Reserver) now() time.Time {
	if r.Now == nil {
		return time.Now().UTC()
	}
	return r.Now()
}
