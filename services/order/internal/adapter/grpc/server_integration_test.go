//go:build integration

// Test tích hợp ngày 2: service đơn và service kho thật qua gRPC (bufconn),
// PostgreSQL thật với RLS, chạy bằng mini_app.
package grpc_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	orderv1 "github.com/tahnhuy/mini_neo/gen/go/mini/order/v1"
	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"github.com/tahnhuy/mini_neo/internal/testdb"
	"github.com/tahnhuy/mini_neo/pkg/db"
	"github.com/tahnhuy/mini_neo/pkg/tenant/tenantgrpc"
	ordergrpc "github.com/tahnhuy/mini_neo/services/order/internal/adapter/grpc"
	orderrepo "github.com/tahnhuy/mini_neo/services/order/internal/adapter/repo"
	"github.com/tahnhuy/mini_neo/services/order/internal/adapter/stockclient"
	orderuc "github.com/tahnhuy/mini_neo/services/order/internal/usecase"
	"github.com/tahnhuy/mini_neo/services/stock/stocktest"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// Người dùng mẫu: tenant và quyền (docs/specs/business-rules.md, mục 4).
var users = map[string][2]string{
	"an":   {testdb.TenantA, "order.create"},
	"binh": {testdb.TenantA, "order.create,order.approve"},
	"chi":  {testdb.TenantA, "order.approve"},
	"dung": {testdb.TenantB, "order.create,order.approve"},
}

// as tạo context mang metadata như BFF gắn cho người dùng user.
func as(user string) context.Context {
	u := users[user]
	return metadata.AppendToOutgoingContext(context.Background(),
		tenantgrpc.MDTenantID, u[0], tenantgrpc.MDUserID, user, tenantgrpc.MDCapabilities, u[1])
}

func serveBufconn(t *testing.T, register func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()
	srv := grpc.NewServer(grpc.UnaryInterceptor(tenantgrpc.UnaryServerInterceptor()))
	register(srv)
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

// newOrderClient dựng cả service kho và service đơn trên cùng một CSDL test (Q18).
func newOrderClient(t *testing.T) orderv1.OrderServiceClient {
	t.Helper()
	pool := testdb.NewAppPool(t, testdb.Order, testdb.Stock)
	stockConn := serveBufconn(t, func(s *grpc.Server) {
		stocktest.Register(s, pool, quiet)
	})
	orderConn := serveBufconn(t, func(s *grpc.Server) {
		orders := orderuc.NewOrders(orderuc.Deps{
			Tx: db.New(pool), Repo: orderrepo.Orders{},
			Stock: stockclient.New(stockv1.NewStockServiceClient(stockConn)),
		})
		orderv1.RegisterOrderServiceServer(s, ordergrpc.NewServer(orders, ordergrpc.NewErrorMapper(quiet)))
	})
	return orderv1.NewOrderServiceClient(orderConn)
}

func createOrder(t *testing.T, c orderv1.OrderServiceClient, user string) *orderv1.Order {
	t.Helper()
	resp, err := c.CreateOrder(as(user), &orderv1.CreateOrderRequest{Lines: []*orderv1.CreateOrderLine{
		{Sku: "BAN-01", Quantity: 2}, {Sku: "GHE-01", Quantity: 3},
	}})
	if err != nil {
		t.Fatalf("%s tạo đơn: %v", user, err)
	}
	return resp.GetOrder()
}

// AC-01 qua gRPC thật: giá lấy từ bảng stock_items, tổng đúng đến đồng.
func TestCreateOrder_AC01_PricesFromStockService(t *testing.T) {
	c := newOrderClient(t)
	o := createOrder(t, c, "an")
	if o.GetTotalAmount().GetAmount() != "5050000.00" || o.GetTotalAmount().GetCurrencyCode() != "VND" {
		t.Errorf("total = %v, want 5050000.00 VND", o.GetTotalAmount())
	}
	got := map[string]string{}
	for _, l := range o.GetLines() {
		got[l.GetSku()] = l.GetUnitPrice().GetAmount() + "/" + l.GetLineAmount().GetAmount()
	}
	if got["BAN-01"] != "1250000.00/2500000.00" || got["GHE-01"] != "850000.00/2550000.00" {
		t.Errorf("dòng đơn = %v", got)
	}
	if o.GetCode() != "DH-000001" || o.GetStatus() != "PENDING_APPROVAL" {
		t.Errorf("đơn = %s %s", o.GetCode(), o.GetStatus())
	}
	_, err := c.CreateOrder(as("an"), &orderv1.CreateOrderRequest{Lines: []*orderv1.CreateOrderLine{{Sku: "TU-01", Quantity: 1}}})
	if status.Code(err) != codes.NotFound {
		t.Errorf("TU-01 thuộc tenant B, tenant A tạo đơn phải NOT_FOUND; got %v", err)
	}
}

// AC-04 mức API (Q2): dung (tenant B) gọi GetOrder đúng id đơn của tenant A → NOT_FOUND.
func TestGetOrder_AC04_OtherTenantNotFound(t *testing.T) {
	c := newOrderClient(t)
	o := createOrder(t, c, "an")
	if _, err := c.GetOrder(as("an"), &orderv1.GetOrderRequest{Id: o.GetId()}); err != nil {
		t.Fatalf("đối chứng: an phải xem được đơn của mình: %v", err)
	}
	_, err := c.GetOrder(as("dung"), &orderv1.GetOrderRequest{Id: o.GetId()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("dung xem đơn tenant A: code = %v, want NotFound", status.Code(err))
	}
	list, err := c.ListOrders(as("dung"), &orderv1.ListOrdersRequest{})
	if err != nil || len(list.GetOrders()) != 0 {
		t.Fatalf("danh sách của dung = %v, %v; want rỗng", list.GetOrders(), err)
	}
}

// AC-03 qua gRPC: mã lý do SELF_APPROVAL nằm trong ErrorDetail để BFF đọc.
func TestApproveSelfApprovalDetail(t *testing.T) {
	c := newOrderClient(t)
	o := createOrder(t, c, "binh")
	_, err := c.ApproveOrder(as("binh"), &orderv1.ApproveOrderRequest{Id: o.GetId()})
	st := status.Convert(err)
	if st.Code() != codes.PermissionDenied || len(st.Details()) != 1 {
		t.Fatalf("status = %v, details = %v", st.Code(), st.Details())
	}
	if d, _ := st.Details()[0].(*orderv1.ErrorDetail); d.GetReason() != "SELF_APPROVAL" {
		t.Fatalf("reason = %v, want SELF_APPROVAL", st.Details()[0])
	}
	got, _ := c.GetOrder(as("chi"), &orderv1.GetOrderRequest{Id: o.GetId()})
	if got.GetOrder().GetStatus() != "PENDING_APPROVAL" || len(got.GetOrder().GetAllowedActions()) != 2 {
		t.Fatalf("sau khi binh tự duyệt: %v", got.GetOrder())
	}
	resp, err := c.ApproveOrder(as("chi"), &orderv1.ApproveOrderRequest{Id: o.GetId()})
	if err != nil || resp.GetOrder().GetStatus() != "APPROVED" || resp.GetOrder().GetDecidedBy() != "chi" {
		t.Fatalf("chi duyệt: %v, %v", resp.GetOrder(), err)
	}
}

// Q7: nhiều đơn tạo song song vẫn nhận mã khác nhau, liền nhau từ DH-000001.
func TestCreateOrderCodesUniqueUnderConcurrency(t *testing.T) {
	c := newOrderClient(t)
	const n = 10
	codesCh := make(chan string, n)
	errCh := make(chan error, n)
	var wg sync.WaitGroup
	req := &orderv1.CreateOrderRequest{Lines: []*orderv1.CreateOrderLine{{Sku: "BAN-01", Quantity: 1}}}
	for range n {
		wg.Add(1)
		// t.Fatal không được gọi từ goroutine khác; gửi lỗi về goroutine của test.
		go func() {
			defer wg.Done()
			resp, err := c.CreateOrder(as("an"), req)
			if err != nil {
				errCh <- err
				return
			}
			codesCh <- resp.GetOrder().GetCode()
		}()
	}
	wg.Wait()
	close(codesCh)
	close(errCh)
	for err := range errCh {
		t.Errorf("tạo đơn song song: %v", err)
	}
	seen := map[string]bool{}
	for code := range codesCh {
		seen[code] = true
	}
	for i := 1; i <= n; i++ {
		if code := fmt.Sprintf("DH-%06d", i); !seen[code] {
			t.Errorf("thiếu mã %s; có %v", code, keys(seen))
		}
	}
}

func keys(m map[string]bool) string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return strings.Join(out, ",")
}
