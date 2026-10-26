package domain

import (
	"math"
	"reflect"
	"testing"
)

func TestPlanReservation(t *testing.T) {
	levels := []Level{{"BAN-01", 10, 2}, {"DEN-01", 1, 0}, {"GHE-01", 20, 20}}
	tests := []struct {
		name      string
		wants     []Want
		wantOK    bool
		wantLevel map[string]int32
		wantShort []string
	}{
		{"đủ hàng mọi dòng", []Want{{"BAN-01", 8}, {"DEN-01", 1}}, true, map[string]int32{"BAN-01": 10, "DEN-01": 1}, nil},
		{"thiếu một dòng thì không giữ dòng nào", []Want{{"BAN-01", 1}, {"GHE-01", 1}}, false, nil, []string{"GHE-01"}},
		{"mã không có trong kho", []Want{{"TU-01", 1}, {"BAN-01", 1}}, false, nil, []string{"TU-01"}},
		{"vượt khả dụng 1 đơn vị", []Want{{"BAN-01", 9}}, false, nil, []string{"BAN-01"}},
		{"dòng trùng mã được gộp trước khi kiểm", []Want{{"DEN-01", 1}, {"DEN-01", 1}}, false, nil, []string{"DEN-01"}},
		{"tổng dòng trùng mã vượt int32 không tràn số", []Want{{"BAN-01", math.MaxInt32}, {"BAN-01", math.MaxInt32}}, false, nil, []string{"BAN-01"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PlanReservation(levels, tt.wants)
			if p.OK != tt.wantOK || !reflect.DeepEqual(p.ShortSKUs, tt.wantShort) {
				t.Fatalf("plan = %+v", p)
			}
			if !tt.wantOK && (p.NewLevels != nil || p.Lines != nil) {
				t.Fatalf("không OK thì không được có mức tồn mới: %+v", p)
			}
			for _, l := range p.NewLevels {
				if l.Reserved != tt.wantLevel[l.SKU] || l.Reserved > l.OnHand {
					t.Errorf("%s reserved = %d, want %d (≤ on_hand %d)", l.SKU, l.Reserved, tt.wantLevel[l.SKU], l.OnHand)
				}
			}
		})
	}
}
