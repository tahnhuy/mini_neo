// Package domain chứa mô hình nghiệp vụ của service kho.
// Không import pgx, grpc, nats.
package domain

// Money là số tiền tính bằng xu (1 đồng = 100 xu), không dùng số thực.
type Money struct {
	Cents    int64
	Currency string
}

// Item là một mặt hàng trong kho của một tenant.
type Item struct {
	SKU       string
	Name      string
	UnitPrice Money
	OnHand    int32
	Reserved  int32
}

// Available = tồn thực tế − đang giữ (docs/specs/business-rules.md, mục 5).
func (i Item) Available() int32 {
	return i.OnHand - i.Reserved
}
