package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestEnvelopeJSONMatchesSpec(t *testing.T) {
	tenantA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	env, err := NewEnvelope(Meta{
		Type: "mini.ord.order.approved", Source: "order", TenantID: tenantA,
		CorrelationID: "o-1", OccurredAt: time.Date(2026, 10, 26, 3, 15, 0, 0, time.UTC),
	}, map[string]any{"orderId": "o-1"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(env)
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	for _, k := range []string{"id", "type", "source", "tenantId", "correlationId", "causationId", "occurredAt", "schemaVersion", "data"} {
		if _, ok := got[k]; !ok {
			t.Errorf("phong bì thiếu trường %q: %s", k, raw)
		}
	}
	if got["causationId"] != nil || got["occurredAt"] != "2026-10-26T03:15:00Z" || got["schemaVersion"] != float64(1) {
		t.Errorf("phong bì = %s", raw)
	}
	back, err := Decode(raw)
	if err != nil || back.ID != env.ID || back.TenantID != tenantA {
		t.Fatalf("Decode = %+v, %v", back, err)
	}
}

func TestDecodeMalformed(t *testing.T) {
	for _, in := range []string{"không phải json", `{"id":"00000000-0000-0000-0000-000000000000"}`} {
		if _, err := Decode([]byte(in)); !errors.Is(err, ErrMalformed) {
			t.Errorf("Decode(%q) err = %v, want ErrMalformed", in, err)
		}
	}
}

type fakeTx struct {
	pgx.Tx
	committed bool
}

func (f *fakeTx) Commit(context.Context) error   { f.committed = true; return nil }
func (f *fakeTx) Rollback(context.Context) error { return nil }

type fakeDB struct{ tx *fakeTx }

func (d *fakeDB) Begin(context.Context) (pgx.Tx, error) { d.tx = &fakeTx{}; return d.tx, nil }

type fakeStore struct {
	msgs   []Message
	marked []uuid.UUID
}

func (s *fakeStore) Claim(context.Context, pgx.Tx, int32) ([]Message, error) { return s.msgs, nil }
func (s *fakeStore) MarkPublished(_ context.Context, _ pgx.Tx, ids []uuid.UUID) error {
	s.marked = ids
	return nil
}

type fakePub struct {
	failAt int // gửi lỗi ở tin thứ failAt (bắt đầu từ 1); 0 = không lỗi
	msgIDs []string
}

func (p *fakePub) Publish(_ context.Context, _, msgID string, _ []byte) error {
	if len(p.msgIDs)+1 == p.failAt {
		return errors.New("nats down")
	}
	p.msgIDs = append(p.msgIDs, msgID)
	return nil
}

func TestRelayRunOnce(t *testing.T) {
	msgs := []Message{{ID: uuid.New(), Type: "a"}, {ID: uuid.New(), Type: "b"}, {ID: uuid.New(), Type: "c"}}
	tests := []struct {
		name       string
		failAt     int
		wantMarked int
		wantErr    bool
	}{
		{"gửi hết", 0, 3, false},
		{"lỗi ở tin thứ 2: chỉ đánh dấu tin 1", 2, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, store, pub := &fakeDB{}, &fakeStore{msgs: msgs}, &fakePub{failAt: tt.failAt}
			r := &Relay{DB: db, Store: store, Pub: pub, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
			n, err := r.RunOnce(context.Background())
			if (err != nil) != tt.wantErr || n != tt.wantMarked || len(store.marked) != tt.wantMarked {
				t.Fatalf("n = %d, marked = %d, err = %v", n, len(store.marked), err)
			}
			if !db.tx.committed {
				t.Error("phải commit để lưu các tin đã gửi")
			}
			if len(pub.msgIDs) > 0 && pub.msgIDs[0] != msgs[0].ID.String() {
				t.Errorf("Nats-Msg-Id = %s, want id sự kiện %s", pub.msgIDs[0], msgs[0].ID)
			}
		})
	}
}

func TestRelayRunStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	r := &Relay{DB: &fakeDB{}, Store: &fakeStore{}, Pub: &fakePub{}, Interval: time.Millisecond,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	go func() { done <- r.Run(ctx) }()
	time.Sleep(5 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("relay không dừng khi ctx bị hủy")
	}
}
