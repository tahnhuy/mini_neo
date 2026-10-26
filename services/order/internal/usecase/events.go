package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

// Tên sự kiện (docs/specs/events.md, mục 2) và tên consumer inbox (Q12).
const (
	EventOrderApproved        = "mini.ord.order.approved"
	EventReservationHeld      = "mini.stk.reservation.held"
	EventReservationFailed    = "mini.stk.reservation.failed"
	ConsumerReservationHeld   = "order.reservation_held"
	ConsumerReservationFailed = "order.reservation_failed"
	source                    = "order"
)

// OutboxWriter ghi phong bì vào outbox trong giao dịch đang mở.
type OutboxWriter interface {
	Add(ctx context.Context, tx pgx.Tx, env outbox.Envelope) error
}

type approvedLine struct {
	SKU      string `json:"sku"`
	Quantity int32  `json:"quantity"`
}

// approvedData là data của mini.ord.order.approved. Không mang tiền.
type approvedData struct {
	OrderID uuid.UUID      `json:"orderId"`
	Lines   []approvedLine `json:"lines"`
}

// approvedEvent dựng phong bì "đơn đã duyệt"; correlationId = orderId, chưa có causationId.
func approvedEvent(o domain.Order, tenantID uuid.UUID, now time.Time) (outbox.Envelope, error) {
	data := approvedData{OrderID: o.ID, Lines: make([]approvedLine, 0, len(o.Lines))}
	for _, l := range o.Lines {
		data.Lines = append(data.Lines, approvedLine{SKU: l.SKU, Quantity: l.Quantity})
	}
	return outbox.NewEnvelope(outbox.Meta{
		Type: EventOrderApproved, Source: source, TenantID: tenantID,
		CorrelationID: o.ID.String(), OccurredAt: now,
	}, data)
}

// resultData là phần chung của mini.stk.reservation.held và .failed mà đơn cần.
type resultData struct {
	OrderID uuid.UUID `json:"orderId"`
}

// ResultHandler nhận kết quả giữ hàng của kho (consumer order.reservation_held / _failed).
type ResultHandler struct {
	Repo OrderRepo
	Now  func() time.Time
	Log  *slog.Logger
}

// Held là inbox.Handler cho mini.stk.reservation.held.
func (h *ResultHandler) Held(ctx context.Context, tx pgx.Tx, env outbox.Envelope) error {
	return h.apply(ctx, tx, env, true)
}

// Failed là inbox.Handler cho mini.stk.reservation.failed.
func (h *ResultHandler) Failed(ctx context.Context, tx pgx.Tx, env outbox.Envelope) error {
	return h.apply(ctx, tx, env, false)
}

// apply chuyển đơn sang RESERVED / OUT_OF_STOCK. Đơn không còn ở APPROVED (hoặc
// không thấy) thì bỏ qua; dòng inbox vẫn được commit nên lần giao lại không xử lý nữa.
func (h *ResultHandler) apply(ctx context.Context, tx pgx.Tx, env outbox.Envelope, held bool) error {
	var data resultData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return fmt.Errorf("%w: data của %s: %v", outbox.ErrMalformed, env.Type, err)
	}
	o, err := h.Repo.Get(ctx, tx, data.OrderID, true)
	if errors.Is(err, ErrOrderNotFound) {
		h.Log.WarnContext(ctx, "bỏ qua kết quả kho cho đơn không có", "order_id", data.OrderID, "event_id", env.ID)
		return nil
	}
	if err != nil {
		return err
	}
	if o.Status != domain.StatusApproved {
		h.Log.InfoContext(ctx, "bỏ qua kết quả kho đến muộn", "order_id", o.ID, "status", o.Status, "event_id", env.ID)
		return nil
	}
	if err := o.ApplyReservation(held, h.now()); err != nil {
		return err
	}
	return h.Repo.UpdateDecision(ctx, tx, o)
}

func (h *ResultHandler) now() time.Time {
	if h.Now == nil {
		return time.Now().UTC()
	}
	return h.Now()
}
