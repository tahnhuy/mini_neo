package tenantgrpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/tahnhuy/mini_neo/pkg/tenant"
)

func TestUnaryServerInterceptor(t *testing.T) {
	tests := []struct {
		name     string
		md       metadata.MD
		wantCode codes.Code
		wantCaps []string
	}{
		{"thiếu x-tenant-id", metadata.Pairs(MDUserID, "an"), codes.Unauthenticated, nil},
		{"x-tenant-id sai", metadata.Pairs(MDTenantID, "khong-phai-uuid"), codes.Unauthenticated, nil},
		{
			"đủ metadata",
			metadata.Pairs(MDTenantID, "11111111-1111-1111-1111-111111111111",
				MDUserID, "binh", MDCapabilities, "order.create, order.approve"),
			codes.OK,
			[]string{"order.create", "order.approve"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got tenant.Identity
			handler := func(ctx context.Context, _ any) (any, error) {
				got, _ = tenant.FromContext(ctx)
				return nil, nil
			}
			ctx := metadata.NewIncomingContext(context.Background(), tt.md)
			_, err := UnaryServerInterceptor()(ctx, nil, &grpc.UnaryServerInfo{}, handler)
			if code := status.Code(err); code != tt.wantCode {
				t.Fatalf("code = %v, want %v", code, tt.wantCode)
			}
			for _, c := range tt.wantCaps {
				if !got.Caps.Has(c) {
					t.Errorf("thiếu quyền %q", c)
				}
			}
		})
	}
}

func TestOutgoingContextRoundTrip(t *testing.T) {
	in := metadata.Pairs(MDTenantID, "22222222-2222-2222-2222-222222222222",
		MDUserID, "dung", MDCapabilities, "order.create")
	id, err := identityFromMetadata(metadata.NewIncomingContext(context.Background(), in))
	if err != nil {
		t.Fatal(err)
	}
	out := OutgoingContext(tenant.WithIdentity(context.Background(), id))
	md, _ := metadata.FromOutgoingContext(out)
	back, err := identityFromMetadata(metadata.NewIncomingContext(context.Background(), md))
	if err != nil {
		t.Fatal(err)
	}
	if back.TenantID != id.TenantID || back.UserID != "dung" || !back.Caps.Has("order.create") {
		t.Errorf("danh tính sau khi truyền lại = %+v", back)
	}
}
