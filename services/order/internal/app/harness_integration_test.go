//go:build integration

// Bộ dựng cho test tích hợp ngày 3: hai service, một CSDL test (Q18), NATS riêng
// cho test (Q36). Relay chạy mỗi 50 ms để test nhanh; mã thật dùng 500 ms.
package app_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"github.com/tahnhuy/mini_neo/internal/testdb"
	"github.com/tahnhuy/mini_neo/pkg/db"
	"github.com/tahnhuy/mini_neo/pkg/tenant"
	"github.com/tahnhuy/mini_neo/pkg/tenant/tenantgrpc"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/stockclient"
	"github.com/tahnhuy/mini_neo/services/order/internal/app"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
	"github.com/tahnhuy/mini_neo/services/order/internal/usecase"
	"github.com/tahnhuy/mini_neo/services/stock/stocktest"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type harness struct {
	pools  testdb.Pools
	js     jetstream.JetStream
	orders *usecase.Orders
}

// newHarness dựng CSDL, service kho (gRPC), usecase đơn. withEvents = true thì
// chạy relay và consumer của cả hai service như cmd/server.
func newHarness(t *testing.T, withEvents bool) *harness {
	t.Helper()
	h := &harness{pools: testdb.NewPools(t, testdb.Order, testdb.Stock)}
	stock := stockv1.NewStockServiceClient(serveStock(t, h))
	h.orders = usecase.NewOrders(usecase.Deps{
		Tx: db.New(h.pools.App), Repo: repo.Orders{}, Outbox: repo.Outbox{}, Stock: stockclient.New(stock),
	})
	if withEvents {
		h.js = testdb.NewJetStream(t)
		ctx, cancel := context.WithCancel(context.Background())
		waitStock := stocktest.StartEvents(ctx, h.pools.App, h.pools.Relay, h.js, quiet)
		waitOrder := app.StartEvents(ctx, app.EventDeps{AppPool: h.pools.App, RelayPool: h.pools.Relay,
			JS: h.js, Log: quiet, RelayInterval: 50 * time.Millisecond})
		t.Cleanup(func() { cancel(); waitStock(); waitOrder() })
	}
	return h
}

func serveStock(t *testing.T, h *harness) *grpc.ClientConn {
	t.Helper()
	srv := grpc.NewServer(grpc.UnaryInterceptor(tenantgrpc.UnaryServerInterceptor()))
	stocktest.Register(srv, h.pools.App, quiet)
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
	return conn
}

// Người dùng mẫu của tenant A (docs/specs/business-rules.md, mục 4).
var caps = map[string][]string{"an": {usecase.CapCreate}, "chi": {usecase.CapApprove}}

func as(user string) context.Context {
	return tenant.WithIdentity(context.Background(), tenant.Identity{
		TenantID: uuid.MustParse(testdb.TenantA), UserID: user, Caps: tenant.NewCapabilities(caps[user]...),
	})
}

// createAndApprove: an tạo đơn, chi duyệt. Trả id đơn.
func (h *harness) createAndApprove(t *testing.T, lines ...usecase.NewLine) uuid.UUID {
	t.Helper()
	v, err := h.orders.CreateOrder(as("an"), lines)
	if err != nil {
		t.Fatalf("an tạo đơn: %v", err)
	}
	if _, err := h.orders.ApproveOrder(as("chi"), v.Order.ID); err != nil {
		t.Fatalf("chi duyệt: %v", err)
	}
	return v.Order.ID
}

// waitDecided chờ đơn rời APPROVED (kho đã trả lời) rồi trả trạng thái cuối.
func (h *harness) waitDecided(t *testing.T, id uuid.UUID) domain.Status {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		v, err := h.orders.GetOrder(as("chi"), id)
		if err != nil {
			t.Fatal(err)
		}
		if v.Order.Status != domain.StatusApproved {
			return v.Order.Status
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("đơn %s vẫn APPROVED sau 15 giây", id)
	return ""
}

// queryInt chạy một truy vấn đếm trong giao dịch của tenant A, bằng mini_app.
func (h *harness) queryInt(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	err := db.New(h.pools.App).InTx(as("chi"), func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), sql, args...).Scan(&n)
	})
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func (h *harness) reserved(t *testing.T, sku string) int {
	t.Helper()
	return h.queryInt(t, "SELECT reserved FROM mini_stock.stock_items WHERE sku = $1", sku)
}

// payloads đọc payload outbox theo loại sự kiện, xếp theo id.
func (h *harness) payloads(t *testing.T, table, typ string) [][]byte {
	t.Helper()
	var out [][]byte
	err := db.New(h.pools.App).InTx(as("chi"), func(tx pgx.Tx) error {
		rows, err := tx.Query(context.Background(), "SELECT payload FROM "+table+" WHERE type = $1 ORDER BY id", typ)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, pgx.RowTo[[]byte])
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
