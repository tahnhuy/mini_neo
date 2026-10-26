package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderv1 "github.com/tahnhuy/mini_neo/gen/go/mini/order/v1"
	"github.com/tahnhuy/mini_neo/pkg/moneyconv"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
	"github.com/tahnhuy/mini_neo/services/order/internal/usecase"
)

var kindToCode = map[domain.Kind]codes.Code{
	domain.KindInvalidArgument:    codes.InvalidArgument,
	domain.KindUnauthenticated:    codes.Unauthenticated,
	domain.KindPermissionDenied:   codes.PermissionDenied,
	domain.KindNotFound:           codes.NotFound,
	domain.KindFailedPrecondition: codes.FailedPrecondition,
}

// ErrorMapper đổi lỗi của usecase sang status gRPC.
type ErrorMapper struct {
	log *slog.Logger
}

// NewErrorMapper tạo bộ đổi lỗi; lỗi không lường trước được log rồi trả INTERNAL.
func NewErrorMapper(log *slog.Logger) ErrorMapper {
	return ErrorMapper{log: log}
}

// toStatus: lỗi nghiệp vụ giữ message tiếng Việt (Q32); lỗi có mã lý do thì
// đính ErrorDetail{reason} vào chi tiết lỗi (Q29) để BFF trả "code" cho web.
func (m ErrorMapper) toStatus(ctx context.Context, err error) error {
	var de *domain.Error
	if !errors.As(err, &de) {
		return m.infraStatus(ctx, err)
	}
	code, ok := kindToCode[de.Kind]
	if !ok {
		code = codes.Internal
	}
	st := status.New(code, de.Message)
	if de.Reason == "" {
		return st.Err()
	}
	withDetail, derr := st.WithDetails(&orderv1.ErrorDetail{Reason: de.Reason})
	if derr != nil {
		m.log.ErrorContext(ctx, "không đính được ErrorDetail", "err", derr)
		return st.Err()
	}
	return withDetail.Err()
}

// infraStatus đổi lỗi hạ tầng. Client hủy thì không log lỗi; service kho quá hạn
// (timeout 3 giây của stockclient) hay không chạy thì trả UNAVAILABLE để bên gọi
// biết là thử lại được, thay vì INTERNAL.
func (m ErrorMapper) infraStatus(ctx context.Context, err error) error {
	// Lỗi do chính client hủy hoặc hết hạn: không phải lỗi của service.
	if ctx.Err() != nil {
		return status.FromContextError(ctx.Err()).Err()
	}
	// status.FromError đọc được cả status gRPC đã bị bọc bằng %w.
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded:
			m.log.WarnContext(ctx, "service phụ thuộc không phản hồi", "err", err)
			return status.Error(codes.Unavailable, "Hệ thống đang bận, vui lòng thử lại")
		}
	}
	m.log.ErrorContext(ctx, "lỗi không lường trước", "err", err)
	return status.Error(codes.Internal, "Lỗi hệ thống, vui lòng thử lại")
}

func money(m domain.Money) *orderv1.Money {
	return &orderv1.Money{Amount: moneyconv.FromCents(m.Cents), CurrencyCode: m.Currency}
}

func toProto(v usecase.View) *orderv1.Order {
	o := v.Order
	out := &orderv1.Order{
		Id: o.ID.String(), Code: o.Code, Status: string(o.Status), CreatedBy: o.CreatedBy,
		DecidedBy: o.DecidedBy, RejectReason: o.RejectReason, TotalAmount: money(o.Total),
		CreatedAt: timestamppb.New(o.CreatedAt), UpdatedAt: timestamppb.New(o.UpdatedAt),
		AllowedActions: v.AllowedActions,
	}
	for _, l := range o.Lines {
		out.Lines = append(out.Lines, &orderv1.OrderLine{
			Sku: l.SKU, Quantity: l.Quantity, UnitPrice: money(l.UnitPrice), LineAmount: money(l.LineAmount),
		})
	}
	return out
}
