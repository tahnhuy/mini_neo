package moneyconv

import "testing"

func TestToCents(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"1250000.00", 125000000, false},
		{"0.1", 10, false},
		{"0.20", 20, false},
		{"320000", 32000000, false},
		{"-5.05", -505, false},
		{"1.234", 0, true},
		{"1e5", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		got, err := ToCents(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ToCents(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ToCents(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// 0.1 + 0.2 phải đúng bằng 0.3 khi tính bằng xu, khác với số thực.
func TestCentsAddExactly(t *testing.T) {
	a, _ := ToCents("0.10")
	b, _ := ToCents("0.20")
	if got := FromCents(a + b); got != "0.30" {
		t.Errorf("0.10 + 0.20 = %s, want 0.30", got)
	}
}

func TestFromCents(t *testing.T) {
	tests := map[int64]string{125000000: "1250000.00", 5: "0.05", 0: "0.00", -505: "-5.05"}
	for in, want := range tests {
		if got := FromCents(in); got != want {
			t.Errorf("FromCents(%d) = %q, want %q", in, got, want)
		}
	}
}
