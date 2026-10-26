// Package usecase điều phối nghiệp vụ của service đơn: kiểm quyền, chặn tự duyệt,
// mở giao dịch có tenant, gọi domain và repo.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/tenant"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

// Tên quyền; kiểm quyền theo tên, không theo tên người hay vai trò.
const (
	CapCreate  = "order.create"
	CapApprove = "order.approve"
)

// ListLimit: ListOrders trả tối đa 50 đơn mới nhất.
const ListLimit = 50

// TxRunner mở giao dịch có tenant; bản thật là *db.DB của pkg/db.
type TxRunner interface {
	InTx(ctx context.Context, fn func(tx pgx.Tx) error) error
}

// StockClient lấy đơn giá từ service kho. Chỉ trả các mã tìm thấy (Q27).
type StockClient interface {
	GetPrices(ctx context.Context, skus []string) (map[string]domain.Money, error)
}

// OrderRepo đọc ghi đơn trong giao dịch đang mở. Get trả ErrOrderNotFound khi không thấy
// (kể cả đơn của tenant khác, vì RLS ẩn dòng đó).
type OrderRepo interface {
	NextNumber(ctx context.Context, tx pgx.Tx) (int64, error)
	Insert(ctx context.Context, tx pgx.Tx, o domain.Order) error
	Get(ctx context.Context, tx pgx.Tx, id uuid.UUID, forUpdate bool) (domain.Order, error)
	List(ctx context.Context, tx pgx.Tx, limit int32) ([]domain.Order, error)
	UpdateDecision(ctx context.Context, tx pgx.Tx, o domain.Order) error
}

// ErrOrderNotFound là lỗi repo trả khi không thấy đơn.
var ErrOrderNotFound = domain.NewError(domain.KindNotFound, "Không tìm thấy đơn")

// View là đơn kèm allowed_actions của người gọi.
type View struct {
	Order          domain.Order
	AllowedActions []string
}

// Deps gom phụ thuộc của Orders.
type Deps struct {
	Tx    TxRunner
	Repo  OrderRepo
	Stock StockClient
	Now   func() time.Time
	NewID func() (uuid.UUID, error)
}

// Orders là các usecase về đơn.
type Orders struct {
	d Deps
}

// NewOrders tạo usecase đơn; Now và NewID mặc định là giờ UTC và UUIDv7.
func NewOrders(d Deps) *Orders {
	if d.Now == nil {
		d.Now = func() time.Time { return time.Now().UTC() }
	}
	if d.NewID == nil {
		d.NewID = uuid.NewV7
	}
	return &Orders{d: d}
}

// identity đòi cả tenant lẫn người dùng: thiếu x-user-id thì created_by, decided_by
// sẽ rỗng và phép chặn tự duyệt so "" với "" nên vô nghĩa.
func identity(ctx context.Context) (tenant.Identity, error) {
	id, ok := tenant.FromContext(ctx)
	if !ok || id.UserID == "" {
		return tenant.Identity{}, domain.NewError(domain.KindUnauthenticated, "Thiếu danh tính người dùng")
	}
	return id, nil
}

func requireAny(id tenant.Identity, caps ...string) error {
	for _, c := range caps {
		if id.Caps.Has(c) {
			return nil
		}
	}
	return domain.NewError(domain.KindPermissionDenied, "Bạn không có quyền thực hiện thao tác này")
}

func view(o domain.Order, id tenant.Identity) View {
	return View{Order: o, AllowedActions: o.AllowedActions(id.Caps.Has(CapApprove), id.UserID)}
}
