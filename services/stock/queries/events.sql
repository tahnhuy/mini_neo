-- Truy vấn cho sự kiện của service kho: outbox, inbox, giữ hàng.

-- Ghi phong bì; tenant_id lấy từ app.tenant_id, RLS (WITH CHECK) chặn ghi sai tenant.
-- name: InsertOutbox :exec
INSERT INTO mini_stock.outbox (id, tenant_id, type, payload)
VALUES (sqlc.arg(id), current_setting('app.tenant_id')::uuid, sqlc.arg(type), sqlc.arg(payload));

-- Relay (vai trò mini_relay, BYPASSRLS): hai relay chạy cùng lúc không lấy trùng dòng.
-- name: ClaimOutbox :many
SELECT id, type, payload FROM mini_stock.outbox
WHERE published_at IS NULL
ORDER BY id
LIMIT sqlc.arg(max_rows)
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE mini_stock.outbox SET published_at = now() WHERE id = ANY(sqlc.arg(ids)::uuid[]);

-- Không chèn được dòng nào (0 dòng) nghĩa là sự kiện đã xử lý rồi.
-- name: ClaimInbox :execrows
INSERT INTO mini_stock.inbox (tenant_id, consumer, event_id)
VALUES (current_setting('app.tenant_id')::uuid, sqlc.arg(consumer), sqlc.arg(event_id))
ON CONFLICT DO NOTHING;

-- Khóa dòng TRƯỚC khi đọc tồn, theo thứ tự sku để hai giao dịch không chờ nhau vòng tròn.
-- name: LockStockLevels :many
SELECT sku, on_hand, reserved
FROM mini_stock.stock_items
WHERE sku = ANY(sqlc.arg(skus)::text[])
ORDER BY sku
FOR UPDATE;

-- Ghi giá trị reserved do Go tính sau khi kiểm. Ghi số tuyệt đối (không phải
-- reserved + n) để nếu ai bỏ FOR UPDATE thì test AC-08 bắt được ghi đè mất dữ liệu.
-- name: SetReserved :exec
UPDATE mini_stock.stock_items SET reserved = sqlc.arg(reserved) WHERE sku = sqlc.arg(sku);

-- name: ReservationExists :one
SELECT EXISTS (SELECT 1 FROM mini_stock.reservations WHERE order_id = sqlc.arg(order_id));

-- name: InsertReservation :exec
INSERT INTO mini_stock.reservations (id, tenant_id, order_id, status)
VALUES (sqlc.arg(id), current_setting('app.tenant_id')::uuid, sqlc.arg(order_id), sqlc.arg(status));

-- name: InsertReservationLine :exec
INSERT INTO mini_stock.reservation_lines (tenant_id, reservation_id, sku, quantity)
VALUES (current_setting('app.tenant_id')::uuid, sqlc.arg(reservation_id), sqlc.arg(sku), sqlc.arg(quantity));
