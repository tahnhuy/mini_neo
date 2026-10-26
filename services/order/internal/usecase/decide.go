package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/tenant"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

// Lỗi khi người lập tự duyệt hoặc tự từ chối đơn của mình: cùng mã lý do
// SELF_APPROVAL (Q30), message riêng cho từng hành động.
var (
	selfApprove = &domain.Error{
		Kind:    domain.KindPermissionDenied,
		Reason:  domain.ReasonSelfApproval,
		Message: "Bạn không thể tự duyệt đơn do mình lập",
	}
	selfReject = &domain.Error{
		Kind:    domain.KindPermissionDenied,
		Reason:  domain.ReasonSelfApproval,
		Message: "Bạn không thể tự từ chối đơn do mình lập",
	}
)

// applyFunc chạy bước chuyển trạng thái của domain trong giao dịch đang mở.
type applyFunc func(ctx context.Context, tx pgx.Tx, o *domain.Order, id tenant.Identity) error

// ApproveOrder duyệt đơn. Thứ tự kiểm (Q31): quyền → tự duyệt → chuyển trạng thái.
// Sự kiện "đơn đã duyệt" ghi vào outbox trong cùng giao dịch; usecase không gọi NATS.
func (u *Orders) ApproveOrder(ctx context.Context, orderID uuid.UUID) (View, error) {
	return u.decide(ctx, orderID, selfApprove, func(ctx context.Context, tx pgx.Tx, o *domain.Order, id tenant.Identity) error {
		now := u.d.Now()
		if err := o.Approve(id.UserID, now); err != nil {
			return err
		}
		env, err := approvedEvent(*o, id.TenantID, now)
		if err != nil {
			return err
		}
		return u.d.Outbox.Add(ctx, tx, env)
	})
}

// RejectOrder từ chối đơn với lý do; cùng thứ tự kiểm như ApproveOrder. Không phát sự kiện.
func (u *Orders) RejectOrder(ctx context.Context, orderID uuid.UUID, reason string) (View, error) {
	return u.decide(ctx, orderID, selfReject, func(_ context.Context, _ pgx.Tx, o *domain.Order, id tenant.Identity) error {
		return o.Reject(id.UserID, reason, u.d.Now())
	})
}

// decide khóa dòng đơn (FOR UPDATE) để hai người duyệt cùng lúc không ghi đè nhau,
// rồi chạy apply và lưu đơn, tất cả trong một giao dịch.
// selfErr là lỗi trả khi người gọi là người lập.
func (u *Orders) decide(ctx context.Context, orderID uuid.UUID, selfErr error, apply applyFunc) (View, error) {
	id, err := identity(ctx)
	if err != nil {
		return View{}, err
	}
	if err := requireAny(id, CapApprove); err != nil {
		return View{}, err
	}
	var o domain.Order
	err = u.d.Tx.InTx(ctx, func(tx pgx.Tx) error {
		var err error
		if o, err = u.d.Repo.Get(ctx, tx, orderID, true); err != nil {
			return err
		}
		// Chặn tự duyệt ở usecase, kể cả khi người lập có quyền duyệt.
		if o.CreatedBy == id.UserID {
			return selfErr
		}
		if err := apply(ctx, tx, &o, id); err != nil {
			return err
		}
		return u.d.Repo.UpdateDecision(ctx, tx, o)
	})
	if err != nil {
		return View{}, fmt.Errorf("decide order: %w", err)
	}
	return view(o, id), nil
}
