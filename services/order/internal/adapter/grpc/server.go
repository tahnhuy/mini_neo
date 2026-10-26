// Package grpc nhận request gRPC của service đơn: kiểm đầu vào (Q28), gọi usecase,
// đổi lỗi sang mã gRPC kèm ErrorDetail (Q29). Không chứa quy tắc nghiệp vụ.
package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/tahnhuy/mini_neo/gen/go/mini/order/v1"
	"github.com/tahnhuy/mini_neo/services/order/internal/usecase"
)

// OrderUsecase là phần usecase mà adapter cần.
type OrderUsecase interface {
	CreateOrder(ctx context.Context, lines []usecase.NewLine) (usecase.View, error)
	GetOrder(ctx context.Context, id uuid.UUID) (usecase.View, error)
	ListOrders(ctx context.Context) ([]usecase.View, error)
	ApproveOrder(ctx context.Context, id uuid.UUID) (usecase.View, error)
	RejectOrder(ctx context.Context, id uuid.UUID, reason string) (usecase.View, error)
}

// Server cài đặt mini.order.v1.OrderService.
type Server struct {
	orderv1.UnimplementedOrderServiceServer
	orders OrderUsecase
	errs   ErrorMapper
}

// NewServer tạo server gRPC của service đơn.
func NewServer(orders OrderUsecase, errs ErrorMapper) *Server {
	return &Server{orders: orders, errs: errs}
}

// CreateOrder kiểm hình dạng đầu vào rồi gọi usecase; request không có trường giá.
func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	lines, err := validateLines(req.GetLines())
	if err != nil {
		return nil, err
	}
	v, err := s.orders.CreateOrder(ctx, lines)
	if err != nil {
		return nil, s.errs.toStatus(ctx, err)
	}
	return &orderv1.CreateOrderResponse{Order: toProto(v)}, nil
}

// GetOrder trả đơn kèm allowed_actions.
func (s *Server) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	v, err := s.orders.GetOrder(ctx, id)
	if err != nil {
		return nil, s.errs.toStatus(ctx, err)
	}
	return &orderv1.GetOrderResponse{Order: toProto(v)}, nil
}

// ListOrders trả tối đa 50 đơn mới nhất.
func (s *Server) ListOrders(ctx context.Context, _ *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	views, err := s.orders.ListOrders(ctx)
	if err != nil {
		return nil, s.errs.toStatus(ctx, err)
	}
	resp := &orderv1.ListOrdersResponse{Orders: make([]*orderv1.Order, 0, len(views))}
	for _, v := range views {
		resp.Orders = append(resp.Orders, toProto(v))
	}
	return resp, nil
}

// ApproveOrder duyệt đơn.
func (s *Server) ApproveOrder(ctx context.Context, req *orderv1.ApproveOrderRequest) (*orderv1.ApproveOrderResponse, error) {
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	v, err := s.orders.ApproveOrder(ctx, id)
	if err != nil {
		return nil, s.errs.toStatus(ctx, err)
	}
	return &orderv1.ApproveOrderResponse{Order: toProto(v)}, nil
}

// RejectOrder từ chối đơn; độ dài lý do do domain kiểm.
func (s *Server) RejectOrder(ctx context.Context, req *orderv1.RejectOrderRequest) (*orderv1.RejectOrderResponse, error) {
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	v, err := s.orders.RejectOrder(ctx, id, req.GetReason())
	if err != nil {
		return nil, s.errs.toStatus(ctx, err)
	}
	return &orderv1.RejectOrderResponse{Order: toProto(v)}, nil
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "Mã đơn không hợp lệ")
	}
	return id, nil
}

// validateLines kiểm hình dạng đầu vào (Q28): có ít nhất một dòng, mã hàng
// không rỗng và không lặp, số lượng > 0.
func validateLines(in []*orderv1.CreateOrderLine) ([]usecase.NewLine, error) {
	if len(in) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Đơn cần ít nhất một dòng")
	}
	seen := make(map[string]bool, len(in))
	lines := make([]usecase.NewLine, 0, len(in))
	for _, l := range in {
		switch {
		case l.GetSku() == "":
			return nil, status.Error(codes.InvalidArgument, "Mã hàng không được để trống")
		case l.GetQuantity() <= 0:
			return nil, status.Errorf(codes.InvalidArgument, "Số lượng của %s phải lớn hơn 0", l.GetSku())
		case seen[l.GetSku()]:
			return nil, status.Errorf(codes.InvalidArgument, "Mã hàng %s bị lặp trong đơn", l.GetSku())
		}
		seen[l.GetSku()] = true
		lines = append(lines, usecase.NewLine{SKU: l.GetSku(), Quantity: l.GetQuantity()})
	}
	return lines, nil
}
