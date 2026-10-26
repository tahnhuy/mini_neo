// Package repo đọc ghi CSDL của service kho qua mã sqlc.
package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/moneyconv"
	"github.com/tahnhuy/mini_neo/services/stock/internal/adapter/repo/sqlcdb"
	"github.com/tahnhuy/mini_neo/services/stock/internal/domain"
)

// Items là repo mặt hàng; mọi hàm chạy trong giao dịch do usecase mở.
type Items struct{}

// ListItems đọc mặt hàng của tenant hiện tại (RLS lọc theo app.tenant_id).
func (Items) ListItems(ctx context.Context, tx pgx.Tx) ([]domain.Item, error) {
	rows, err := sqlcdb.New(tx).ListItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("repo: list items: %w", err)
	}
	items := make([]domain.Item, 0, len(rows))
	for _, r := range rows {
		cents, err := moneyconv.ToCents(r.UnitPrice)
		if err != nil {
			return nil, fmt.Errorf("repo: unit_price của %s: %w", r.Sku, err)
		}
		items = append(items, domain.Item{
			SKU:       r.Sku,
			Name:      r.Name,
			UnitPrice: domain.Money{Cents: cents, Currency: r.CurrencyCode},
			OnHand:    r.OnHand,
			Reserved:  r.Reserved,
		})
	}
	return items, nil
}

// GetPrices đọc đơn giá của các mã trong skus; mã không có thì không có trong kết quả.
func (Items) GetPrices(ctx context.Context, tx pgx.Tx, skus []string) ([]domain.Item, error) {
	rows, err := sqlcdb.New(tx).GetItemPrices(ctx, skus)
	if err != nil {
		return nil, fmt.Errorf("repo: get item prices: %w", err)
	}
	items := make([]domain.Item, 0, len(rows))
	for _, r := range rows {
		cents, err := moneyconv.ToCents(r.UnitPrice)
		if err != nil {
			return nil, fmt.Errorf("repo: unit_price của %s: %w", r.Sku, err)
		}
		items = append(items, domain.Item{SKU: r.Sku, UnitPrice: domain.Money{Cents: cents, Currency: r.CurrencyCode}})
	}
	return items, nil
}
