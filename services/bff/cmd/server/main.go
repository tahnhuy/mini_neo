// BFF chạy HTTP và chuyển tiếp tới hai service qua gRPC.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	orderv1 "github.com/tahnhuy/mini_neo/gen/go/mini/order/v1"
	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	httpadapter "github.com/tahnhuy/mini_neo/services/bff/internal/adapter/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := run(); err != nil {
		slog.Error("BFF dừng", "err", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func run() error {
	order, err := grpc.NewClient(env("ORDER_GRPC_ADDR", "127.0.0.1:50051"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("kết nối đơn: %w", err)
	}
	defer order.Close()
	stock, err := grpc.NewClient(env("STOCK_GRPC_ADDR", "127.0.0.1:50052"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("kết nối kho: %w", err)
	}
	defer stock.Close()
	srv := &http.Server{Addr: env("BFF_HTTP_ADDR", "127.0.0.1:8080"), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second,
		Handler: httpadapter.New(orderv1.NewOrderServiceClient(order), stockv1.NewStockServiceClient(stock))}
	return serve(srv)
}

func serve(srv *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- srv.ListenAndServe() }()
	slog.Info("BFF đang chạy", "addr", srv.Addr)
	select {
	case err := <-done:
		return fmt.Errorf("HTTP: %w", err)
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			_ = srv.Close()
			return fmt.Errorf("dừng HTTP: %w", err)
		}
		err := <-done
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
