package inbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/pkg/tenant"
)

type fakeTx struct{ tenants []uuid.UUID }

func (f *fakeTx) InTx(ctx context.Context, fn func(pgx.Tx) error) error {
	id, _ := tenant.FromContext(ctx)
	f.tenants = append(f.tenants, id.TenantID)
	return fn(nil)
}

// fakeStore giống bảng inbox: khóa chính (consumer, event_id).
type fakeStore struct{ seen map[string]bool }

func (s *fakeStore) Claim(_ context.Context, _ pgx.Tx, consumer string, id uuid.UUID) (bool, error) {
	key := consumer + "/" + id.String()
	if s.seen[key] {
		return false, nil
	}
	s.seen[key] = true
	return true, nil
}

func payload(t *testing.T, tenantID uuid.UUID) []byte {
	t.Helper()
	env, err := outbox.NewEnvelope(outbox.Meta{Type: "mini.ord.order.approved", Source: "order",
		TenantID: tenantID, CorrelationID: "o-1", OccurredAt: time.Now()}, map[string]string{"orderId": "o-1"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(env)
	return raw
}

// Cùng một sự kiện giao hai lần: việc nghiệp vụ chỉ chạy một lần.
func TestProcessDeduplicates(t *testing.T) {
	tenantB := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	tx, calls := &fakeTx{}, 0
	p := &Processor{Consumer: "stock.reserve", Tx: tx, Store: &fakeStore{seen: map[string]bool{}},
		Handle: func(context.Context, pgx.Tx, outbox.Envelope) error { calls++; return nil }}
	msg := payload(t, tenantB)
	for range 2 {
		if err := p.Process(context.Background(), msg); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Errorf("handler chạy %d lần, want 1", calls)
	}
	if len(tx.tenants) != 2 || tx.tenants[0] != tenantB {
		t.Errorf("tenant của giao dịch = %v, want tenant trong phong bì", tx.tenants)
	}
}

func TestProcessErrors(t *testing.T) {
	boom := errors.New("db down")
	p := &Processor{Consumer: "stock.reserve", Tx: &fakeTx{}, Store: &fakeStore{seen: map[string]bool{}},
		Handle: func(context.Context, pgx.Tx, outbox.Envelope) error { return boom }}
	if err := p.Process(context.Background(), []byte("{")); !errors.Is(err, outbox.ErrMalformed) {
		t.Errorf("phong bì hỏng: err = %v, want ErrMalformed", err)
	}
	if err := p.Process(context.Background(), payload(t, uuid.New())); !errors.Is(err, boom) {
		t.Errorf("lỗi nghiệp vụ: err = %v, want bọc %v", err, boom)
	}
}
