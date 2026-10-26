//go:build integration

// Mục tiêu ngày 1: service kho trả ListItems qua gRPC, có RLS thật và dữ liệu mẫu Q39.
package grpc_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"github.com/tahnhuy/mini_neo/internal/testdb"
	"github.com/tahnhuy/mini_neo/pkg/db"
	"github.com/tahnhuy/mini_neo/pkg/tenant/tenantgrpc"
	grpcadapter "github.com/tahnhuy/mini_neo/services/stock/internal/adapter/grpc"
	"github.com/tahnhuy/mini_neo/services/stock/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/stock/internal/usecase"
)

func newClient(t *testing.T) stockv1.StockServiceClient {
	t.Helper()
	pool := testdb.NewAppPool(t, testdb.Stock)
	srv := grpc.NewServer(grpc.UnaryInterceptor(tenantgrpc.UnaryServerInterceptor()))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	stockv1.RegisterStockServiceServer(srv, grpcadapter.NewServer(usecase.NewItems(db.New(pool), repo.Items{}), log))

	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return stockv1.NewStockServiceClient(conn)
}

func asTenant(tenantID string) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), tenantgrpc.MDTenantID, tenantID)
}

func TestListItems(t *testing.T) {
	client := newClient(t)

	t.Run("tenant A thấy 3 mặt hàng, DEN-01 còn 1", func(t *testing.T) {
		resp, err := client.ListItems(asTenant(testdb.TenantA), &stockv1.ListItemsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Items) != 3 {
			t.Fatalf("có %d mặt hàng, want 3", len(resp.Items))
		}
		got := map[string]*stockv1.Item{}
		for _, it := range resp.Items {
			got[it.Sku] = it
		}
		if den := got["DEN-01"]; den == nil || den.Available != 1 {
			t.Errorf("DEN-01 = %+v, want available 1", den)
		}
		if ban := got["BAN-01"]; ban == nil || ban.UnitPrice.Amount != "1250000.00" || ban.UnitPrice.CurrencyCode != "VND" {
			t.Errorf("BAN-01 = %+v, want 1250000.00 VND", ban)
		}
	})
	t.Run("tenant B chỉ thấy 2 mặt hàng của mình", func(t *testing.T) {
		resp, err := client.ListItems(asTenant(testdb.TenantB), &stockv1.ListItemsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Items) != 2 {
			t.Fatalf("có %d mặt hàng, want 2", len(resp.Items))
		}
	})
	t.Run("thiếu x-tenant-id thì UNAUTHENTICATED", func(t *testing.T) {
		_, err := client.ListItems(context.Background(), &stockv1.ListItemsRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})
}
