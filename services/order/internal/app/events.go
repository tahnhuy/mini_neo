// Package app nối dây phần sự kiện của service đơn: relay outbox và hai consumer
// nhận kết quả giữ hàng. Dùng chung cho cmd/server và test tích hợp.
package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/tahnhuy/mini_neo/pkg/db"
	"github.com/tahnhuy/mini_neo/pkg/inbox"
	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/order/internal/usecase"
)

// EventDeps là phụ thuộc của phần sự kiện.
type EventDeps struct {
	AppPool       *pgxpool.Pool // vai trò mini_app: xử lý sự kiện, chịu RLS
	RelayPool     *pgxpool.Pool // vai trò mini_relay: đọc outbox của mọi tenant
	JS            jetstream.JetStream
	Log           *slog.Logger
	RelayInterval time.Duration // 0 = mặc định 500 ms
}

// StartEvents chạy relay và hai consumer trong goroutine riêng. Mỗi goroutine
// dừng khi ctx bị hủy; hàm trả về chờ chúng dừng hẳn.
func StartEvents(ctx context.Context, d EventDeps) (wait func()) {
	relay := &outbox.Relay{
		DB: d.RelayPool, Store: repo.Outbox{}, Pub: outbox.JetStreamPublisher{JS: d.JS},
		Log: d.Log, Interval: d.RelayInterval,
	}
	held, failed := ResultProcessors(d.AppPool, d.Log)
	subs := []*inbox.Subscription{
		{Durable: "order-reservation_held", Subject: usecase.EventReservationHeld, Proc: held, Log: d.Log},
		{Durable: "order-reservation_failed", Subject: usecase.EventReservationFailed, Proc: failed, Log: d.Log},
	}
	var wg sync.WaitGroup
	wg.Go(func() { _ = relay.Run(ctx) })
	for _, s := range subs {
		wg.Go(func() {
			if err := s.Run(ctx, d.JS); err != nil {
				d.Log.ErrorContext(ctx, "consumer dừng vì lỗi", "durable", s.Durable, "err", err)
			}
		})
	}
	return wg.Wait
}

// ResultProcessors trả processor của hai consumer order.reservation_held và
// order.reservation_failed. Test gọi trực tiếp được (AC-08) mà không qua NATS.
func ResultProcessors(appPool *pgxpool.Pool, log *slog.Logger) (held, failed *inbox.Processor) {
	h := &usecase.ResultHandler{Repo: repo.Orders{}, Log: log}
	tx := db.New(appPool)
	held = &inbox.Processor{Consumer: usecase.ConsumerReservationHeld, Tx: tx, Store: repo.Inbox{}, Handle: h.Held}
	failed = &inbox.Processor{Consumer: usecase.ConsumerReservationFailed, Tx: tx, Store: repo.Inbox{}, Handle: h.Failed}
	return held, failed
}
