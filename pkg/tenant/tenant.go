// Package tenant giữ danh tính người gọi (tenant, người dùng, quyền) trong context.
//
// Đây là cách làm tạm của bài: danh tính đến từ metadata gRPC do BFF gắn
// (docs/specs/api-contract.md, mục 6), không có đăng nhập thật.
package tenant

import (
	"context"

	"github.com/google/uuid"
)

// Capabilities là tập quyền theo tên, ví dụ "order.approve".
// Kiểm quyền luôn theo tên quyền, không theo tên người hay vai trò.
type Capabilities map[string]struct{}

// NewCapabilities tạo tập quyền từ danh sách tên; tên rỗng bị bỏ qua.
func NewCapabilities(names ...string) Capabilities {
	caps := make(Capabilities, len(names))
	for _, n := range names {
		if n != "" {
			caps[n] = struct{}{}
		}
	}
	return caps
}

// Has báo người gọi có quyền name hay không.
func (c Capabilities) Has(name string) bool {
	_, ok := c[name]
	return ok
}

// Identity là danh tính của một lời gọi.
type Identity struct {
	TenantID uuid.UUID
	UserID   string
	Caps     Capabilities
}

type ctxKey struct{}

// WithIdentity gắn danh tính vào context.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext đọc danh tính; ok = false khi context chưa có danh tính
// hoặc tenant rỗng.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	if !ok || id.TenantID == uuid.Nil {
		return Identity{}, false
	}
	return id, true
}
