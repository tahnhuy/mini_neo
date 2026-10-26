//go:build integration

package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/services/order/internal/app"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
	"github.com/tahnhuy/mini_neo/services/order/internal/usecase"
	"github.com/tahnhuy/mini_neo/services/stock/stocktest"
)

// AC-05: chi duyệt đơn đủ hàng → đơn sang RESERVED; reserved của từng mặt hàng
// tăng đúng số lượng. Đi trọn đường: outbox → relay → NATS → inbox, hai chiều.
func TestReserve_AC05_EnoughStockReserved(t *testing.T) {
	h := newHarness(t, true)
	id := h.createAndApprove(t, usecase.NewLine{SKU: "BAN-01", Quantity: 2}, usecase.NewLine{SKU: "GHE-01", Quantity: 3})
	if got := h.waitDecided(t, id); got != domain.StatusReserved {
		t.Fatalf("status = %s, want RESERVED", got)
	}
	if b, g, d := h.reserved(t, "BAN-01"), h.reserved(t, "GHE-01"), h.reserved(t, "DEN-01"); b != 2 || g != 3 || d != 0 {
		t.Fatalf("reserved BAN-01=%d GHE-01=%d DEN-01=%d, want 2, 3, 0", b, g, d)
	}
	if n := h.queryInt(t, "SELECT count(*) FROM mini_stock.reservations WHERE order_id = $1 AND status = 'HELD'", id); n != 1 {
		t.Fatalf("có %d phiếu HELD, want 1", n)
	}
}

// AC-06: đơn có một dòng thiếu hàng → OUT_OF_STOCK; không dòng nào bị giữ.
func TestReserve_AC06_ShortLineNothingHeld(t *testing.T) {
	h := newHarness(t, true)
	id := h.createAndApprove(t, usecase.NewLine{SKU: "BAN-01", Quantity: 1}, usecase.NewLine{SKU: "DEN-01", Quantity: 2})
	if got := h.waitDecided(t, id); got != domain.StatusOutOfStock {
		t.Fatalf("status = %s, want OUT_OF_STOCK", got)
	}
	if b, d := h.reserved(t, "BAN-01"), h.reserved(t, "DEN-01"); b != 0 || d != 0 {
		t.Fatalf("reserved BAN-01=%d DEN-01=%d, want 0, 0", b, d)
	}
	if n := h.queryInt(t, "SELECT count(*) FROM mini_stock.reservations"); n != 0 {
		t.Fatalf("có %d phiếu giữ hàng, want 0", n)
	}
}

// AC-07: cùng một sự kiện mini.ord.order.approved được giao 2 lần → chỉ giữ hàng một lần.
// Lần giao thứ hai gửi lại đúng payload (cùng id sự kiện) với Nats-Msg-Id khác, như
// relay gửi lại sau khi cửa sổ chống trùng của JetStream đã hết.
func TestInbox_AC07_DuplicateEventReservedOnce(t *testing.T) {
	h := newHarness(t, true)
	id := h.createAndApprove(t, usecase.NewLine{SKU: "BAN-01", Quantity: 2})
	if got := h.waitDecided(t, id); got != domain.StatusReserved {
		t.Fatalf("status = %s, want RESERVED", got)
	}
	approved := h.payloads(t, "mini_order.outbox", usecase.EventOrderApproved)
	if len(approved) != 1 {
		t.Fatalf("có %d sự kiện approved, want 1", len(approved))
	}
	pub := outbox.JetStreamPublisher{JS: h.js}
	if err := pub.Publish(context.Background(), usecase.EventOrderApproved, "giao-lan-2", approved[0]); err != nil {
		t.Fatal(err)
	}
	waitConsumerIdle(t, h.js, "stock-reserve", 2)

	if b := h.reserved(t, "BAN-01"); b != 2 {
		t.Fatalf("reserved BAN-01 = %d, want 2 (giữ một lần)", b)
	}
	if n := len(h.payloads(t, "mini_stock.outbox", "mini.stk.reservation.held")); n != 1 {
		t.Fatalf("kho phát %d sự kiện held, want 1", n)
	}
	if n := h.queryInt(t, "SELECT count(*) FROM mini_stock.inbox WHERE consumer = 'stock.reserve'"); n != 1 {
		t.Fatalf("inbox có %d dòng, want 1", n)
	}
}

// waitConsumerIdle chờ durable đã nhận ít nhất delivered tin và không còn tin chờ ack.
func waitConsumerIdle(t *testing.T, js jetstream.JetStream, durable string, delivered uint64) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		cons, err := js.Consumer(ctx, outbox.StreamName, durable)
		if err == nil {
			info, ierr := cons.Info(ctx)
			cancel()
			if ierr == nil && info.Delivered.Consumer >= delivered && info.NumAckPending == 0 && info.NumPending == 0 {
				return
			}
		} else {
			cancel()
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("consumer %s chưa xử lý xong %d tin sau 15 giây", durable, delivered)
}

// AC-08: hai đơn cùng cần DEN-01 (chỉ còn 1), hai consumer kho xử lý song song.
// Đúng một đơn RESERVED, một đơn OUT_OF_STOCK; reserved không bao giờ vượt on_hand.
// Gọi thẳng processor của consumer (cùng code với consumer thật, gồm inbox và giao dịch)
// từ hai goroutine xuất phát cùng lúc, để chắc hai giao dịch thật sự chồng nhau.
// Chạy ít nhất 20 lần: go test -tags integration -run AC08 -count=20 ./services/order/...
func TestReserve_AC08_ConcurrentLastItem(t *testing.T) {
	h := newHarness(t, false)
	a := h.createAndApprove(t, usecase.NewLine{SKU: "DEN-01", Quantity: 1})
	b := h.createAndApprove(t, usecase.NewLine{SKU: "DEN-01", Quantity: 1})

	approved := h.payloads(t, "mini_order.outbox", usecase.EventOrderApproved)
	warmPool(t, h.pools.App, 4)
	runConcurrently(t, stocktest.ReserveProcessor(h.pools.App), approved)
	held, failed := app.ResultProcessors(h.pools.App, quiet)
	for _, p := range h.payloads(t, "mini_stock.outbox", "mini.stk.reservation.held") {
		mustProcess(t, held.Process(context.Background(), p))
	}
	for _, p := range h.payloads(t, "mini_stock.outbox", "mini.stk.reservation.failed") {
		mustProcess(t, failed.Process(context.Background(), p))
	}

	got := map[domain.Status]int{h.waitDecided(t, a): 1}
	got[h.waitDecided(t, b)]++
	if got[domain.StatusReserved] != 1 || got[domain.StatusOutOfStock] != 1 {
		t.Fatalf("trạng thái hai đơn = %v, want 1 RESERVED + 1 OUT_OF_STOCK", got)
	}
	onHand := h.queryInt(t, "SELECT on_hand FROM mini_stock.stock_items WHERE sku = 'DEN-01'")
	if r := h.reserved(t, "DEN-01"); r != 1 || r > onHand {
		t.Fatalf("DEN-01 reserved = %d, on_hand = %d; want reserved 1 ≤ on_hand", r, onHand)
	}
	if n := h.queryInt(t, "SELECT count(*) FROM mini_stock.reservations"); n != 1 {
		t.Fatalf("có %d phiếu giữ hàng, want 1", n)
	}
}

type processor interface {
	Process(ctx context.Context, payload []byte) error
}

// warmPool mở sẵn n kết nối, để hai goroutine không phải chờ kết nối mới
// (lệch nhau vài ms) và hai giao dịch thật sự chồng nhau.
func warmPool(t *testing.T, pool *pgxpool.Pool, n int) {
	t.Helper()
	conns := make([]*pgxpool.Conn, 0, n)
	for range n {
		c, err := pool.Acquire(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, c)
	}
	for _, c := range conns {
		c.Release()
	}
}

// runConcurrently cho mỗi payload một goroutine, cùng chờ một tín hiệu xuất phát.
func runConcurrently(t *testing.T, p processor, payloads [][]byte) {
	t.Helper()
	if len(payloads) != 2 {
		t.Fatalf("có %d sự kiện approved, want 2", len(payloads))
	}
	start := make(chan struct{})
	errs := make(chan error, len(payloads))
	var wg sync.WaitGroup
	for _, pl := range payloads {
		wg.Go(func() {
			<-start
			errs <- p.Process(context.Background(), pl)
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		mustProcess(t, err)
	}
}

func mustProcess(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("xử lý sự kiện: %v", err)
	}
}
