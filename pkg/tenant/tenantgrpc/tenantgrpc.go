// Package tenantgrpc chuyển danh tính giữa metadata gRPC và context
// (docs/specs/api-contract.md, mục 6).
//
// Tách khỏi pkg/tenant để mã không dùng gRPC không phải import grpc.
package tenantgrpc

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/tahnhuy/mini_neo/pkg/tenant"
)

// Tên khóa metadata, đúng như đề.
const (
	MDTenantID     = "x-tenant-id"
	MDUserID       = "x-user-id"
	MDCapabilities = "x-capabilities"
)

// UnaryServerInterceptor đọc metadata và đặt danh tính vào context.
// Thiếu hoặc sai x-tenant-id thì trả UNAUTHENTICATED.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		id, err := identityFromMetadata(ctx)
		if err != nil {
			return nil, err
		}
		return next(tenant.WithIdentity(ctx, id), req)
	}
}

func identityFromMetadata(ctx context.Context) (tenant.Identity, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	raw := first(md, MDTenantID)
	if raw == "" {
		return tenant.Identity{}, status.Error(codes.Unauthenticated, "thiếu x-tenant-id")
	}
	tenantID, err := uuid.Parse(raw)
	if err != nil || tenantID == uuid.Nil {
		return tenant.Identity{}, status.Error(codes.Unauthenticated, "x-tenant-id không hợp lệ")
	}
	return tenant.Identity{
		TenantID: tenantID,
		UserID:   first(md, MDUserID),
		Caps:     parseCapabilities(first(md, MDCapabilities)),
	}, nil
}

// OutgoingContext gắn danh tính trong ctx vào metadata gửi đi,
// dùng khi service đơn gọi service kho (Q8).
func OutgoingContext(ctx context.Context) context.Context {
	id, ok := tenant.FromContext(ctx)
	if !ok {
		return ctx
	}
	names := make([]string, 0, len(id.Caps))
	for n := range id.Caps {
		names = append(names, n)
	}
	return metadata.AppendToOutgoingContext(ctx,
		MDTenantID, id.TenantID.String(),
		MDUserID, id.UserID,
		MDCapabilities, strings.Join(names, ","),
	)
}

func parseCapabilities(raw string) tenant.Capabilities {
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return tenant.NewCapabilities(parts...)
}

func first(md metadata.MD, key string) string {
	if vals := md.Get(key); len(vals) > 0 {
		return strings.TrimSpace(vals[0])
	}
	return ""
}
