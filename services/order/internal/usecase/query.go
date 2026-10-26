package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

// GetOrder trả đơn kèm allowed_actions. Đơn của tenant khác bị RLS ẩn nên trả NOT_FOUND.
func (u *Orders) GetOrder(ctx context.Context, orderID uuid.UUID) (View, error) {
	id, err := identity(ctx)
	if err != nil {
		return View{}, err
	}
	if err := requireAny(id, CapCreate, CapApprove); err != nil {
		return View{}, err
	}
	var o domain.Order
	err = u.d.Tx.InTx(ctx, func(tx pgx.Tx) error {
		var err error
		o, err = u.d.Repo.Get(ctx, tx, orderID, false)
		return err
	})
	if err != nil {
		return View{}, fmt.Errorf("get order: %w", err)
	}
	return view(o, id), nil
}

// ListOrders trả tối đa 50 đơn mới nhất của tenant.
func (u *Orders) ListOrders(ctx context.Context) ([]View, error) {
	id, err := identity(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireAny(id, CapCreate, CapApprove); err != nil {
		return nil, err
	}
	var orders []domain.Order
	err = u.d.Tx.InTx(ctx, func(tx pgx.Tx) error {
		var err error
		orders, err = u.d.Repo.List(ctx, tx, ListLimit)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	views := make([]View, 0, len(orders))
	for _, o := range orders {
		views = append(views, view(o, id))
	}
	return views, nil
}
