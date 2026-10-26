// Package usecase điều phối nghiệp vụ của service kho: kiểm danh tính,
// mở giao dịch có tenant, gọi repo và domain.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/tenant"
	"github.com/tahnhuy/mini_neo/services/stock/internal/domain"
)

// ErrUnauthenticated: lời gọi không có danh tính tenant.
var ErrUnauthenticated = errors.New("usecase: thiếu danh tính")

// TxRunner mở giao dịch có tenant; bản thật là *db.DB của pkg/db.
type TxRunner interface {
	InTx(ctx context.Context, fn func(tx pgx.Tx) error) error
}

// ItemRepo đọc mặt hàng trong giao dịch đang mở.
type ItemRepo interface {
	ListItems(ctx context.Context, tx pgx.Tx) ([]domain.Item, error)
	GetPrices(ctx context.Context, tx pgx.Tx, skus []string) ([]domain.Item, error)
}

// Items là các usecase về mặt hàng.
type Items struct {
	tx   TxRunner
	repo ItemRepo
}

// NewItems tạo usecase mặt hàng.
func NewItems(tx TxRunner, repo ItemRepo) *Items {
	return &Items{tx: tx, repo: repo}
}

// ListItems trả mặt hàng của tenant người gọi; người dùng bất kỳ của tenant đều được xem.
func (u *Items) ListItems(ctx context.Context) ([]domain.Item, error) {
	if _, ok := tenant.FromContext(ctx); !ok {
		return nil, ErrUnauthenticated
	}
	var items []domain.Item
	err := u.tx.InTx(ctx, func(tx pgx.Tx) error {
		var err error
		items, err = u.repo.ListItems(ctx, tx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	return items, nil
}

// GetPrices trả đơn giá của các mã tìm thấy. Chỉ service đơn gọi, khi tạo đơn
// (Q8: không có xác thực riêng cho lời gọi nội bộ, chỉ cần danh tính tenant).
func (u *Items) GetPrices(ctx context.Context, skus []string) ([]domain.Item, error) {
	if _, ok := tenant.FromContext(ctx); !ok {
		return nil, ErrUnauthenticated
	}
	var items []domain.Item
	err := u.tx.InTx(ctx, func(tx pgx.Tx) error {
		var err error
		items, err = u.repo.GetPrices(ctx, tx, skus)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get prices: %w", err)
	}
	return items, nil
}
