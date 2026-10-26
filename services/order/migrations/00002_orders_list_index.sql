-- Chỉ mục cho ListOrders (50 đơn mới nhất của tenant, xếp created_at giảm dần).
-- RLS thêm điều kiện tenant_id = app.tenant_id vào mọi truy vấn, nên chỉ mục bắt đầu
-- bằng tenant_id; thiếu chỉ mục này thì mỗi lần mở danh sách phải quét và sắp xếp
-- toàn bộ đơn. Thêm file mới vì migration đã merge thì không sửa.

-- +goose Up
CREATE INDEX orders_tenant_created_idx
  ON mini_order.orders (tenant_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX mini_order.orders_tenant_created_idx;
