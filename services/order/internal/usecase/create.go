package usecase

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

// NewLine là một dòng client gửi: chỉ mã hàng và số lượng, không có giá.
type NewLine struct {
	SKU      string
	Quantity int32
}

// CreateOrder tạo đơn: đơn giá lấy từ service kho, tổng tiền do domain tính,
// mã đơn sinh qua bảng đếm của tenant trong cùng giao dịch (Q7).
func (u *Orders) CreateOrder(ctx context.Context, lines []NewLine) (View, error) {
	id, err := identity(ctx)
	if err != nil {
		return View{}, err
	}
	if err := requireAny(id, CapCreate); err != nil {
		return View{}, err
	}
	inputs, err := u.priceLines(ctx, lines)
	if err != nil {
		return View{}, err
	}
	orderID, err := u.d.NewID()
	if err != nil {
		return View{}, fmt.Errorf("create order: sinh id: %w", err)
	}
	o, err := domain.NewOrder(orderID, id.UserID, inputs, u.d.Now())
	if err != nil {
		return View{}, err
	}
	err = u.d.Tx.InTx(ctx, func(tx pgx.Tx) error {
		n, err := u.d.Repo.NextNumber(ctx, tx)
		if err != nil {
			return err
		}
		o.Code = domain.FormatCode(n)
		return u.d.Repo.Insert(ctx, tx, o)
	})
	if err != nil {
		return View{}, fmt.Errorf("create order: %w", err)
	}
	return view(o, id), nil
}

// priceLines gắn đơn giá từ kho vào từng dòng; mã hàng không có thì NOT_FOUND (Q27).
// Gọi kho trước khi mở giao dịch để không giữ giao dịch trong lúc gọi mạng.
func (u *Orders) priceLines(ctx context.Context, lines []NewLine) ([]domain.LineInput, error) {
	skus := make([]string, 0, len(lines))
	for _, l := range lines {
		skus = append(skus, l.SKU)
	}
	prices, err := u.d.Stock.GetPrices(ctx, skus)
	if err != nil {
		return nil, fmt.Errorf("create order: lấy giá từ kho: %w", err)
	}
	inputs := make([]domain.LineInput, 0, len(lines))
	for _, l := range lines {
		price, ok := prices[l.SKU]
		if !ok {
			return nil, domain.NewError(domain.KindNotFound, fmt.Sprintf("Không tìm thấy mặt hàng %s", l.SKU))
		}
		inputs = append(inputs, domain.LineInput{SKU: l.SKU, Quantity: l.Quantity, UnitPrice: price})
	}
	return inputs, nil
}
