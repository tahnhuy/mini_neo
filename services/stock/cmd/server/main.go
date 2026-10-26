// Service kho: đọc cấu hình từ biến môi trường, nối dây, chạy server gRPC.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"github.com/tahnhuy/mini_neo/pkg/db"
	"github.com/tahnhuy/mini_neo/pkg/tenant/tenantgrpc"
	grpcadapter "github.com/tahnhuy/mini_neo/services/stock/internal/adapter/grpc"
	"github.com/tahnhuy/mini_neo/services/stock/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/stock/internal/usecase"
)

type config struct {
	databaseURL string
	grpcAddr    string
}

func loadConfig() (config, error) {
	cfg := config{
		databaseURL: os.Getenv("STOCK_DATABASE_URL"),
		grpcAddr:    os.Getenv("STOCK_GRPC_ADDR"),
	}
	if cfg.databaseURL == "" {
		return cfg, errors.New("thiếu biến môi trường STOCK_DATABASE_URL")
	}
	if cfg.grpcAddr == "" {
		cfg.grpcAddr = "127.0.0.1:50052"
	}
	return cfg, nil
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("service kho dừng vì lỗi", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := openPool(ctx, cfg.databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	items := usecase.NewItems(db.New(pool), repo.Items{})
	srv := grpc.NewServer(grpc.UnaryInterceptor(tenantgrpc.UnaryServerInterceptor()))
	stockv1.RegisterStockServiceServer(srv, grpcadapter.NewServer(items, log))
	return serve(ctx, log, srv, cfg.grpcAddr)
}

func openPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("mở pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping CSDL: %w", err)
	}
	return pool, nil
}

// serve chạy server tới khi ctx bị hủy (Ctrl+C, SIGTERM) rồi dừng êm.
func serve(ctx context.Context, log *slog.Logger, srv *grpc.Server, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(lis) }()
	log.Info("service kho đang chạy", "addr", addr)

	select {
	case <-ctx.Done():
		log.Info("đang dừng service kho")
		srv.GracefulStop()
		return nil
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	}
}
