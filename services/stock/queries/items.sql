-- Tiền đọc ra dạng chuỗi ("1250000.00") để adapter đổi sang int64 xu,
-- không bao giờ qua số thực. Không cần WHERE tenant_id: RLS đã lọc.

-- name: ListItems :many
SELECT sku, name, unit_price::text AS unit_price, currency_code::text AS currency_code, on_hand, reserved
FROM mini_stock.stock_items
ORDER BY sku;
