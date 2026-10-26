// Package stocktest dựng service kho thật cho test tích hợp của service khác
// (ví dụ service đơn), vì luật "internal" của Go không cho chúng import
// services/stock/internal. Không dùng trong mã chạy thật.
package stocktest

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"

	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"github.com/tahnhuy/mini_neo/pkg/db"
	"github.com/tahnhuy/mini_neo/pkg/inbox"
	grpcadapter "github.com/tahnhuy/mini_neo/services/stock/internal/adapter/grpc"
	"github.com/tahnhuy/mini_neo/services/stock/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/stock/internal/app"
	"github.com/tahnhuy/mini_neo/services/stock/internal/usecase"
)

// Register gắn StockService thật (usecase, repo, RLS qua db.InTx) vào srv.
func Register(srv *grpc.Server, pool *pgxpool.Pool, log *slog.Logger) {
	items := usecase.NewItems(db.New(pool), repo.Items{})
	stockv1.RegisterStockServiceServer(srv, grpcadapter.NewServer(items, log))
}

// StartEvents chạy relay và consumer stock.reserve như cmd/server của service kho.
func StartEvents(ctx context.Context, appPool, relayPool *pgxpool.Pool, js jetstream.JetStream, log *slog.Logger) (wait func()) {
	return app.StartEvents(ctx, app.EventDeps{
		AppPool: appPool, RelayPool: relayPool, JS: js, Log: log, RelayInterval: 50 * time.Millisecond,
	})
}

// ReserveProcessor trả processor stock.reserve để test gọi song song (AC-08).
func ReserveProcessor(appPool *pgxpool.Pool) *inbox.Processor {
	return app.ReserveProcessor(appPool)
}
