// Package stockclient gọi service kho để lấy đơn giá lúc tạo đơn.
package stockclient

import (
	"context"
	"fmt"
	"time"

	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"github.com/tahnhuy/mini_neo/pkg/moneyconv"
	"github.com/tahnhuy/mini_neo/pkg/tenant/tenantgrpc"
	"github.com/tahnhuy/mini_neo/services/order/internal/domain"
)

// Timeout cho mỗi lời gọi sang service kho.
const Timeout = 3 * time.Second

// Client cài đặt usecase.StockClient.
type Client struct {
	stock stockv1.StockServiceClient
}

// New tạo client từ client gRPC sinh sẵn.
func New(stock stockv1.StockServiceClient) *Client {
	return &Client{stock: stock}
}

// GetPrices gọi StockService.GetItems, truyền lại metadata tenant của người gọi (Q8).
func (c *Client) GetPrices(ctx context.Context, skus []string) (map[string]domain.Money, error) {
	ctx, cancel := context.WithTimeout(tenantgrpc.OutgoingContext(ctx), Timeout)
	defer cancel()
	resp, err := c.stock.GetItems(ctx, &stockv1.GetItemsRequest{Skus: skus})
	if err != nil {
		return nil, fmt.Errorf("stockclient: GetItems: %w", err)
	}
	prices := make(map[string]domain.Money, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		cents, err := moneyconv.ToCents(it.GetUnitPrice().GetAmount())
		if err != nil {
			return nil, fmt.Errorf("stockclient: giá của %s: %w", it.GetSku(), err)
		}
		prices[it.GetSku()] = domain.Money{Cents: cents, Currency: it.GetUnitPrice().GetCurrencyCode()}
	}
	return prices, nil
}
