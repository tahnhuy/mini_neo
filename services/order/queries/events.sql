-- Truy vấn cho sự kiện của service đơn: outbox, inbox.

-- Ghi phong bì; tenant_id lấy từ app.tenant_id, RLS (WITH CHECK) chặn ghi sai tenant.
-- name: InsertOutbox :exec
INSERT INTO mini_order.outbox (id, tenant_id, type, payload)
VALUES (sqlc.arg(id), current_setting('app.tenant_id')::uuid, sqlc.arg(type), sqlc.arg(payload));

-- Relay (vai trò mini_relay, BYPASSRLS): hai relay chạy cùng lúc không lấy trùng dòng.
-- name: ClaimOutbox :many
SELECT id, type, payload FROM mini_order.outbox
WHERE published_at IS NULL
ORDER BY id
LIMIT sqlc.arg(max_rows)
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE mini_order.outbox SET published_at = now() WHERE id = ANY(sqlc.arg(ids)::uuid[]);

-- Không chèn được dòng nào (0 dòng) nghĩa là sự kiện đã xử lý rồi.
-- name: ClaimInbox :execrows
INSERT INTO mini_order.inbox (tenant_id, consumer, event_id)
VALUES (current_setting('app.tenant_id')::uuid, sqlc.arg(consumer), sqlc.arg(event_id))
ON CONFLICT DO NOTHING;
