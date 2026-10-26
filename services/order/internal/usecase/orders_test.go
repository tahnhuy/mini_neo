package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

// AC-01: an tạo đơn 2 dòng; giá lấy từ kho, tổng đúng đến đồng.
// Phần "client gửi giá bị bỏ qua" test ở handler BFF (Q35): NewLine không có trường giá.
func TestCreateOrder_AC01_TotalFromStockPrice(t *testing.T) {
	u, repo, stock := newTestOrders()
	v, err := u.CreateOrder(as("an"), []NewLine{{"BAN-01", 2}, {"GHE-01", 3}})
	if err != nil {
		t.Fatal(err)
	}
	o := v.Order
	if len(stock.asked) != 2 {
		t.Errorf("đã hỏi kho %v, want 2 mã", stock.asked)
	}
	if o.Lines[0].UnitPrice != vnd(1250000) || o.Lines[1].UnitPrice != vnd(850000) {
		t.Errorf("đơn giá = %+v, %+v; want giá của kho", o.Lines[0].UnitPrice, o.Lines[1].UnitPrice)
	}
	if o.Total != vnd(5050000) {
		t.Errorf("total = %+v, want 5.050.000 VND", o.Total)
	}
	if o.Code != "DH-000001" || o.Status != domain.StatusPendingApproval || o.CreatedBy != "an" {
		t.Errorf("đơn = %s %s %s", o.Code, o.Status, o.CreatedBy)
	}
	if _, ok := repo.orders[o.ID]; !ok {
		t.Error("đơn chưa được lưu")
	}
}

func TestCreateOrderErrors(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		line NewLine
		kind domain.Kind
	}{
		{"thiếu danh tính", context.Background(), NewLine{"BAN-01", 1}, domain.KindUnauthenticated},
		{"thiếu x-user-id", as(""), NewLine{"BAN-01", 1}, domain.KindUnauthenticated},
		{"chi không có order.create", as("chi"), NewLine{"BAN-01", 1}, domain.KindPermissionDenied},
		{"mã hàng không có", as("an"), NewLine{"KHONG-CO", 1}, domain.KindNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, repo, _ := newTestOrders()
			_, err := u.CreateOrder(tt.ctx, []NewLine{tt.line})
			assertDomainErr(t, err, tt.kind, "")
			if len(repo.orders) != 0 {
				t.Error("không được lưu đơn khi lỗi")
			}
		})
	}
}

// AC-02: an (không có order.approve) duyệt đơn → PERMISSION_DENIED;
// allowed_actions của an rỗng nên web không hiện nút Duyệt.
func TestApprove_AC02_MissingCapabilityDenied(t *testing.T) {
	u, repo, _ := newTestOrders()
	id := seedOrder(t, repo, "binh", domain.StatusPendingApproval)

	_, err := u.ApproveOrder(as("an"), id)
	assertDomainErr(t, err, domain.KindPermissionDenied, "")
	if repo.orders[id].Status != domain.StatusPendingApproval {
		t.Errorf("status = %s, want giữ nguyên PENDING_APPROVAL", repo.orders[id].Status)
	}
	v, err := u.GetOrder(as("an"), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.AllowedActions) != 0 {
		t.Errorf("allowed_actions của an = %v, want []", v.AllowedActions)
	}
}

// AC-03: binh duyệt (hoặc từ chối, Q30) đơn do chính binh lập → PERMISSION_DENIED,
// lý do SELF_APPROVAL; đơn giữ nguyên PENDING_APPROVAL.
func TestApprove_AC03_SelfApprovalDenied(t *testing.T) {
	tests := []struct {
		name    string
		call    func(u *Orders, id uuid.UUID) error
		message string
	}{
		{"tự duyệt", func(u *Orders, id uuid.UUID) error {
			_, err := u.ApproveOrder(as("binh"), id)
			return err
		}, "Bạn không thể tự duyệt đơn do mình lập"},
		{"tự từ chối", func(u *Orders, id uuid.UUID) error {
			_, err := u.RejectOrder(as("binh"), id, "Lý do đủ mười ký tự")
			return err
		}, "Bạn không thể tự từ chối đơn do mình lập"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, repo, _ := newTestOrders()
			id := seedOrder(t, repo, "binh", domain.StatusPendingApproval)
			err := tt.call(u, id)
			assertDomainErr(t, err, domain.KindPermissionDenied, domain.ReasonSelfApproval)
			var de *domain.Error
			if errors.As(err, &de) && de.Message != tt.message {
				t.Errorf("message = %q, want %q", de.Message, tt.message)
			}
			if o := repo.orders[id]; o.Status != domain.StatusPendingApproval || o.DecidedBy != "" {
				t.Errorf("đơn bị đổi: %+v", o)
			}
		})
	}
}

// AC-09: duyệt đơn đang REJECTED → FAILED_PRECONDITION, lý do INVALID_TRANSITION.
func TestApprove_AC09_RejectedInvalidTransition(t *testing.T) {
	u, repo, _ := newTestOrders()
	id := seedOrder(t, repo, "an", domain.StatusRejected)
	_, err := u.ApproveOrder(as("chi"), id)
	assertDomainErr(t, err, domain.KindFailedPrecondition, domain.ReasonInvalidTransition)
	if repo.orders[id].Status != domain.StatusRejected {
		t.Errorf("status = %s, want giữ nguyên REJECTED", repo.orders[id].Status)
	}
}

// Thứ tự kiểm Q31: quyền → tự duyệt → chuyển trạng thái.
func TestDecideCheckOrder(t *testing.T) {
	u, repo, _ := newTestOrders()
	rejectedByBinh := seedOrder(t, repo, "binh", domain.StatusRejected)
	_, err := u.ApproveOrder(as("binh"), rejectedByBinh)
	assertDomainErr(t, err, domain.KindPermissionDenied, domain.ReasonSelfApproval)

	_, err = u.ApproveOrder(as("chi"), uuid.New())
	assertDomainErr(t, err, domain.KindNotFound, "")
}

func TestApproveAndReject(t *testing.T) {
	u, repo, _ := newTestOrders()
	a := seedOrder(t, repo, "an", domain.StatusPendingApproval)
	v, err := u.ApproveOrder(as("chi"), a)
	if err != nil || v.Order.Status != domain.StatusApproved || v.Order.DecidedBy != "chi" {
		t.Fatalf("approve: %+v, %v", v.Order, err)
	}
	r := seedOrder(t, repo, "an", domain.StatusPendingApproval)
	_, err = u.RejectOrder(as("chi"), r, "ngắn")
	assertDomainErr(t, err, domain.KindInvalidArgument, "")
	v, err = u.RejectOrder(as("chi"), r, "Khách đổi ý, không mua nữa")
	if err != nil || v.Order.Status != domain.StatusRejected {
		t.Fatalf("reject: %+v, %v", v.Order, err)
	}
}

func TestListOrdersRequiresCapability(t *testing.T) {
	u, repo, _ := newTestOrders()
	seedOrder(t, repo, "an", domain.StatusPendingApproval)
	views, err := u.ListOrders(as("chi"))
	if err != nil || len(views) != 1 || len(views[0].AllowedActions) != 2 {
		t.Fatalf("ListOrders(chi) = %+v, %v", views, err)
	}
}
