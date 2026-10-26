// Service đơn: đọc cấu hình từ biến môi trường, nối dây, chạy server gRPC,
// relay outbox và consumer kết quả giữ hàng.
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
	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/pkg/tenant/tenantgrpc"
	grpcadapter "github.com/tahnhuy/mini_neo/services/order/internal/adapter/grpc"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/stockclient"
	"github.com/tahnhuy/mini_neo/services/order/internal/app"
	"github.com/tahnhuy/mini_neo/services/order/internal/usecase"
)

type config struct {
	databaseURL      string
	relayDatabaseURL string
	grpcAddr         string
	stockTarget      string
	natsURL          string
}

func loadConfig() (config, error) {
	cfg := config{
		databaseURL:      os.Getenv("ORDER_DATABASE_URL"),
		relayDatabaseURL: os.Getenv("ORDER_RELAY_DATABASE_URL"),
		grpcAddr:         envOr("ORDER_GRPC_ADDR", "127.0.0.1:50051"),
		stockTarget:      envOr("STOCK_GRPC_ADDR", "127.0.0.1:50052"),
		natsURL:          envOr("NATS_URL", "nats://127.0.0.1:4222"),
	}
	if cfg.databaseURL == "" || cfg.relayDatabaseURL == "" {
		return cfg, errors.New("thiếu biến môi trường ORDER_DATABASE_URL hoặc ORDER_RELAY_DATABASE_URL")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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

	appPool, relayPool, err := openPools(ctx, cfg)
	if err != nil {
		return err
	}
	defer appPool.Close()
	defer relayPool.Close()

	nc, js, err := outbox.Connect(ctx, cfg.natsURL)
	if err != nil {
		return err
	}
	defer nc.Close()

	// Kết nối nội bộ trên máy, chưa có TLS: cách làm tạm của bài (api-contract.md, mục 6).
	stockConn, err := grpc.NewClient(cfg.stockTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("kết nối service kho: %w", err)
	}
	defer stockConn.Close()

	wait := app.StartEvents(ctx, app.EventDeps{AppPool: appPool, RelayPool: relayPool, JS: js, Log: log})
	defer wait() // chờ relay và consumer dừng trước khi đóng pool và NATS
	return serve(ctx, log, newServer(appPool, stockConn, log), cfg.grpcAddr)
}

func newServer(pool *pgxpool.Pool, stockConn *grpc.ClientConn, log *slog.Logger) *grpc.Server {
	orders := usecase.NewOrders(usecase.Deps{
		Tx:     db.New(pool),
		Repo:   repo.Orders{},
		Stock:  stockclient.New(stockv1.NewStockServiceClient(stockConn)),
		Outbox: repo.Outbox{},
	})
	srv := grpc.NewServer(grpc.UnaryInterceptor(tenantgrpc.UnaryServerInterceptor()))
	orderv1.RegisterOrderServiceServer(srv, grpcadapter.NewServer(orders, grpcadapter.NewErrorMapper(log)))
	return srv
}

// openPools mở pool của mini_app (nghiệp vụ) và mini_relay (relay outbox).
func openPools(ctx context.Context, cfg config) (*pgxpool.Pool, *pgxpool.Pool, error) {
	appPool, err := openPool(ctx, cfg.databaseURL)
	if err != nil {
		return nil, nil, err
	}
	relayPool, err := openPool(ctx, cfg.relayDatabaseURL)
	if err != nil {
		appPool.Close()
		return nil, nil, err
	}
	return appPool, relayPool, nil
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
