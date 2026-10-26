-- Dữ liệu mẫu Q39 (docs/specs/business-rules.md, mục 4).
-- Bảng có FORCE ROW LEVEL SECURITY nên chủ bảng cũng phải đặt app.tenant_id
-- trước khi chèn; goose chạy file này trong một giao dịch nên set_config(..., true) đủ dùng.

-- +goose Up
SELECT set_config('app.tenant_id', '11111111-1111-1111-1111-111111111111', true);
INSERT INTO mini_stock.stock_items (tenant_id, sku, name, unit_price, currency_code, on_hand, reserved) VALUES
  ('11111111-1111-1111-1111-111111111111', 'BAN-01', 'Bàn làm việc', 1250000.00, 'VND', 10, 0),
  ('11111111-1111-1111-1111-111111111111', 'GHE-01', 'Ghế xoay',      850000.00, 'VND', 20, 0),
  ('11111111-1111-1111-1111-111111111111', 'DEN-01', 'Đèn bàn',       320000.00, 'VND',  1, 0);

SELECT set_config('app.tenant_id', '22222222-2222-2222-2222-222222222222', true);
INSERT INTO mini_stock.stock_items (tenant_id, sku, name, unit_price, currency_code, on_hand, reserved) VALUES
  ('22222222-2222-2222-2222-222222222222', 'BAN-01', 'Bàn họp',  2500000.00, 'VND', 5, 0),
  ('22222222-2222-2222-2222-222222222222', 'TU-01',  'Tủ hồ sơ', 1800000.00, 'VND', 3, 0);

-- +goose Down
SELECT set_config('app.tenant_id', '11111111-1111-1111-1111-111111111111', true);
DELETE FROM mini_stock.stock_items WHERE sku IN ('BAN-01', 'GHE-01', 'DEN-01');

SELECT set_config('app.tenant_id', '22222222-2222-2222-2222-222222222222', true);
DELETE FROM mini_stock.stock_items WHERE sku IN ('BAN-01', 'TU-01');
