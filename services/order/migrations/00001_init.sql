-- Schema mini_order (docs/specs/data-model.md, mục 2, 4, 5).
-- Chạy bằng mini_owner (chủ bảng). Migration đã merge thì không sửa; muốn đổi thì thêm file mới.

-- +goose Up
CREATE SCHEMA mini_order;

CREATE TABLE mini_order.orders (
  id            uuid PRIMARY KEY,
  tenant_id     uuid NOT NULL,
  code          text NOT NULL,
  status        text NOT NULL,
  created_by    text NOT NULL,
  decided_by    text,
  reject_reason text,
  total_amount  numeric(18,2) NOT NULL,
  currency_code char(3) NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, code)
);

CREATE TABLE mini_order.order_lines (
  id            uuid PRIMARY KEY,
  tenant_id     uuid NOT NULL,
  order_id      uuid NOT NULL REFERENCES mini_order.orders (id),
  sku           text NOT NULL,
  quantity      integer NOT NULL CHECK (quantity > 0),
  unit_price    numeric(18,2) NOT NULL,
  line_amount   numeric(18,2) NOT NULL,
  currency_code char(3) NOT NULL,
  UNIQUE (tenant_id, order_id, sku)
);

-- Q7: bảng đếm để sinh mã DH-000001 theo tenant, khóa dòng trong giao dịch tạo đơn.
CREATE TABLE mini_order.order_counters (
  tenant_id   uuid PRIMARY KEY,
  last_number bigint NOT NULL
);

CREATE TABLE mini_order.outbox (
  id           uuid PRIMARY KEY,          -- = id của sự kiện (UUIDv7)
  tenant_id    uuid NOT NULL,
  type         text NOT NULL,             -- tên sự kiện, cũng là subject NATS
  payload      jsonb NOT NULL,            -- cả phong bì sự kiện
  created_at   timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz
);

CREATE TABLE mini_order.inbox (
  tenant_id    uuid NOT NULL,
  consumer     text NOT NULL,             -- ví dụ 'order.reservation_held'
  event_id     uuid NOT NULL,
  processed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, consumer, event_id)
);

-- RLS: FORCE để áp cả với chủ bảng; chưa đặt app.tenant_id thì current_setting báo lỗi.
ALTER TABLE mini_order.orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_order.orders FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_order.orders
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_order.order_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_order.order_lines FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_order.order_lines
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_order.order_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_order.order_counters FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_order.order_counters
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_order.outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_order.outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_order.outbox
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_order.inbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_order.inbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_order.inbox
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Quyền: mini_app làm việc nghiệp vụ; mini_relay chỉ chạm outbox.
GRANT USAGE ON SCHEMA mini_order TO mini_app, mini_relay;
GRANT SELECT, INSERT, UPDATE ON
  mini_order.orders, mini_order.order_lines, mini_order.order_counters,
  mini_order.outbox, mini_order.inbox
  TO mini_app;
GRANT SELECT, UPDATE ON mini_order.outbox TO mini_relay;

-- +goose Down
DROP SCHEMA mini_order CASCADE;
