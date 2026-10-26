package domain

import "sort"

// ReservationHeld là trạng thái duy nhất của phiếu giữ hàng (Q3).
const ReservationHeld = "HELD"

// Level là tồn của một mặt hàng, đọc sau khi đã khóa dòng (FOR UPDATE).
type Level struct {
	SKU      string
	OnHand   int32
	Reserved int32
}

// Want là số lượng một dòng đơn cần giữ.
type Want struct {
	SKU      string
	Quantity int32
}

// Plan là kết quả kiểm giữ hàng cho cả đơn.
type Plan struct {
	OK        bool
	NewLevels []Level  // chỉ có khi OK: reserved mới của từng mặt hàng
	Lines     []Want   // các dòng sẽ ghi vào reservation_lines (đã gộp theo mã)
	ShortSKUs []string // chỉ có khi không OK: các mã thiếu, xếp theo mã
}

// PlanReservation áp luật tất cả hoặc không gì cả: mọi dòng đều có
// on_hand - reserved >= quantity thì cộng reserved cho tất cả; thiếu một dòng
// (kể cả mã không có trong kho) thì không cộng dòng nào.
func PlanReservation(levels []Level, wants []Want) Plan {
	byLevel := make(map[string]Level, len(levels))
	for _, l := range levels {
		byLevel[l.SKU] = l
	}
	// Cộng bằng int64: hai dòng trùng mã có tổng vượt int32 sẽ tràn thành số âm
	// và lọt qua phép so khả dụng bên dưới.
	need := map[string]int64{}
	for _, w := range wants {
		need[w.SKU] += int64(w.Quantity)
	}
	skus := make([]string, 0, len(need))
	for sku := range need {
		skus = append(skus, sku)
	}
	sort.Strings(skus)

	plan := Plan{OK: true}
	for _, sku := range skus {
		l, ok := byLevel[sku]
		if !ok || int64(l.OnHand)-int64(l.Reserved) < need[sku] {
			plan.OK = false
			plan.ShortSKUs = append(plan.ShortSKUs, sku)
			continue
		}
		// need ≤ on_hand - reserved nên đổi về int32 an toàn.
		qty := int32(need[sku])
		plan.NewLevels = append(plan.NewLevels, Level{SKU: sku, OnHand: l.OnHand, Reserved: l.Reserved + qty})
		plan.Lines = append(plan.Lines, Want{SKU: sku, Quantity: qty})
	}
	if !plan.OK {
		plan.NewLevels, plan.Lines = nil, nil
	}
	return plan
}
