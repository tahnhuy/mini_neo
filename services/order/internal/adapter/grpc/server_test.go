package grpc

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/tahnhuy/mini_neo/gen/go/mini/order/v1"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

func TestValidateLines(t *testing.T) {
	line := func(sku string, q int32) *orderv1.CreateOrderLine {
		return &orderv1.CreateOrderLine{Sku: sku, Quantity: q}
	}
	tests := []struct {
		name string
		in   []*orderv1.CreateOrderLine
		ok   bool
	}{
		{"rỗng", nil, false},
		{"mã rỗng", []*orderv1.CreateOrderLine{line("", 1)}, false},
		{"số lượng 0", []*orderv1.CreateOrderLine{line("BAN-01", 0)}, false},
		{"mã lặp", []*orderv1.CreateOrderLine{line("BAN-01", 1), line("BAN-01", 2)}, false},
		{"hợp lệ", []*orderv1.CreateOrderLine{line("BAN-01", 1), line("GHE-01", 2)}, true},
	}
	for _, tt := range tests {
		_, err := validateLines(tt.in)
		if tt.ok != (err == nil) {
			t.Errorf("%s: err = %v", tt.name, err)
		}
		if !tt.ok && status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: code = %v, want InvalidArgument", tt.name, status.Code(err))
		}
	}
}

// Lỗi SELF_APPROVAL phải tới BFF dưới dạng PERMISSION_DENIED kèm ErrorDetail.
func TestToStatusAttachesReason(t *testing.T) {
	m := NewErrorMapper(slog.New(slog.NewTextHandler(io.Discard, nil)))
	err := m.toStatus(context.Background(), &domain.Error{
		Kind: domain.KindPermissionDenied, Reason: domain.ReasonSelfApproval, Message: "Bạn không thể tự duyệt đơn do mình lập",
	})
	st := status.Convert(err)
	if st.Code() != codes.PermissionDenied || st.Message() != "Bạn không thể tự duyệt đơn do mình lập" {
		t.Fatalf("status = %v %q", st.Code(), st.Message())
	}
	details := st.Details()
	if len(details) != 1 {
		t.Fatalf("details = %v, want 1 ErrorDetail", details)
	}
	if d, ok := details[0].(*orderv1.ErrorDetail); !ok || d.GetReason() != domain.ReasonSelfApproval {
		t.Fatalf("detail = %#v", details[0])
	}
}

// Kho quá hạn hoặc không chạy: UNAVAILABLE (thử lại được), không phải INTERNAL.
// Client tự hủy: trả mã của context, không log như lỗi hệ thống.
func TestToStatusInfraErrors(t *testing.T) {
	m := NewErrorMapper(slog.New(slog.NewTextHandler(io.Discard, nil)))
	stockTimeout := fmt.Errorf("stockclient: GetItems: %w", status.Error(codes.DeadlineExceeded, "context deadline exceeded"))
	if got := status.Code(m.toStatus(context.Background(), stockTimeout)); got != codes.Unavailable {
		t.Errorf("kho quá hạn: code = %v, want Unavailable", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := status.Code(m.toStatus(ctx, fmt.Errorf("db: begin: %w", context.Canceled))); got != codes.Canceled {
		t.Errorf("client hủy: code = %v, want Canceled", got)
	}
}

func TestToStatusHidesUnexpectedError(t *testing.T) {
	m := NewErrorMapper(slog.New(slog.NewTextHandler(io.Discard, nil)))
	st := status.Convert(m.toStatus(context.Background(), io.ErrUnexpectedEOF))
	if st.Code() != codes.Internal || len(st.Details()) != 0 {
		t.Fatalf("status = %v, details = %v", st.Code(), st.Details())
	}
}
