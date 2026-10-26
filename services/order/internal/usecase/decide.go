package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

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

// ApproveOrder duyệt đơn. Thứ tự kiểm (Q31): quyền → tự duyệt → chuyển trạng thái.
func (u *Orders) ApproveOrder(ctx context.Context, orderID uuid.UUID) (View, error) {
	return u.decide(ctx, orderID, selfApprove, func(o *domain.Order, by string) error {
		return o.Approve(by, u.d.Now())
	})
}

// RejectOrder từ chối đơn với lý do; cùng thứ tự kiểm như ApproveOrder.
func (u *Orders) RejectOrder(ctx context.Context, orderID uuid.UUID, reason string) (View, error) {
	return u.decide(ctx, orderID, selfReject, func(o *domain.Order, by string) error {
		return o.Reject(by, reason, u.d.Now())
	})
}

// decide khóa dòng đơn (FOR UPDATE) để hai người duyệt cùng lúc không ghi đè nhau,
// rồi chạy apply (bước chuyển trạng thái của domain). selfErr là lỗi trả khi người gọi là người lập.
func (u *Orders) decide(ctx context.Context, orderID uuid.UUID, selfErr error, apply func(*domain.Order, string) error) (View, error) {
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
		if err := apply(&o, id.UserID); err != nil {
			return err
		}
		return u.d.Repo.UpdateDecision(ctx, tx, o)
	})
	if err != nil {
		return View{}, fmt.Errorf("decide order: %w", err)
	}
	return view(o, id), nil
}
