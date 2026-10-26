package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/services/stock/internal/domain"
)

type fakeResRepo struct {
	levels []domain.Level
	exists bool
	held   *domain.Plan
}

func (f *fakeResRepo) LockLevels(context.Context, pgx.Tx, []string) ([]domain.Level, error) {
	return f.levels, nil
}
func (f *fakeResRepo) Exists(context.Context, pgx.Tx, uuid.UUID) (bool, error) { return f.exists, nil }
func (f *fakeResRepo) Hold(_ context.Context, _ pgx.Tx, _, _ uuid.UUID, p domain.Plan) error {
	f.held = &p
	return nil
}

type fakeOutbox struct{ envs []outbox.Envelope }

func (f *fakeOutbox) Add(_ context.Context, _ pgx.Tx, env outbox.Envelope) error {
	f.envs = append(f.envs, env)
	return nil
}

func approvedEnv(t *testing.T, orderID uuid.UUID, lines string) outbox.Envelope {
	t.Helper()
	env, err := outbox.NewEnvelope(outbox.Meta{Type: EventOrderApproved, Source: "order",
		TenantID: uuid.New(), CorrelationID: orderID.String(), OccurredAt: time.Now()},
		json.RawMessage(`{"orderId":"`+orderID.String()+`","lines":`+lines+`}`))
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func TestReserverHandle(t *testing.T) {
	levels := []domain.Level{{SKU: "BAN-01", OnHand: 10}, {SKU: "DEN-01", OnHand: 1}}
	tests := []struct {
		name     string
		lines    string
		exists   bool
		wantType string
		wantHold bool
	}{
		{"đủ hàng", `[{"sku":"BAN-01","quantity":2},{"sku":"DEN-01","quantity":1}]`, false, EventReservationHeld, true},
		{"thiếu một dòng", `[{"sku":"BAN-01","quantity":2},{"sku":"DEN-01","quantity":2}]`, false, EventReservationFailed, false},
		{"đơn đã có phiếu", `[{"sku":"BAN-01","quantity":1}]`, true, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, ob := &fakeResRepo{levels: levels, exists: tt.exists}, &fakeOutbox{}
			orderID := uuid.New()
			cause := approvedEnv(t, orderID, tt.lines)
			if err := (&Reserver{Repo: repo, Outbox: ob}).Handle(context.Background(), nil, cause); err != nil {
				t.Fatal(err)
			}
			if (repo.held != nil) != tt.wantHold {
				t.Errorf("held = %+v, want hold %v", repo.held, tt.wantHold)
			}
			if tt.wantType == "" {
				if len(ob.envs) != 0 {
					t.Errorf("không được phát sự kiện: %+v", ob.envs)
				}
				return
			}
			got := ob.envs[0]
			if len(ob.envs) != 1 || got.Type != tt.wantType || got.Source != "stock" ||
				got.CorrelationID != orderID.String() || got.CausationID == nil || *got.CausationID != cause.ID.String() {
				t.Errorf("sự kiện = %+v", got)
			}
		})
	}
}

// Data sai hợp đồng là phong bì hỏng (consumer Term), không được chạm tồn hay outbox.
func TestReserverHandleMalformed(t *testing.T) {
	tests := map[string]string{
		"không có dòng":    `[]`,
		"số lượng 0":       `[{"sku":"BAN-01","quantity":0}]`,
		"số lượng âm":      `[{"sku":"BAN-01","quantity":-5}]`,
		"thiếu mã hàng":    `[{"sku":"","quantity":1}]`,
		"sai kiểu dữ liệu": `"BAN-01"`,
	}
	for name, lines := range tests {
		t.Run(name, func(t *testing.T) {
			repo, ob := &fakeResRepo{levels: []domain.Level{{SKU: "BAN-01", OnHand: 10, Reserved: 5}}}, &fakeOutbox{}
			err := (&Reserver{Repo: repo, Outbox: ob}).Handle(context.Background(), nil, approvedEnv(t, uuid.New(), lines))
			if !errors.Is(err, outbox.ErrMalformed) {
				t.Fatalf("err = %v, want ErrMalformed", err)
			}
			if repo.held != nil || len(ob.envs) != 0 {
				t.Fatalf("không được giữ hàng hay phát sự kiện: held=%+v envs=%+v", repo.held, ob.envs)
			}
		})
	}
	t.Run("thiếu orderId", func(t *testing.T) {
		err := (&Reserver{Repo: &fakeResRepo{}, Outbox: &fakeOutbox{}}).Handle(context.Background(), nil,
			approvedEnv(t, uuid.Nil, `[{"sku":"BAN-01","quantity":1}]`))
		if !errors.Is(err, outbox.ErrMalformed) {
			t.Fatalf("err = %v, want ErrMalformed", err)
		}
	})
}

// correlationId của sự kiện kết quả là orderId, kể cả khi bên gửi để sai.
func TestReserverCorrelationIsOrderID(t *testing.T) {
	orderID, ob := uuid.New(), &fakeOutbox{}
	cause := approvedEnv(t, orderID, `[{"sku":"BAN-01","quantity":1}]`)
	cause.CorrelationID = "sai"
	r := &Reserver{Repo: &fakeResRepo{levels: []domain.Level{{SKU: "BAN-01", OnHand: 10}}}, Outbox: ob}
	if err := r.Handle(context.Background(), nil, cause); err != nil {
		t.Fatal(err)
	}
	if len(ob.envs) != 1 || ob.envs[0].CorrelationID != orderID.String() {
		t.Fatalf("sự kiện = %+v, want correlationId %s", ob.envs, orderID)
	}
}
