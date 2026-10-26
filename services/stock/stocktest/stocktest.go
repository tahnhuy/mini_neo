// Package stocktest dựng service kho thật cho test tích hợp của service khác
// (ví dụ service đơn), vì luật "internal" của Go không cho chúng import
// services/stock/internal. Không dùng trong mã chạy thật.
package stocktest

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"github.com/tahnhuy/mini_neo/pkg/db"
	grpcadapter "github.com/tahnhuy/mini_neo/services/stock/internal/adapter/grpc"
	"github.com/tahnhuy/mini_neo/services/stock/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/stock/internal/usecase"
)

// Register gắn StockService thật (usecase, repo, RLS qua db.InTx) vào srv.
func Register(srv *grpc.Server, pool *pgxpool.Pool, log *slog.Logger) {
	items := usecase.NewItems(db.New(pool), repo.Items{})
	stockv1.RegisterStockServiceServer(srv, grpcadapter.NewServer(items, log))
}
