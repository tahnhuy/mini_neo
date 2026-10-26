# **NEO Mini — bài thực hành Đặt hàng và Giữ kho** 

**Tóm tắt:** Bài thực hành 3–4 ngày, **mỗi người tự làm toàn bộ bài một mình** : hai microservice Go nói chuyện bằng gRPC và sự kiện NATS, một BFF, một web Next.js. File này tự đủ, không cần mở tài liệu nào khác. 

**Đọc khi:** Nhận bài, làm bài, chấm bài. **Không cần đọc nếu:** Không tham gia bài thực hành. 

**Mục chính:** 2 Nghiệp vụ · 4 Kỹ thuật bắt buộc · 6 API và sự kiện · 8 Kế hoạch 4 ngày · 9 Tiêu chí đạt · 10 Checklist chấm. 

## **1. Bài này để làm gì** 

Dự án thật (NEO) là hệ thống nhiều microservice Go, web Next.js, PostgreSQL, NATS. Bài này dựng lại đúng kiến trúc đó ở cỡ rất nhỏ để mỗi người tự tay làm một lần các việc sau: 

- Chia một service Go thành ba tầng `domain → usecase → adapter` . 

- Viết hợp đồng gRPC bằng Protobuf, sinh code bằng Buf. 

- Cách ly dữ liệu giữa các công ty (tenant) bằng Row Level Security của PostgreSQL. 

- Phát sự kiện an toàn bằng **outbox** , nhận sự kiện không trùng bằng **inbox** . 

- Giữ ba quy tắc nghiệp vụ hay sai nhất: không vượt số khả dụng, tiền không dùng số thực, người lập không tự duyệt. 

Web chỉ hiển thị, không tính toán nghiệp vụ. 

Làm ít tính năng nhưng làm đúng. Một luồng chạy đúng có test tốt hơn năm luồng chạy "gần đúng". 

## **2. Nghiệp vụ** 

Một công ty bán hàng có kho. Luồng chính: 

1. Nhân viên bán **tạo đơn** gồm một hay nhiều dòng (mã hàng, số lượng). Đơn tạo xong ở trạng thái **Chờ duyệt** . Đơn giá lấy từ service kho lúc tạo đơn; client không gửi giá, có gửi cũng bị bỏ qua. 

2. **Người khác** có quyền duyệt bấm **Duyệt** hoặc **Từ chối** . Người lập đơn không tự duyệt được, kể cả khi người đó có quyền duyệt. 

3. Đơn được duyệt thì service đơn phát sự kiện "đơn đã duyệt". Service kho nhận sự kiện và **giữ hàng cho cả đơn** : đủ hàng cho mọi dòng thì giữ hết; thiếu một dòng thì không giữ dòng nào. 

4. Kho phát sự kiện kết quả. Service đơn nhận và chuyển đơn sang **Đã giữ hàng** hoặc **Thiếu hàng** . 

5. Tổng tiền đơn do service đơn tính. BFF và web chỉ hiển thị. 

**Tồn khả dụng = tồn thực tế − đang giữ.** Không bao giờ được để "đang giữ" lớn hơn "tồn thực tế". 

### **Trạng thái đơn** 

|**Trạng thái**|**Nghĩa**|
|---|---|
|`PENDING_APPROVAL`|Chờ duyệt (trạng thái lúc tạo)|



|**Trạng thái**|**Nghĩa**|
|---|---|
|`APPROVED`|Đã duyệt, đang chờ kho trả lời|
|`RESERVED`|Kho đã giữ đủ hàng|
|`OUT_OF_STOCK`|Kho không đủ hàng, không giữ dòng nào|
|`REJECTED`|Bị từ chối|



|**Hành động**|**Từ**|**Sang**|**Điều kiện**|
|---|---|---|---|
|Duyệt|`PENDING_APPROVAL`|`APPROVED`|Có quyền<br>`order.approve`và không phải người|
||||lập|
|Từ chối|`PENDING_APPROVAL`|`REJECTED`|Như trên; lý do dài 10–500 ký tự|
|Nhận "kho đã giữ|`APPROVED`|`RESERVED`|—|
|hàng"||||
|Nhận "kho thiếu hàng"|`APPROVED`|`OUT_OF_STOCK`|—|



Chuyển trạng thái không có trong bảng thì báo lỗi `INVALID_TRANSITION` . Quy tắc chuyển trạng thái viết ở tầng `domain` . 

### **Người dùng mẫu** 

Chưa làm đăng nhập thật. Có bốn người dùng cố định, mỗi người thuộc một công ty (tenant) và có một số **quyền** (capability): 

|**Người dùng**|**Tenant**|**Quyền**|
|---|---|---|
|`an`|A|`order.create`|
|`binh`|A|`order.create`,<br>`order.approve`|
|`chi`|A|`order.approve`|
|`dung`|B|`order.create`,<br>`order.approve`|



Dữ liệu mẫu: tenant A có 3 mặt hàng, trong đó một mặt hàng chỉ còn 1 cái (để thử hai đơn tranh nhau). Tenant B có 2 mặt hàng. Dùng UUID cố định cho hai tenant để dễ test. 

**Kiểm quyền theo tên quyền, không theo tên người hay tên vai trò.** Viết `if caps.Has("order.approve")` , không viết `if user == "binh"` hay `if role == "manager"` . 

## **3. Thành phần** 

|`web (Next.js) ──REST──> bff (Go) ──gRPC──> order (Go) ──gRPC──> stock (Go)`|
|---|
|`│    ▲                │   ▲`|
|`▼    │                ▼   │`|
|`PostgreSQL           PostgreSQL`|
|`schema mini_order    schema mini_stock`|
|`│    ▲                │   ▲`|
|`└────┴──── NATS ──────┴───┘`|
|`(sự kiện qua outbox/inbox)`|



|**Thành**<br>**phần**|**Thư mục**|**Việc**|
|---|---|---|
|`order`|`services/order/`|Đơn, dòng đơn, duyệt; phát "đơn đã duyệt"; nghe<br>kết quả giữ hàng|
|`stock`|`services/stock/`|Mặt hàng, tồn, phiếu giữ hàng; nghe "đơn đã<br>duyệt"; phát kết quả|
|`bff`|`services/bff/`|Đổi REST/JSON sang gRPC; gắn danh tính người<br>dùng;**không có CSDL, không tính toán**|
|Thư viện<br>chung|`pkg/`|`tenant`(đọc danh tính từ context),<br>`db`(mở giao<br>dịch có tenant),<br>`outbox`,<br>`inbox`|
|Hợp đồng|`proto/mini/order/v1/`,<br>`proto/mini/stock/v1/`; mã sinh ở<br>`gen/go/`||
|Web|`web/`|Bốn màn|
|Hạ tầng|`deploy/compose.yaml`|PostgreSQL và NATS|



Hướng gọi đồng bộ chỉ có: web → bff → order hoặc stock; order → stock (lấy đơn giá). **Stock không gọi ngược order** ; việc ngược chiều đi bằng sự kiện. 

Một repo, một `go.mod` ở gốc cho cả ba service Go. 

## **4. Kỹ thuật bắt buộc** 

### **4.1 Công nghệ và phiên bản** 

Ghim đúng phiên bản dưới đây. Muốn thêm thư viện khác thì hỏi người giao bài. 

|**Hạng mục**|**Phiên bản**|
|---|---|
|Go|1.27.x|
|`github.com/jackc/pgx/v5`|v5.11.0|
|`google.golang.org/grpc`|v1.83.2|
|`github.com/nats-io/nats.go`|v1.48.0|



|**Hạng mục**|**Phiên bản**|
|---|---|
|`github.com/google/uuid`|v1.6.0 (dùng UUIDv7)|
|goose (migration)|v3.26.0|
|sqlc (sinh code truy vấn)|v1.30.0|
|Buf (lint, sinh code proto)|v1.54.0|
|PostgreSQL|image<br>`postgres:18`(ghim bản vá cụ thể, ví dụ<br>`18.1`)|
|NATS|image<br>`nats:2.15.1-alpine3.22`, chạy với<br>`-js`|
|Node.js, pnpm|Node 24, pnpm 11|
|Next.js, React, Tailwind CSS|16.x, 19.x, 4.x|



Trong `compose.yaml` , mọi cổng chỉ mở trên máy mình: `"127.0.0.1:5432:5432"` , không viết `"5432:5432"` . 

### **4.2 Khuôn một service Go** 

```
services/order/
```

- `├── cmd/server/main.go     đọc cấu hình từ biến môi trường, nối dây, chạy server ├── internal/ │   ├── domain/            Order, trạng thái, quy tắc chuyển; KHÔNG import pgx, grpc, nats │   ├── usecase/           mở giao dịch, gọi domain, ghi bảng + outbox, kiểm quyền │   └── adapter/ │       ├── repo/          truy vấn sqlc │       ├── grpc/          nhận request, kiểm đầu vào, gọi usecase, đổi lỗi sang mã gRPC │       └── consumer/      nghe NATS, gọi usecase ├── migrations/            file SQL goose đánh số ├── queries/               file .sql cho sqlc └── sqlc.yaml` 

Phụ thuộc chỉ đi một chiều: `adapter → usecase → domain` . 

- Adapter gRPC không chứa quy tắc nghiệp vụ. 

- `context.Context` là tham số đầu của mọi hàm có I/O. Lỗi bọc bằng `fmt.Errorf("...: %w", err)` . Log 

- bằng `log/slog` . Không `panic` ngoài `main` . 

- Giới hạn: file ≤ 500 dòng, hàm ≤ 60 dòng, ≤ 5 tham số, lồng `if/for` ≤ 4 cấp. 

### **4.3 Tiền không dùng số thực** 

   - CSDL: cột tiền `NUMERIC(18,2)` kèm `currency_code CHAR(3)` (bài chỉ dùng `VND` ). 

   - Go: **không dùng** **`float64`** cho tiền. Trong domain dùng `int64` tính bằng xu (1 đồng = 100 xu); adapter đổi qua lại với chuỗi `"1250000.00"` . 

   - Proto và JSON: tiền là chuỗi thập phân kèm mã tiền tệ, ví dụ `{ "amount": "1250000.00", "currencyCode": "VND" }` . 

   - `line_amount = quantity × unit_price` , `total_amount = tổng line_amount` , tính trong domain của service 

   - đơn. 

- Lý do: `0.1 + 0.2` trong số thực không bằng `0.3` ; cộng nhiều dòng tiền sẽ lệch vài đồng. 

### **4.4 Cách ly tenant bằng Row Level Security** 

Mỗi bảng nghiệp vụ có cột `tenant_id UUID NOT NULL` . PostgreSQL tự lọc theo tenant, nên quên `WHERE tenant_id = ...` cũng không lộ dữ liệu công ty khác. 

```
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders FORCE ROW LEVEL SECURITY;   -- áp cả với chủ bảng
CREATE POLICY tenant_isolation ON orders
  USING      (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
```

Ba điều hay sai: 

1. **Service phải kết nối bằng một vai trò thường** , ví dụ `mini_app` : không phải superuser, không phải chủ bảng. Superuser luôn bỏ qua RLS, nên test chạy bằng `postgres` sẽ xanh dù chính sách sai. Migration chạy bằng vai trò khác (chủ bảng). 

2. **Đặt tenant trong từng giao dịch** , ngay sau `BEGIN` : 

```
SELECT set_config('app.tenant_id', $1, true);  -- true = chỉ trong giao dịch này
```

Không đặt cho cả kết nối: kết nối trong pool được dùng lại cho request của tenant khác. 

3. Chưa đặt `app.tenant_id` thì `current_setting` báo lỗi. Đó là điều mong muốn: quên đặt tenant thì truy vấn hỏng, không lộ dữ liệu. 

Gói `pkg/db` chỉ cho mã nghiệp vụ một cách mở giao dịch: `db.InTx(ctx, func(tx pgx.Tx) error)` . Hàm này lấy tenant từ `ctx` , mở giao dịch, đặt `app.tenant_id` , chạy hàm, rồi commit hoặc rollback. 

Ngoại lệ duy nhất: tiến trình relay (mục 4.5) đọc outbox của mọi tenant, nên dùng vai trò riêng `mini_relay` có `BYPASSRLS` và chỉ được cấp quyền trên bảng `outbox` . 

### **4.5 Phát sự kiện bằng outbox** 

Vấn đề: nếu ghi CSDL rồi mới gửi NATS, mà máy chết giữa hai việc, thì dữ liệu đã đổi nhưng sự kiện mất. Nếu gửi trước rồi ghi CSDL lỗi, thì sự kiện kể một việc chưa xảy ra. 

Cách làm: **ghi sự kiện vào bảng** **`outbox` trong cùng giao dịch** với thay đổi nghiệp vụ. Một tiến trình riêng (relay) đọc bảng rồi gửi NATS. Mã nghiệp vụ không bao giờ gọi NATS trực tiếp. 

```
CREATE TABLE outbox (
  id           uuid PRIMARY KEY,          -- = id của sự kiện (UUIDv7)
  tenant_id    uuid NOT NULL,
  type         text NOT NULL,             -- tên sự kiện, cũng là subject NATS
  payload      jsonb NOT NULL,            -- cả phong bì sự kiện
  created_at   timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz
);
```

Để gọn, relay chạy như một goroutine trong chính service (mỗi service một relay cho bảng outbox của mình), dừng khi server tắt. Relay lặp mỗi khoảng 500 ms: 

```
BEGIN;
SELECT id, type, payload FROM outbox
 WHERE published_at IS NULL
 ORDER BY id
 LIMIT 100
 FOR UPDATE SKIP LOCKED;     -- hai relay chạy cùng lúc không lấy trùng dòng
-- gửi từng dòng lên JetStream, header Nats-Msg-Id = id
UPDATE outbox SET published_at = now() WHERE id = ANY($1);
COMMIT;
```

Relay có thể gửi xong rồi chết trước khi đánh dấu, nên một sự kiện có thể **đến hai lần** . Bên nhận phải tự chống trùng (mục 4.6). 

### **4.6 Nhận sự kiện bằng inbox** 

```
CREATE TABLE inbox (
  tenant_id    uuid NOT NULL,
  consumer     text NOT NULL,             -- ví dụ 'stock.reserve'
  event_id     uuid NOT NULL,
  processed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, consumer, event_id)
);
```

Xử lý một sự kiện, tất cả trong **một giao dịch** : 

1. Đặt `app.tenant_id` bằng `tenant_id` trong phong bì sự kiện. 

2. `INSERT INTO inbox ... ON CONFLICT DO NOTHING` . Không chèn được dòng nào nghĩa là sự kiện đã xử lý rồi: commit, ack, dừng. 

3. Làm việc nghiệp vụ, ghi outbox nếu cần phát sự kiện tiếp. 

4. Commit, rồi mới `ack` cho NATS. Lỗi thì không ack để NATS gửi lại. 

Dùng JetStream: một stream `MINI` gom subject `mini.>` , mỗi consumer một durable consumer riêng. 

### **4.7 Giữ hàng không vượt tồn** 

Sai kinh điển: đọc tồn, thấy còn 1, rồi ghi. Hai giao dịch chạy song song cùng đọc thấy 1, cùng giữ, thành âm. 

Làm đúng: **khóa dòng trước khi đọc** , khóa theo thứ tự `sku` để hai giao dịch không chờ nhau vòng tròn (deadlock): 

```
SELECT sku, on_hand, reserved
  FROM stock_items
 WHERE sku = ANY($1)
 ORDER BY sku
 FOR UPDATE;
```

Kiểm trong Go: mọi dòng đều `on_hand - reserved >= quantity` thì cộng `reserved` cho tất cả; thiếu một dòng thì không cộng dòng nào. Thêm ràng buộc `CHECK (reserved >= 0 AND on_hand >= reserved)` làm lưới an toàn cuối, nhưng không dựa vào nó để kiểm. 

### **4.8 Web** 

- Next.js App Router, render phía server. `'use client'` chỉ đặt ở component có tương tác (form, nút). 

- **Trang không gọi** **`fetch` trực tiếp.** Mọi lời gọi BFF nằm trong `web/src/lib/data/` (ví dụ `listOrders()` , `approveOrder()` ); trang chỉ gọi các hàm này rồi truyền props xuống component. 

- **Web không tính toán nghiệp vụ** : không tự cộng tổng tiền, không tự tính tồn khả dụng, không tự quyết ai được bấm Duyệt. Nút nào hiện là theo `allowedActions` do API trả. 

- Tiền hiển thị qua một hàm `formatMoney()` duy nhất, kiểu `1.250.000 ₫` . Không tự format rải rác. 

- Giao diện tiếng Việt, chữ viết thẳng trong code. Không dùng màu hex hay `style=` rải rác; khai màu, khoảng cách một lần trong theme Tailwind rồi dùng lớp. 

- Component React ≤ 200 dòng. 

## **5. Bảng CSDL** 

Schema `mini_order` : 

|**Bảng**|**Cột**|**Ràng buộc**|
|---|---|---|
|`orders`|`id`,<br>`tenant_id`,<br>`code`(<br>`DH-000001`),<br>`status`,<br>`created_by`,|`UNIQUE (tenant_id, code)`|
||`decided_by`,<br>`reject_reason`,<br>`total_amount`,<br>`currency_code`,<br>`created_at`,<br>`updated_at`||
|`order_lines`|`id`,<br>`tenant_id`,<br>`order_id`,<br>`sku`,<br>`quantity`,<br>`unit_price`,|`quantity > 0`;<br>`UNIQUE`|
||`line_amount`|`(tenant_id, order_id,`|
|||`sku)`|
|`outbox`,|Như mục 4.5, 4.6||
|`inbox`|||



#### Schema `mini_stock` : 

|**Bảng**|**Cột**|**Ràng buộc**|
|---|---|---|
|`stock_items`|`tenant_id`,<br>`sku`,<br>`name`,<br>`unit_price`,|Khóa chính<br>`(tenant_id, sku)`;|
||`currency_code`,<br>`on_hand`,<br>`reserved`|`CHECK`như mục 4.7|
|`reservations`|`id`,<br>`tenant_id`,<br>`order_id`,<br>`status`(<br>`HELD`),|`UNIQUE (tenant_id, order_id)`|
||`created_at`||
|`reservation_lines`|`tenant_id`,<br>`reservation_id`,<br>`sku`,<br>`quantity`|`quantity > 0`|



|**Bảng**<br>`outbox`,<br>`inbox`|**Cột**<br>Như mục 4.5, 4.6|**Ràng buộc**|
|---|---|---|



Mọi bảng trên đều bật RLS như mục 4.4 (trừ việc relay dùng vai trò `BYPASSRLS` ). Khóa chính UUIDv7, thời gian `timestamptz` lưu UTC. 

## **6. API và sự kiện** 

### **6.1 gRPC** 

|**Service**|**RPC**|**Quyền cần**|**Ghi chú**|
|---|---|---|---|
|`mini.order.v1.OrderService`|`CreateOrder(lines[{sku,`<br>`quantity}])`|`order.create`|Gọi<br>`StockService.GetItems`<br>lấy giá; mã hàng không có thì<br>`NOT_FOUND`|
||`GetOrder(id)`|`order.create`<br>hoặc<br>`order.approve`|Trả kèm<br>`allowed_actions`|
||`ListOrders()`|như trên|Tối đa 50 đơn mới nhất, chưa<br>cần phân trang|
||`ApproveOrder(id)`|`order.approve`|Người lập tự duyệt:<br>`PERMISSION_DENIED`, lý do<br>`SELF_APPROVAL`|
||`RejectOrder(id, reason)`|`order.approve`||
|`mini.stock.v1.StockService`|`ListItems()`|người dùng bất kỳ<br>của tenant|Trả cả<br>`available = on_hand`<br>`− reserved`|
||`GetItems(skus[])`|gọi nội bộ|Chỉ service đơn gọi|



- `allowed_actions` do service đơn tính từ trạng thái, quyền và người gọi, ví dụ `["APPROVE", "REJECT"]` 

- hoặc `[]` . 

- Mã lỗi: `INVALID_ARGUMENT` (đầu vào sai), `UNAUTHENTICATED` (thiếu danh tính), `PERMISSION_DENIED` (thiếu 

- quyền, tự duyệt), `NOT_FOUND` , `FAILED_PRECONDITION` (chuyển trạng thái sai). Mã lý do ( `SELF_APPROVAL` , `INVALID_TRANSITION` …) gửi kèm trong chi tiết lỗi để BFF trả về cho web. 

- Proto đặt ở `proto/mini/<service>/v1/` , `buf lint` sạch, mã sinh commit vào `gen/go/` . 

### **6.2 Danh tính đi qua các tầng** 

- Web lưu người dùng đã chọn trong cookie `dev_user` và gửi header `X-Dev-User: binh` tới BFF. 

- BFF tra bảng người dùng mẫu (mục 2), rồi gửi sang service qua metadata gRPC: `x-tenant-id` , `x-user-id` , `x-capabilities` (cách nhau bằng dấu phẩy). Người dùng lạ thì BFF trả 401. 

- Mỗi service có một interceptor gRPC đọc metadata, đặt vào `context` . Thiếu `x-tenant-id` thì trả `UNAUTHENTICATED` . 

- Đây là cách làm tạm cho bài. Hệ thống thật đăng nhập qua Keycloak và kiểm token ở gateway. 

### **6.3 REST của BFF** 

|**Phương thức, đường dẫn**|**Gọi tới**|
|---|---|
|`GET /api/me`|Người dùng, tenant, quyền|
|`GET /api/orders`|`ListOrders`|
|`POST /api/orders`|`CreateOrder`|
|`GET /api/orders/{id}`|`GetOrder`|
|`POST /api/orders/{id}/approve`|`ApproveOrder`|
|`POST /api/orders/{id}/reject`|`RejectOrder`(body<br>`{ "reason": "..." }`)|
|`GET /api/stock-items`|`ListItems`|



JSON dùng camelCase. 

- Lỗi trả `{ "code": "SELF_APPROVAL", "message": "Bạn không thể tự duyệt đơn do mình lập" }` . 

- Đổi mã: `INVALID_ARGUMENT` → 400, `UNAUTHENTICATED` → 401, `PERMISSION_DENIED` → 403, `NOT_FOUND` → 404, `FAILED_PRECONDITION` → 409. 

### **6.4 Sự kiện** 

Tên sự kiện: `mini.<service>.<thực thể>.<động từ quá khứ>` . Sự kiện kể việc **đã xảy ra** , không ra lệnh (đúng: `order.approved` ; sai: `reserve_stock` ). 

|**Sự kiện**|**Phát**|**Nghe**|**`data`**||
|---|---|---|---|---|
|`mini.ord.order.approved`|order|stock|`orderId`,|<br>`lines[{sku, quantity}]`|
|`mini.stk.reservation.held`|stock|order|`orderId`,|<br>`reservationId`|
|`mini.stk.reservation.failed`|stock|order|`orderId`,|<br>`shortSkus[]`|



Phong bì (cột `payload` của outbox): 

```
{
  "id": "0192f0c4-7a1e-7c3a-9b1d-4f2e8a6c1d00",
  "type": "mini.ord.order.approved",
  "source": "order",
  "tenantId": "11111111-1111-1111-1111-111111111111",
  "correlationId": "<orderId>",
  "causationId": null,
  "occurredAt": "2026-10-26T03:15:00Z",
  "schemaVersion": 1,
  "data": { "orderId": "...", "lines": [{ "sku": "BAN-01", "quantity": 2 }] }
}
```

- `correlationId` = `orderId` cho cả hành trình; `causationId` = `id` của sự kiện gây ra sự kiện này. 

- Sự kiện **không mang tiền** . Bên nào cần số tiền thì hỏi service sở hữu số đó. 

- Sự kiện kho đến khi đơn không còn ở `APPROVED` thì bỏ qua, nhưng vẫn ghi inbox. 

## **7. Màn web** 

|**Route**|**Nội dung**|
|---|---|
|`/`|Chọn người dùng mẫu (<br>`an`,<br>`binh`,<br>`chi`,<br>`dung`), lưu cookie, chuyển sang<br>`/orders`|
|`/orders`|Danh sách đơn: mã, người lập, tổng tiền, trạng thái; nút "Tạo đơn" nếu có<br>`order.create`|
|`/orders/new`|Bảng mặt hàng (tên, đơn giá, khả dụng), nhập số lượng, bấm "Tạo đơn". Tổng tiền chỉ hiện sau khi<br>service trả về|
|`/orders/[id]`|Chi tiết đơn, các dòng, nút Duyệt hoặc Từ chối theo<br>`allowedActions`; ô nhập lý do khi từ chối. Đơn<br>đang<br>`APPROVED`thì có nút "Làm mới"|



Mỗi màn có đủ trạng thái: đang tải, rỗng, lỗi (hiện `message` từ BFF). 

## **8. Kế hoạch 4 ngày** 

Mỗi người làm cả bài trong repo riêng của mình. Được dùng trợ lý AI, nhưng phải giải thích được từng dòng mình nộp: buổi chấm sẽ hỏi vì sao viết như vậy (mục 10). 

##### **Ngày Mục tiêu cuối ngày** 

- 1 Viết proto và migration theo mục 5, 6; `compose up` chạy được PostgreSQL và NATS; RLS bật, dữ liệu mẫu có sẵn; `pkg/db` xong; service stock trả `ListItems` qua gRPC; test AC-04 đạt 

- 2 Service order: `CreateOrder` (gọi stock lấy giá), `GetOrder` , `ListOrders` , `ApproveOrder` , `RejectOrder` ; unit test AC-01, AC-02, AC-03, AC-09 đạt 

- 3 Outbox, relay, inbox; stock giữ hàng, đơn sang `RESERVED` hoặc `OUT_OF_STOCK` ; test tích hợp AC-05 đến AC08 đạt 

- 4 BFF và bốn màn web; `README.md` của repo có lệnh chạy từ máy trống tới demo; tự rà theo checklist mục 10 rồi nộp 

Thứ tự ưu tiên khi thiếu thời gian: phần Go và test (ngày 1–3) quan trọng hơn web. Web xấu nhưng đúng luật mục 4.8 vẫn đạt; web đẹp mà thiếu AC-07, AC-08 thì không đạt. 

Ngày 1 đáng đầu tư nhất: proto và bảng sai thì ngày 2–3 phải sửa lại cả chuỗi. 

## **9. Tiêu chí đạt** 

Tên test chứa mã ở cột đầu, ví dụ `TestApprove_AC03_SelfApprovalDenied` . 

|**Mã**|**Tình huống**|**Kết quả mong đợi**|
|---|---|---|
|AC-<br>01|`an`tạo đơn 2 dòng|Giá lấy từ kho; tổng đúng đến đồng; client gửi giá thì bị bỏ qua|
|AC-|`an`(không có<br>`order.approve`) duyệt|`PERMISSION_DENIED`; web không hiện nút Duyệt cho<br>`an`|
|02|đơn||
|AC-|`binh`duyệt đơn do chính<br>`binh`lập|`PERMISSION_DENIED`, lý do<br>`SELF_APPROVAL`; đơn giữ nguyên|
|03||`PENDING_APPROVAL`|



|**Mã**|**Tình huống**|**Kết quả mong đợi**|
|---|---|---|
|AC-<br>04|`dung`(tenant B) xem đơn của tenant A<br>bằng đúng<br>`id`|`NOT_FOUND`. Thử thêm ở mức SQL bằng vai trò<br>`mini_app`: cũng<br>không thấy dòng nào|
|AC-<br>05|`chi`duyệt đơn đủ hàng|Đơn sang<br>`RESERVED`;<br>`reserved`của từng mặt hàng tăng đúng số<br>lượng|
|AC-<br>06|Duyệt đơn có một dòng thiếu hàng|Đơn sang<br>`OUT_OF_STOCK`; không dòng nào bị giữ|
|AC-|Cùng một sự kiện|Chỉ giữ hàng một lần|
|07|`mini.ord.order.approved`được giao 2<br>lần||
|AC-|Hai đơn cùng cần mặt hàng chỉ còn 1, xử|Đúng một đơn<br>`RESERVED`, một đơn<br>`OUT_OF_STOCK`;<br>`reserved`|
|08|lý song song|không bao giờ vượt<br>`on_hand`. Chạy test này ít nhất 20 lần liền|
|AC-<br>09|Duyệt đơn đang<br>`REJECTED`|`FAILED_PRECONDITION`, lý do<br>`INVALID_TRANSITION`|



Loại test: 

- AC-01 (phần tính tổng), AC-03, AC-09: unit test dạng bảng ở tầng `domain` / `usecase` , không cần CSDL. 

- AC-04 đến AC-08: test tích hợp với PostgreSQL và NATS thật, đặt build tag `//go:build integration` , chạy bằng `go test -tags integration ./...` . Dùng CSDL riêng cho test (tạo mới rồi xóa), không đụng dữ liệu đang demo. 

- AC-08 phải có **đối chứng âm** : tạm bỏ `FOR UPDATE` thì test phải đỏ. Test không đỏ khi bỏ khóa là test chưa chứng minh được gì. 

### **Làm thêm nếu còn thời gian** 

|**Mã**|**Việc**|
|---|---|
|EX-01|Hủy đơn đã giữ hàng: phát<br>`mini.ord.order.cancelled`, kho nhả lượng đã giữ (bước bù trừ)|
|EX-02|Tắt relay, duyệt đơn, bật lại relay: sự kiện vẫn đến, đơn vẫn sang<br>`RESERVED`|
|EX-03|Thủ kho điều chỉnh tồn, không được làm<br>`on_hand < reserved`|
|EX-04|Một kịch bản Playwright chạy luồng đủ trên web|
|EX-05|Thử lại tối đa 5 lần khi consumer lỗi rồi ghi log lỗi|



## **10. Checklist chấm** 

Nộp bài: link repo (hoặc file nén) kèm danh sách mã AC đã đạt và kết quả thật của các lệnh kiểm. Commit nhỏ theo từng bước, không dồn một commit cuối ngày 4. 

Người chấm chạy lại bài từ `README.md` , rà theo danh sách dưới, rồi hỏi 15–20 phút về code (ví dụ: "bỏ `FOR UPDATE` thì chuyện gì xảy ra?", "vì sao không gửi NATS ngay trong usecase?"). Một mục **Bắt buộc** trượt thì bài chưa đạt. 

#### **Chung** 

- ☐ Bắt buộc: bài nộp ghi mã AC đã đạt và lệnh kiểm đã chạy, kèm kết quả thật. 

- ☐ Bắt buộc: giải thích được code mình nộp khi được hỏi. 

- ☐ Bắt buộc: không commit mật khẩu, file `.env` ; cấu hình đọc từ biến môi trường. 

- ☐ Bắt buộc: file ≤ 500 dòng, hàm ≤ 60 dòng, component React ≤ 200 dòng. 

- ☐ Nên: tên rõ nghĩa, comment giải thích **vì sao** chứ không kể lại code; không để code bị comment. 

#### **Go** 

- ☐ Bắt buộc: `gofmt` , `go vet` , `go test ./...` sạch. 

- ☐ Bắt buộc: `domain` không import pgx, grpc, nats; adapter gRPC không chứa quy tắc nghiệp vụ. 

- ☐ Bắt buộc: kiểm quyền theo tên quyền, không theo tên người hay vai trò. 

- ☐ Bắt buộc: tiền không đi qua `float64` ở bất kỳ đâu. 

- ☐ Bắt buộc: mọi lời gọi gRPC và NATS có timeout; goroutine có đường dừng khi tắt server. 

#### **CSDL** 

- ☐ Bắt buộc: mọi bảng có `tenant_id` bật `ENABLE` và `FORCE ROW LEVEL SECURITY` . 

- ☐ Bắt buộc: service chạy bằng `mini_app` , không phải `postgres` ; `app.tenant_id` đặt bằng `set_config(..., true)` trong giao dịch. 

- ☐ Bắt buộc: truy vấn qua sqlc, không ghép chuỗi SQL. 

- ☐ Bắt buộc: migration đã merge thì không sửa; muốn đổi thì thêm file mới. 

#### **Sự kiện và nghiệp vụ** 

- ☐ Bắt buộc: outbox ghi cùng giao dịch với thay đổi nghiệp vụ; không chỗ nào trong usecase gọi NATS. 

- ☐ Bắt buộc: inbox ghi cùng giao dịch xử lý; ack sau khi commit. 

- ☐ Bắt buộc: giữ hàng khóa dòng ( `FOR UPDATE` , theo thứ tự `sku` ) trước khi đọc tồn; tất cả hoặc không gì cả. 

- ☐ Bắt buộc: chặn tự duyệt ở usecase, không chỉ ẩn nút trên web. 

#### **BFF và web** 

- ☐ Bắt buộc: BFF không có CSDL, không tính tổng, không tự quyết quyền. 

- ☐ Bắt buộc: trang không `fetch` trực tiếp, chỉ gọi hàm trong `lib/data` . 

- ☐ Bắt buộc: nút theo `allowedActions` ; tiền qua `formatMoney()` . 

- ☐ Nên: Server Component mặc định, `'use client'` đặt sâu nhất có thể. 

#### **Hạ tầng** 

- ☐ Bắt buộc: cổng compose chỉ mở trên `127.0.0.1` ; image ghim phiên bản. 

- ☐ Bắt buộc: `README.md` của repo có các lệnh đi từ máy trống tới lúc demo được. 

## **11. Đọc thêm (tùy chọn)** 

- PostgreSQL Row Level Security: https://www.postgresql.org/docs/current/ddl-rowsecurity.html NATS JetStream: https://docs.nats.io/nats-concepts/jetstream 

- sqlc: https://docs.sqlc.dev 

goose: https://pressly.github.io/goose/ 

Buf: https://buf.build/docs 

Next.js App Router: https://nextjs.org/docs/app 

