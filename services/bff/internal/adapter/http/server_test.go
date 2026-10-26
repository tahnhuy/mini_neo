package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orderv1 "github.com/tahnhuy/mini_neo/gen/go/mini/order/v1"
	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type orderStub struct {
	orderv1.OrderServiceClient
	call func(context.Context, *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error)
}

func (s orderStub) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest, _ ...grpc.CallOption) (*orderv1.CreateOrderResponse, error) {
	return s.call(ctx, req)
}
func TestCreate_AC01_ClientPriceIgnored(t *testing.T) {
	stub := orderStub{call: func(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
		md, _ := metadata.FromOutgoingContext(ctx)
		if md.Get("x-user-id")[0] != "an" || md.Get("x-tenant-id")[0] != "11111111-1111-1111-1111-111111111111" {
			t.Fatal(md)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("thiếu timeout")
		}
		if len(req.Lines) != 1 || req.Lines[0].Sku != "BAN-01" || req.Lines[0].Quantity != 2 {
			t.Fatal(req)
		}
		return &orderv1.CreateOrderResponse{Order: &orderv1.Order{TotalAmount: &orderv1.Money{Amount: "2500000.00", CurrencyCode: "VND"}}}, nil
	}}
	req := httptest.NewRequest("POST", "/api/orders", strings.NewReader(`{"totalAmount":0,"lines":[{"sku":"BAN-01","quantity":2,"unitPrice":1}]}`))
	req.Header.Set("X-Dev-User", "an")
	w := httptest.NewRecorder()
	New(stub, nil).ServeHTTP(w, req)
	body := compact(t, w.Body.Bytes())
	if w.Code != 200 || !strings.Contains(body, `"amount":"2500000.00"`) || !strings.Contains(body, `"allowedActions":[]`) {
		t.Fatal(w.Code, body)
	}
}

// compact đưa JSON về dạng ổn định: protojson cố ý chèn khoảng trắng ngẫu nhiên
// theo từng bản build, nên không so trực tiếp chuỗi output.
func compact(t *testing.T, data []byte) string {
	t.Helper()
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err, string(data))
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
func TestErrorMapping(t *testing.T) {
	tests := []struct {
		code codes.Code
		http int
		name string
	}{
		{codes.InvalidArgument, 400, "INVALID_ARGUMENT"}, {codes.Unauthenticated, 401, "UNAUTHENTICATED"},
		{codes.PermissionDenied, 403, "PERMISSION_DENIED"}, {codes.NotFound, 404, "NOT_FOUND"},
		{codes.FailedPrecondition, 409, "FAILED_PRECONDITION"}, {codes.Unavailable, 500, "UNAVAILABLE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeError(w, status.Error(tt.code, "Thông báo gốc"))
			var body map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != tt.http || body["code"] != tt.name || body["message"] != "Thông báo gốc" {
				t.Fatal(w.Code, body)
			}
		})
	}
}
func TestReasonDetails(t *testing.T) {
	st, err := status.New(codes.PermissionDenied, "Bạn không thể tự duyệt đơn do mình lập").WithDetails(&orderv1.ErrorDetail{Reason: "SELF_APPROVAL"})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	writeError(w, st.Err())
	if w.Code != 403 || !strings.Contains(w.Body.String(), "SELF_APPROVAL") {
		t.Fatal(w.Body.String())
	}
}
func TestIdentity(t *testing.T) {
	for _, name := range []string{"an", "binh", "chi", "dung", "la", ""} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/me", nil)
			req.Header.Set("X-Dev-User", name)
			w := httptest.NewRecorder()
			New(nil, nil).ServeHTTP(w, req)
			u, ok := lookup(name)
			if !ok {
				if w.Code != 401 {
					t.Fatal(w.Code)
				}
				return
			}
			var got user
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.UserID != u.UserID || got.TenantID != u.TenantID || len(got.Capabilities) != len(u.Capabilities) {
				t.Fatal(got)
			}
		})
	}
}

type allOrders struct{ orderv1.OrderServiceClient }

func (allOrders) ListOrders(context.Context, *orderv1.ListOrdersRequest, ...grpc.CallOption) (*orderv1.ListOrdersResponse, error) {
	return &orderv1.ListOrdersResponse{}, nil
}
func (allOrders) GetOrder(_ context.Context, r *orderv1.GetOrderRequest, _ ...grpc.CallOption) (*orderv1.GetOrderResponse, error) {
	return &orderv1.GetOrderResponse{Order: &orderv1.Order{Id: r.Id}}, nil
}
func (allOrders) ApproveOrder(_ context.Context, r *orderv1.ApproveOrderRequest, _ ...grpc.CallOption) (*orderv1.ApproveOrderResponse, error) {
	return &orderv1.ApproveOrderResponse{Order: &orderv1.Order{Id: r.Id}}, nil
}
func (allOrders) RejectOrder(_ context.Context, r *orderv1.RejectOrderRequest, _ ...grpc.CallOption) (*orderv1.RejectOrderResponse, error) {
	return &orderv1.RejectOrderResponse{Order: &orderv1.Order{Id: r.Id, RejectReason: r.Reason}}, nil
}

type allStock struct{ stockv1.StockServiceClient }

func (allStock) ListItems(context.Context, *stockv1.ListItemsRequest, ...grpc.CallOption) (*stockv1.ListItemsResponse, error) {
	return &stockv1.ListItemsResponse{}, nil
}
func TestRoutes(t *testing.T) {
	tests := []struct{ method, path, body, want string }{
		{"GET", "/api/orders", "", `"orders":[]`}, {"GET", "/api/orders/abc", "", `"id":"abc"`},
		{"POST", "/api/orders/abc/approve", "", `"id":"abc"`},
		{"POST", "/api/orders/abc/reject", `{"id":"spoof","reason":"Không đủ ngân sách"}`, `"id":"abc"`},
		{"GET", "/api/stock-items", "", `"items":[]`},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("X-Dev-User", "chi")
			w := httptest.NewRecorder()
			New(allOrders{}, allStock{}).ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatal(w.Code, w.Body.String())
			}
			if body := compact(t, w.Body.Bytes()); !strings.Contains(body, tt.want) {
				t.Fatal(body)
			}
		})
	}
}
