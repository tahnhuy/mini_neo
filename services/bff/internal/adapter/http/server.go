// Package httpadapter chuyển REST sang gRPC, không tính nghiệp vụ.
package httpadapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	orderv1 "github.com/tahnhuy/mini_neo/gen/go/mini/order/v1"
	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	order orderv1.OrderServiceClient
	stock stockv1.StockServiceClient
}

type user struct {
	UserID       string   `json:"userId"`
	TenantID     string   `json:"tenantId"`
	Capabilities []string `json:"capabilities"`
}

const (
	tenantA = "11111111-1111-1111-1111-111111111111"
	tenantB = "22222222-2222-2222-2222-222222222222"
)

// users là bảng người dùng mẫu (business-rules.md, mục 4); khai một lần, chỉ đọc.
var users = map[string]user{
	"an":   {"an", tenantA, []string{"order.create"}},
	"binh": {"binh", tenantA, []string{"order.create", "order.approve"}},
	"chi":  {"chi", tenantA, []string{"order.approve"}},
	"dung": {"dung", tenantB, []string{"order.create", "order.approve"}},
}

func lookup(name string) (user, bool) {
	u, ok := users[name]
	return u, ok
}

func New(order orderv1.OrderServiceClient, stock stockv1.StockServiceClient) http.Handler {
	s := &Server{order: order, stock: stock}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("GET /api/orders", s.list)
	mux.HandleFunc("POST /api/orders", s.create)
	mux.HandleFunc("GET /api/orders/{id}", s.get)
	mux.HandleFunc("POST /api/orders/{id}/approve", s.approve)
	mux.HandleFunc("POST /api/orders/{id}/reject", s.reject)
	mux.HandleFunc("GET /api/stock-items", s.items)
	return authenticate(mux)
}

func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := lookup(r.Header.Get("X-Dev-User"))
		if !ok {
			writeError(w, status.Error(codes.Unauthenticated, "Người dùng không hợp lệ"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-user-id", u.UserID,
			"x-tenant-id", u.TenantID, "x-capabilities", strings.Join(u.Capabilities, ",")))
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, _ := lookup(r.Header.Get("X-Dev-User"))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(u)
}

func respond(w http.ResponseWriter, msg proto.Message, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	data, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(msg)
	if err != nil {
		writeError(w, status.Error(codes.Internal, "Không thể mã hóa phản hồi"))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(data)
}

func decode(w http.ResponseWriter, r *http.Request, msg proto.Message) error {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		return status.Error(codes.InvalidArgument, "Không đọc được nội dung yêu cầu")
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, msg); err != nil {
		return status.Error(codes.InvalidArgument, "Nội dung JSON không hợp lệ")
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	st := status.Convert(err)
	code := grpcName(st.Code())
	for _, detail := range st.Details() {
		if d, ok := detail.(*orderv1.ErrorDetail); ok && d.Reason != "" {
			code = d.Reason
		}
	}
	httpCode, ok := httpStatus[st.Code()]
	if !ok {
		// Mã gRPC đề không ánh xạ: trả 500 (api-contract.md, mục 4, gợi ý).
		httpCode = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": st.Message()})
}

// httpStatus là bảng ánh xạ của api-contract.md, mục 4.
var httpStatus = map[codes.Code]int{
	codes.InvalidArgument:    http.StatusBadRequest,
	codes.Unauthenticated:    http.StatusUnauthorized,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.NotFound:           http.StatusNotFound,
	codes.FailedPrecondition: http.StatusConflict,
}

// grpcNames theo tên chuẩn của gRPC (UPPER_SNAKE), dùng cho `code` khi không có mã lý do (Q14).
var grpcNames = map[codes.Code]string{
	codes.Canceled: "CANCELLED", codes.Unknown: "UNKNOWN", codes.InvalidArgument: "INVALID_ARGUMENT",
	codes.DeadlineExceeded: "DEADLINE_EXCEEDED", codes.NotFound: "NOT_FOUND", codes.AlreadyExists: "ALREADY_EXISTS",
	codes.PermissionDenied: "PERMISSION_DENIED", codes.ResourceExhausted: "RESOURCE_EXHAUSTED",
	codes.FailedPrecondition: "FAILED_PRECONDITION", codes.Aborted: "ABORTED", codes.OutOfRange: "OUT_OF_RANGE",
	codes.Unimplemented: "UNIMPLEMENTED", codes.Internal: "INTERNAL", codes.Unavailable: "UNAVAILABLE",
	codes.DataLoss: "DATA_LOSS", codes.Unauthenticated: "UNAUTHENTICATED",
}

func grpcName(code codes.Code) string {
	if name, ok := grpcNames[code]; ok {
		return name
	}
	return "UNKNOWN"
}
