package domain

import "testing"

func TestItemAvailable(t *testing.T) {
	tests := []struct {
		name     string
		item     Item
		expected int32
	}{
		{"chưa giữ", Item{OnHand: 10, Reserved: 0}, 10},
		{"đã giữ một phần", Item{OnHand: 10, Reserved: 4}, 6},
		{"giữ hết", Item{OnHand: 1, Reserved: 1}, 0},
	}
	for _, tt := range tests {
		if got := tt.item.Available(); got != tt.expected {
			t.Errorf("%s: Available() = %d, want %d", tt.name, got, tt.expected)
		}
	}
}
