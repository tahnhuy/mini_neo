-- Tiền đọc ghi dạng chuỗi ("1250000.00") rồi ép sang numeric trong SQL, để Go
-- không bao giờ cầm số thực. tenant_id lấy từ app.tenant_id của giao dịch;
-- RLS (WITH CHECK) chặn ghi sai tenant, nên không cần WHERE tenant_id.

-- Q7: tăng số đếm của tenant. INSERT ... ON CONFLICT DO UPDATE khóa dòng đếm
-- cho đến hết giao dịch, nên hai đơn song song không lấy trùng số; đơn đầu tiên
-- của tenant cũng không bị lỗi khóa chính như SELECT ... FOR UPDATE rồi INSERT.
-- name: NextOrderNumber :one
INSERT INTO mini_order.order_counters (tenant_id, last_number)
VALUES (current_setting('app.tenant_id')::uuid, 1)
ON CONFLICT (tenant_id) DO UPDATE SET last_number = mini_order.order_counters.last_number + 1
RETURNING last_number;

-- name: InsertOrder :exec
INSERT INTO mini_order.orders (id, tenant_id, code, status, created_by, total_amount, currency_code, created_at, updated_at)
VALUES (sqlc.arg(id), current_setting('app.tenant_id')::uuid, sqlc.arg(code), sqlc.arg(status), sqlc.arg(created_by),
        sqlc.arg(total_amount)::text::numeric, sqlc.arg(currency_code), sqlc.arg(created_at), sqlc.arg(created_at));

-- name: InsertOrderLine :exec
INSERT INTO mini_order.order_lines (id, tenant_id, order_id, sku, quantity, unit_price, line_amount, currency_code)
VALUES (sqlc.arg(id), current_setting('app.tenant_id')::uuid, sqlc.arg(order_id), sqlc.arg(sku), sqlc.arg(quantity),
        sqlc.arg(unit_price)::text::numeric, sqlc.arg(line_amount)::text::numeric, sqlc.arg(currency_code));

-- name: GetOrder :one
SELECT id, code, status, created_by,
       coalesce(decided_by, '')::text AS decided_by, coalesce(reject_reason, '')::text AS reject_reason,
       total_amount::text AS total_amount, currency_code::text AS currency_code, created_at, updated_at
FROM mini_order.orders
WHERE id = sqlc.arg(id);

-- Khóa dòng đơn khi duyệt/từ chối để hai quyết định cùng lúc không ghi đè nhau.
-- name: GetOrderForUpdate :one
SELECT id, code, status, created_by,
       coalesce(decided_by, '')::text AS decided_by, coalesce(reject_reason, '')::text AS reject_reason,
       total_amount::text AS total_amount, currency_code::text AS currency_code, created_at, updated_at
FROM mini_order.orders
WHERE id = sqlc.arg(id)
FOR UPDATE;

-- name: ListOrders :many
SELECT id, code, status, created_by,
       coalesce(decided_by, '')::text AS decided_by, coalesce(reject_reason, '')::text AS reject_reason,
       total_amount::text AS total_amount, currency_code::text AS currency_code, created_at, updated_at
FROM mini_order.orders
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(max_rows);

-- name: ListOrderLines :many
SELECT sku, quantity, unit_price::text AS unit_price, line_amount::text AS line_amount, currency_code::text AS currency_code
FROM mini_order.order_lines
WHERE order_id = sqlc.arg(order_id)
ORDER BY sku;

-- name: UpdateOrderDecision :exec
UPDATE mini_order.orders
SET status        = sqlc.arg(status),
    decided_by    = NULLIF(sqlc.arg(decided_by)::text, ''),
    reject_reason = NULLIF(sqlc.arg(reject_reason)::text, ''),
    updated_at    = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
