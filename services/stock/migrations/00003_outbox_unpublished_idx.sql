-- Chỉ mục cho vòng lặp relay (ClaimOutbox: published_at IS NULL, ORDER BY id, LIMIT).
-- Chỉ có khóa chính thì mỗi 500 ms relay phải đi qua mọi dòng đã gửi trước khi gặp
-- dòng chưa gửi, nên càng chạy lâu càng chậm. Chỉ mục một phần chỉ chứa dòng chưa
-- gửi, nên luôn nhỏ. Thêm file mới vì migration đã merge thì không sửa.

-- +goose Up
CREATE INDEX outbox_unpublished_idx
  ON mini_stock.outbox (id) WHERE published_at IS NULL;

-- +goose Down
DROP INDEX mini_stock.outbox_unpublished_idx;
