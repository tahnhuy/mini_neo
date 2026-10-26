// Service đơn: đọc cấu hình từ biến môi trường, nối dây, chạy server gRPC.
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
	"google.golang.org/grpc/credentials/insecure"

	orderv1 "github.com/tahnhuy/mini_neo/gen/go/mini/order/v1"
	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"github.com/tahnhuy/mini_neo/pkg/db"
	"github.com/tahnhuy/mini_neo/pkg/tenant/tenantgrpc"
	grpcadapter "github.com/tahnhuy/mini_neo/services/order/internal/adapter/grpc"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/stockclient"
	"github.com/tahnhuy/mini_neo/services/order/internal/usecase"
)

type config struct {
	databaseURL string
	grpcAddr    string
	stockTarget string
}

func loadConfig() (config, error) {
	cfg := config{
		databaseURL: os.Getenv("ORDER_DATABASE_URL"),
		grpcAddr:    os.Getenv("ORDER_GRPC_ADDR"),
		stockTarget: os.Getenv("STOCK_GRPC_ADDR"),
	}
	if cfg.databaseURL == "" {
		return cfg, errors.New("thiếu biến môi trường ORDER_DATABASE_URL")
	}
	if cfg.grpcAddr == "" {
		cfg.grpcAddr = "127.0.0.1:50051"
	}
	if cfg.stockTarget == "" {
		cfg.stockTarget = "127.0.0.1:50052"
	}
	return cfg, nil
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("service đơn dừng vì lỗi", "err", err)
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

	// Kết nối nội bộ trên máy, chưa có TLS: cách làm tạm của bài (api-contract.md, mục 6).
	stockConn, err := grpc.NewClient(cfg.stockTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("kết nối service kho: %w", err)
	}
	defer stockConn.Close()

	orders := usecase.NewOrders(usecase.Deps{
		Tx:    db.New(pool),
		Repo:  repo.Orders{},
		Stock: stockclient.New(stockv1.NewStockServiceClient(stockConn)),
	})
	srv := grpc.NewServer(grpc.UnaryInterceptor(tenantgrpc.UnaryServerInterceptor()))
	orderv1.RegisterOrderServiceServer(srv, grpcadapter.NewServer(orders, grpcadapter.NewErrorMapper(log)))
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
	log.Info("service đơn đang chạy", "addr", addr)

	select {
	case <-ctx.Done():
		log.Info("đang dừng service đơn")
		srv.GracefulStop()
		return nil
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	}
}
