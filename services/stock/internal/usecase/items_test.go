package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/tenant"
	"github.com/tahnhuy/mini_neo/services/stock/internal/domain"
)

type fakeTx struct{ calls int }

func (f *fakeTx) InTx(_ context.Context, fn func(pgx.Tx) error) error {
	f.calls++
	return fn(nil)
}

type fakeRepo struct {
	items []domain.Item
	err   error
}

func (f fakeRepo) ListItems(context.Context, pgx.Tx) ([]domain.Item, error) {
	return f.items, f.err
}

func (f fakeRepo) GetPrices(context.Context, pgx.Tx, []string) ([]domain.Item, error) {
	return f.items, f.err
}

func TestGetPricesRequiresTenant(t *testing.T) {
	tx := &fakeTx{}
	_, err := NewItems(tx, fakeRepo{}).GetPrices(context.Background(), []string{"BAN-01"})
	if !errors.Is(err, ErrUnauthenticated) || tx.calls != 0 {
		t.Fatalf("err = %v, tx calls = %d; want ErrUnauthenticated, 0", err, tx.calls)
	}
}

func ctxTenantA() context.Context {
	return tenant.WithIdentity(context.Background(), tenant.Identity{
		TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		UserID:   "an",
	})
}

func TestListItems(t *testing.T) {
	repoErr := errors.New("db down")
	tests := []struct {
		name      string
		ctx       context.Context
		repo      fakeRepo
		wantErr   error
		wantLen   int
		wantTxRun int
	}{
		{"thiếu danh tính", context.Background(), fakeRepo{}, ErrUnauthenticated, 0, 0},
		{"lỗi repo được bọc lại", ctxTenantA(), fakeRepo{err: repoErr}, repoErr, 0, 1},
		{"trả mặt hàng", ctxTenantA(), fakeRepo{items: make([]domain.Item, 3)}, nil, 3, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &fakeTx{}
			got, err := NewItems(tx, tt.repo).ListItems(tt.ctx)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if len(got) != tt.wantLen || tx.calls != tt.wantTxRun {
				t.Errorf("len = %d, tx calls = %d; want %d, %d", len(got), tx.calls, tt.wantLen, tt.wantTxRun)
			}
		})
	}
}
