package usecase

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

// Duyệt thành công thì ghi đúng một sự kiện vào outbox; data không mang tiền.
func TestApproveWritesApprovedEvent(t *testing.T) {
	u, repo, _ := newTestOrders()
	ob := u.d.Outbox.(*fakeOutbox)
	id := seedOrder(t, repo, "an", domain.StatusPendingApproval)
	if _, err := u.ApproveOrder(as("chi"), id); err != nil {
		t.Fatal(err)
	}
	if len(ob.envs) != 1 {
		t.Fatalf("outbox có %d sự kiện, want 1", len(ob.envs))
	}
	env := ob.envs[0]
	if env.Type != EventOrderApproved || env.Source != "order" || env.CorrelationID != id.String() || env.CausationID != nil {
		t.Errorf("phong bì = %+v", env)
	}
	var data map[string]any
	_ = json.Unmarshal(env.Data, &data)
	if data["orderId"] != id.String() || len(data) != 2 {
		t.Errorf("data = %s, want chỉ orderId và lines", env.Data)
	}
}

// Không phát sự kiện khi từ chối hoặc khi duyệt bị chặn.
func TestNoEventWhenNotApproved(t *testing.T) {
	u, repo, _ := newTestOrders()
	ob := u.d.Outbox.(*fakeOutbox)
	rejected := seedOrder(t, repo, "an", domain.StatusPendingApproval)
	_, _ = u.RejectOrder(as("chi"), rejected, "Khách đổi ý, không mua nữa")
	_, _ = u.ApproveOrder(as("an"), seedOrder(t, repo, "binh", domain.StatusPendingApproval))
	_, _ = u.ApproveOrder(as("binh"), seedOrder(t, repo, "binh", domain.StatusPendingApproval))
	if len(ob.envs) != 0 {
		t.Fatalf("outbox = %+v, want rỗng", ob.envs)
	}
}

func resultEnv(t *testing.T, orderID uuid.UUID) outbox.Envelope {
	t.Helper()
	env, err := outbox.NewEnvelope(outbox.Meta{Type: EventReservationHeld, Source: "stock", TenantID: uuid.New(),
		CorrelationID: orderID.String(), OccurredAt: time.Now()}, map[string]string{"orderId": orderID.String()})
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func TestResultHandler(t *testing.T) {
	tests := []struct {
		name string
		from domain.Status
		held bool
		want domain.Status
	}{
		{"kho giữ được", domain.StatusApproved, true, domain.StatusReserved},
		{"kho thiếu hàng", domain.StatusApproved, false, domain.StatusOutOfStock},
		{"đơn không còn APPROVED: bỏ qua", domain.StatusReserved, false, domain.StatusReserved},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			id := seedOrder(t, repo, "an", tt.from)
			h := &ResultHandler{Repo: repo, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
			handle := h.Failed
			if tt.held {
				handle = h.Held
			}
			if err := handle(context.Background(), nil, resultEnv(t, id)); err != nil {
				t.Fatal(err)
			}
			if got := repo.orders[id].Status; got != tt.want {
				t.Errorf("status = %s, want %s", got, tt.want)
			}
		})
	}
	h := &ResultHandler{Repo: newFakeRepo(), Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := h.Held(context.Background(), nil, resultEnv(t, uuid.New())); err != nil {
		t.Errorf("đơn không có phải bỏ qua, got %v", err)
	}
}
