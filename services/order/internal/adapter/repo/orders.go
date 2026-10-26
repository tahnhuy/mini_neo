// Package repo đọc ghi CSDL của service đơn qua mã sqlc.
// Mọi hàm chạy trong giao dịch do usecase mở bằng db.InTx (đã đặt app.tenant_id).
package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/moneyconv"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/repo/sqlcdb"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
	"github.com/tahnhuy/mini_neo/services/order/internal/usecase"
)

// Orders là repo đơn.
type Orders struct{}

// orderCols có cùng trường với GetOrderRow, GetOrderForUpdateRow, ListOrdersRow
// nên đổi kiểu trực tiếp được, chỉ cần một hàm chuyển sang domain.
type orderCols sqlcdb.GetOrderRow

// NextNumber tăng và trả số đếm đơn của tenant (Q7).
func (Orders) NextNumber(ctx context.Context, tx pgx.Tx) (int64, error) {
	n, err := sqlcdb.New(tx).NextOrderNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("repo: next order number: %w", err)
	}
	return n, nil
}

// Insert ghi đơn và các dòng của đơn.
func (Orders) Insert(ctx context.Context, tx pgx.Tx, o domain.Order) error {
	q := sqlcdb.New(tx)
	err := q.InsertOrder(ctx, sqlcdb.InsertOrderParams{
		ID: o.ID, Code: o.Code, Status: string(o.Status), CreatedBy: o.CreatedBy,
		TotalAmount: moneyconv.FromCents(o.Total.Cents), CurrencyCode: o.Total.Currency, CreatedAt: o.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("repo: insert order: %w", err)
	}
	for _, l := range o.Lines {
		lineID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("repo: sinh id dòng đơn: %w", err)
		}
		err = q.InsertOrderLine(ctx, sqlcdb.InsertOrderLineParams{
			ID: lineID, OrderID: o.ID, Sku: l.SKU, Quantity: l.Quantity,
			UnitPrice: moneyconv.FromCents(l.UnitPrice.Cents), LineAmount: moneyconv.FromCents(l.LineAmount.Cents),
			CurrencyCode: l.UnitPrice.Currency,
		})
		if err != nil {
			return fmt.Errorf("repo: insert order line %s: %w", l.SKU, err)
		}
	}
	return nil
}

// Get đọc đơn kèm các dòng; forUpdate = true thì khóa dòng đơn đến hết giao dịch.
// Không thấy (kể cả đơn của tenant khác, bị RLS ẩn) thì trả usecase.ErrOrderNotFound.
func (Orders) Get(ctx context.Context, tx pgx.Tx, id uuid.UUID, forUpdate bool) (domain.Order, error) {
	q := sqlcdb.New(tx)
	var cols orderCols
	var err error
	if forUpdate {
		var row sqlcdb.GetOrderForUpdateRow
		row, err = q.GetOrderForUpdate(ctx, id)
		cols = orderCols(row)
	} else {
		var row sqlcdb.GetOrderRow
		row, err = q.GetOrder(ctx, id)
		cols = orderCols(row)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, usecase.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("repo: get order: %w", err)
	}
	o, err := cols.toDomain()
	if err != nil {
		return domain.Order{}, err
	}
	o.Lines, err = lines(ctx, q, id)
	return o, err
}

// List đọc tối đa limit đơn mới nhất, không kèm dòng đơn (màn danh sách không cần).
func (Orders) List(ctx context.Context, tx pgx.Tx, limit int32) ([]domain.Order, error) {
	rows, err := sqlcdb.New(tx).ListOrders(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("repo: list orders: %w", err)
	}
	orders := make([]domain.Order, 0, len(rows))
	for _, r := range rows {
		o, err := orderCols(r).toDomain()
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// UpdateDecision ghi kết quả duyệt/từ chối.
func (Orders) UpdateDecision(ctx context.Context, tx pgx.Tx, o domain.Order) error {
	err := sqlcdb.New(tx).UpdateOrderDecision(ctx, sqlcdb.UpdateOrderDecisionParams{
		ID: o.ID, Status: string(o.Status), DecidedBy: o.DecidedBy, RejectReason: o.RejectReason, UpdatedAt: o.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("repo: update order decision: %w", err)
	}
	return nil
}

func (c orderCols) toDomain() (domain.Order, error) {
	total, err := moneyconv.ToCents(c.TotalAmount)
	if err != nil {
		return domain.Order{}, fmt.Errorf("repo: total_amount của %s: %w", c.Code, err)
	}
	return domain.Order{
		ID: c.ID, Code: c.Code, Status: domain.Status(c.Status), CreatedBy: c.CreatedBy,
		DecidedBy: c.DecidedBy, RejectReason: c.RejectReason,
		Total:     domain.Money{Cents: total, Currency: c.CurrencyCode},
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}, nil
}

func lines(ctx context.Context, q *sqlcdb.Queries, orderID uuid.UUID) ([]domain.Line, error) {
	rows, err := q.ListOrderLines(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("repo: list order lines: %w", err)
	}
	out := make([]domain.Line, 0, len(rows))
	for _, r := range rows {
		price, err1 := moneyconv.ToCents(r.UnitPrice)
		amount, err2 := moneyconv.ToCents(r.LineAmount)
		if err := errors.Join(err1, err2); err != nil {
			return nil, fmt.Errorf("repo: tiền của dòng %s: %w", r.Sku, err)
		}
		out = append(out, domain.Line{
			SKU: r.Sku, Quantity: r.Quantity,
			UnitPrice:  domain.Money{Cents: price, Currency: r.CurrencyCode},
			LineAmount: domain.Money{Cents: amount, Currency: r.CurrencyCode},
		})
	}
	return out, nil
}
