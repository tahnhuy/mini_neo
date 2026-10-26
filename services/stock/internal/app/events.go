// Package app nối dây phần sự kiện của service kho: relay outbox và consumer
// stock.reserve. Dùng chung cho cmd/server và test tích hợp.
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
	"github.com/tahnhuy/mini_neo/services/stock/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/stock/internal/usecase"
)

// EventDeps là phụ thuộc của phần sự kiện.
type EventDeps struct {
	AppPool       *pgxpool.Pool // vai trò mini_app: xử lý sự kiện, chịu RLS
	RelayPool     *pgxpool.Pool // vai trò mini_relay: đọc outbox của mọi tenant
	JS            jetstream.JetStream
	Log           *slog.Logger
	RelayInterval time.Duration // 0 = mặc định 500 ms
}

// StartEvents chạy relay và consumer trong goroutine riêng. Mỗi goroutine dừng
// khi ctx bị hủy; hàm trả về chờ chúng dừng hẳn.
func StartEvents(ctx context.Context, d EventDeps) (wait func()) {
	relay := &outbox.Relay{
		DB: d.RelayPool, Store: repo.Outbox{}, Pub: outbox.JetStreamPublisher{JS: d.JS},
		Log: d.Log, Interval: d.RelayInterval,
	}
	sub := &inbox.Subscription{
		Durable: "stock-reserve",
		Subject: usecase.EventOrderApproved,
		Log:     d.Log,
		Proc:    ReserveProcessor(d.AppPool),
	}
	var wg sync.WaitGroup
	wg.Go(func() { _ = relay.Run(ctx) })
	wg.Go(func() {
		if err := sub.Run(ctx, d.JS); err != nil {
			d.Log.ErrorContext(ctx, "consumer stock-reserve dừng vì lỗi", "err", err)
		}
	})
	return wg.Wait
}

// ReserveProcessor trả processor của consumer stock.reserve, để test gọi trực tiếp
// nhiều bản song song (AC-08) mà không qua NATS.
func ReserveProcessor(appPool *pgxpool.Pool) *inbox.Processor {
	return &inbox.Processor{
		Consumer: usecase.ConsumerReserve,
		Tx:       db.New(appPool),
		Store:    repo.Inbox{},
		Handle:   (&usecase.Reserver{Repo: repo.Reservations{}, Outbox: repo.Outbox{}}).Handle,
	}
}
