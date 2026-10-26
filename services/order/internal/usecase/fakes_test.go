package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/tenant"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

// Người dùng mẫu, quyền đúng như docs/specs/business-rules.md, mục 4.
var users = map[string][]string{
	"an":   {CapCreate},
	"binh": {CapCreate, CapApprove},
	"chi":  {CapApprove},
	"dung": {CapCreate, CapApprove},
}

func as(user string) context.Context {
	tenantID := "11111111-1111-1111-1111-111111111111"
	if user == "dung" {
		tenantID = "22222222-2222-2222-2222-222222222222"
	}
	return tenant.WithIdentity(context.Background(), tenant.Identity{
		TenantID: uuid.MustParse(tenantID),
		UserID:   user,
		Caps:     tenant.NewCapabilities(users[user]...),
	})
}

type fakeTx struct{}

func (fakeTx) InTx(_ context.Context, fn func(pgx.Tx) error) error { return fn(nil) }

// fakeRepo giữ đơn trong bộ nhớ, như một tenant duy nhất.
type fakeRepo struct {
	n      int64
	orders map[uuid.UUID]domain.Order
}

func newFakeRepo() *fakeRepo { return &fakeRepo{orders: map[uuid.UUID]domain.Order{}} }

func (r *fakeRepo) NextNumber(context.Context, pgx.Tx) (int64, error) { r.n++; return r.n, nil }

func (r *fakeRepo) Insert(_ context.Context, _ pgx.Tx, o domain.Order) error {
	r.orders[o.ID] = o
	return nil
}

func (r *fakeRepo) Get(_ context.Context, _ pgx.Tx, id uuid.UUID, _ bool) (domain.Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return domain.Order{}, ErrOrderNotFound
	}
	return o, nil
}

func (r *fakeRepo) List(context.Context, pgx.Tx, int32) ([]domain.Order, error) {
	out := make([]domain.Order, 0, len(r.orders))
	for _, o := range r.orders {
		out = append(out, o)
	}
	return out, nil
}

func (r *fakeRepo) UpdateDecision(_ context.Context, _ pgx.Tx, o domain.Order) error {
	r.orders[o.ID] = o
	return nil
}

// fakeStock trả giá cố định; ghi lại mã đã hỏi để kiểm giá thật sự lấy từ kho.
type fakeStock struct {
	prices map[string]domain.Money
	asked  []string
}

func (s *fakeStock) GetPrices(_ context.Context, skus []string) (map[string]domain.Money, error) {
	s.asked = append(s.asked, skus...)
	out := map[string]domain.Money{}
	for _, sku := range skus {
		if p, ok := s.prices[sku]; ok {
			out[sku] = p
		}
	}
	return out, nil
}

func vnd(dong int64) domain.Money { return domain.Money{Cents: dong * 100, Currency: "VND"} }

func newTestOrders() (*Orders, *fakeRepo, *fakeStock) {
	repo := newFakeRepo()
	stock := &fakeStock{prices: map[string]domain.Money{"BAN-01": vnd(1250000), "GHE-01": vnd(850000)}}
	u := NewOrders(Deps{
		Tx: fakeTx{}, Repo: repo, Stock: stock,
		Now: func() time.Time { return time.Date(2026, 10, 26, 3, 15, 0, 0, time.UTC) },
	})
	return u, repo, stock
}

// seedOrder tạo một đơn của creator ở trạng thái status, bỏ qua usecase.
func seedOrder(t *testing.T, repo *fakeRepo, creator string, status domain.Status) uuid.UUID {
	t.Helper()
	o, err := domain.NewOrder(uuid.New(), creator, []domain.LineInput{{SKU: "BAN-01", Quantity: 1, UnitPrice: vnd(1250000)}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	o.Status = status
	repo.orders[o.ID] = o
	return o.ID
}

func assertDomainErr(t *testing.T, err error, kind domain.Kind, reason string) {
	t.Helper()
	var de *domain.Error
	if !errors.As(err, &de) {
		t.Fatalf("err = %v, want *domain.Error", err)
	}
	if de.Kind != kind || de.Reason != reason {
		t.Fatalf("err = kind %v reason %q, want kind %v reason %q", de.Kind, de.Reason, kind, reason)
	}
}
