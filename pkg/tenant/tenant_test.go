package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCapabilitiesHas(t *testing.T) {
	caps := NewCapabilities("order.create", "", "order.approve")
	tests := []struct {
		name string
		want bool
	}{
		{"order.create", true},
		{"order.approve", true},
		{"stock.adjust", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := caps.Has(tt.name); got != tt.want {
			t.Errorf("Has(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestFromContext(t *testing.T) {
	tenantA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tests := []struct {
		name   string
		ctx    context.Context
		wantOK bool
	}{
		{"không có danh tính", context.Background(), false},
		{"tenant rỗng", WithIdentity(context.Background(), Identity{UserID: "an"}), false},
		{"đủ danh tính", WithIdentity(context.Background(), Identity{TenantID: tenantA, UserID: "an"}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := FromContext(tt.ctx)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && id.TenantID != tenantA {
				t.Errorf("TenantID = %v, want %v", id.TenantID, tenantA)
			}
		})
	}
}
