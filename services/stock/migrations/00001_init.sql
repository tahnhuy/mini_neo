-- Schema mini_stock (docs/specs/data-model.md, mục 3, 4, 5).
-- Chạy bằng mini_owner (chủ bảng). Migration đã merge thì không sửa; muốn đổi thì thêm file mới.

-- +goose Up
CREATE SCHEMA mini_stock;

CREATE TABLE mini_stock.stock_items (
  tenant_id     uuid NOT NULL,
  sku           text NOT NULL,
  name          text NOT NULL,
  unit_price    numeric(18,2) NOT NULL,
  currency_code char(3) NOT NULL,
  on_hand       integer NOT NULL,
  reserved      integer NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, sku),
  -- Lưới an toàn cuối; việc kiểm tồn làm trong Go sau khi khóa dòng.
  CHECK (reserved >= 0 AND on_hand >= reserved)
);

CREATE TABLE mini_stock.reservations (
  id         uuid PRIMARY KEY,
  tenant_id  uuid NOT NULL,
  order_id   uuid NOT NULL,
  status     text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, order_id)
);

-- Q6: khóa chính (tenant_id, reservation_id, sku).
CREATE TABLE mini_stock.reservation_lines (
  tenant_id      uuid NOT NULL,
  reservation_id uuid NOT NULL REFERENCES mini_stock.reservations (id),
  sku            text NOT NULL,
  quantity       integer NOT NULL CHECK (quantity > 0),
  PRIMARY KEY (tenant_id, reservation_id, sku)
);

CREATE TABLE mini_stock.outbox (
  id           uuid PRIMARY KEY,          -- = id của sự kiện (UUIDv7)
  tenant_id    uuid NOT NULL,
  type         text NOT NULL,             -- tên sự kiện, cũng là subject NATS
  payload      jsonb NOT NULL,            -- cả phong bì sự kiện
  created_at   timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz
);

CREATE TABLE mini_stock.inbox (
  tenant_id    uuid NOT NULL,
  consumer     text NOT NULL,             -- ví dụ 'stock.reserve'
  event_id     uuid NOT NULL,
  processed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, consumer, event_id)
);

-- RLS: FORCE để áp cả với chủ bảng; chưa đặt app.tenant_id thì current_setting báo lỗi.
ALTER TABLE mini_stock.stock_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.stock_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.stock_items
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_stock.reservations ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.reservations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.reservations
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_stock.reservation_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.reservation_lines FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.reservation_lines
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_stock.outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.outbox
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_stock.inbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.inbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.inbox
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Quyền: mini_app làm việc nghiệp vụ; mini_relay chỉ chạm outbox.
GRANT USAGE ON SCHEMA mini_stock TO mini_app, mini_relay;
GRANT SELECT, INSERT, UPDATE ON
  mini_stock.stock_items, mini_stock.reservations, mini_stock.reservation_lines,
  mini_stock.outbox, mini_stock.inbox
  TO mini_app;
GRANT SELECT, UPDATE ON mini_stock.outbox TO mini_relay;

-- +goose Down
DROP SCHEMA mini_stock CASCADE;
