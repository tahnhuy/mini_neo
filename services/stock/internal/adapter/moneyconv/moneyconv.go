// Package moneyconv đổi tiền giữa chuỗi thập phân ("1250000.00", dạng của
// CSDL, proto và JSON) và int64 xu của domain. Không đi qua số thực.
package moneyconv

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var decimalRe = regexp.MustCompile(`^-?\d+(\.\d{1,2})?$`)

// ToCents đổi "1250000.00" thành 125000000 xu.
func ToCents(s string) (int64, error) {
	if !decimalRe.MatchString(s) {
		return 0, fmt.Errorf("moneyconv: số tiền không hợp lệ %q", s)
	}
	neg := strings.HasPrefix(s, "-")
	whole, frac, _ := strings.Cut(strings.TrimPrefix(s, "-"), ".")
	frac = (frac + "00")[:2]
	cents, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("moneyconv: số tiền vượt giới hạn %q: %w", s, err)
	}
	if neg {
		cents = -cents
	}
	return cents, nil
}

// FromCents đổi 125000000 xu thành "1250000.00".
func FromCents(cents int64) string {
	sign := ""
	u := uint64(cents)
	if cents < 0 {
		sign = "-"
		u = uint64(-cents)
	}
	return fmt.Sprintf("%s%d.%02d", sign, u/100, u%100)
}
