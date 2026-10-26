// Package grpc nhận request gRPC của service kho, gọi usecase và đổi lỗi
// sang mã gRPC. Không chứa quy tắc nghiệp vụ.
package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"github.com/tahnhuy/mini_neo/services/stock/internal/adapter/moneyconv"
	"github.com/tahnhuy/mini_neo/services/stock/internal/domain"
	"github.com/tahnhuy/mini_neo/services/stock/internal/usecase"
)

// ItemLister là phần usecase mà adapter cần.
type ItemLister interface {
	ListItems(ctx context.Context) ([]domain.Item, error)
}

// Server cài đặt mini.stock.v1.StockService. GetItems làm ở ngày 2.
type Server struct {
	stockv1.UnimplementedStockServiceServer
	items ItemLister
	log   *slog.Logger
}

// NewServer tạo server gRPC của service kho.
func NewServer(items ItemLister, log *slog.Logger) *Server {
	return &Server{items: items, log: log}
}

// ListItems trả mặt hàng của tenant, kèm available do domain tính.
func (s *Server) ListItems(ctx context.Context, _ *stockv1.ListItemsRequest) (*stockv1.ListItemsResponse, error) {
	items, err := s.items.ListItems(ctx)
	if err != nil {
		return nil, s.toStatus(ctx, err)
	}
	resp := &stockv1.ListItemsResponse{Items: make([]*stockv1.Item, 0, len(items))}
	for _, it := range items {
		resp.Items = append(resp.Items, &stockv1.Item{
			Sku:       it.SKU,
			Name:      it.Name,
			UnitPrice: &stockv1.Money{Amount: moneyconv.FromCents(it.UnitPrice.Cents), CurrencyCode: it.UnitPrice.Currency},
			OnHand:    it.OnHand,
			Reserved:  it.Reserved,
			Available: it.Available(),
		})
	}
	return resp, nil
}

func (s *Server) toStatus(ctx context.Context, err error) error {
	if errors.Is(err, usecase.ErrUnauthenticated) {
		return status.Error(codes.Unauthenticated, "thiếu danh tính")
	}
	s.log.ErrorContext(ctx, "lỗi không lường trước", "err", err)
	return status.Error(codes.Internal, "lỗi hệ thống")
}
