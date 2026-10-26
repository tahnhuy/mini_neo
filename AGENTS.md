# AGENTS.md - Quy tắc viết docs cho NEO Mini

## VAI TRÒ

Bạn là technical writer kiêm kỹ sư backend Go, viết bộ docs cho bài thực hành "NEO Mini - Đặt hàng và Giữ kho".

## NGUỒN SỰ THẬT

1. `docs/source/neo-mini.md` (đề bài). Đọc TOÀN BỘ trước khi viết.
2. Mục "QUYẾT ĐỊNH ĐÃ CHỐT" bên dưới. Khi đề và quyết định mâu thuẫn, quyết định thắng, nhưng phải đánh dấu "bổ sung ngoài đề" tại chỗ dùng.

Không dùng kiến thức ngoài để thêm yêu cầu.

## CÂY DOCS ĐÍCH

```
README.md
docs/source/neo-mini.md        (đề, không sửa)
docs/tech-stack.md
docs/conventions.md
docs/specs/business-rules.md
docs/specs/data-model.md
docs/specs/api-contract.md
docs/specs/events.md
docs/specs/web.md
docs/specs/acceptance.md
docs/specs/tasks.md
```

Chỉ tạo đúng file và đường dẫn mà phase hiện tại yêu cầu.

## QUY TẮC VIẾT

1. Không bịa. Tên bảng, cột, RPC, sự kiện, quyền, mã lỗi, phiên bản, mã AC phải khớp từng ký tự với nguồn.
2. Điều đề không nói và không có trong quyết định thì ghi trong khối "Gợi ý (ngoài đề)" riêng, không trộn vào nội dung chính.
3. Mỗi thông tin chỉ định nghĩa ở MỘT nơi, nơi khác dùng link tương đối. Phân công: phiên bản chỉ ở `tech-stack.md`; quy tắc code, tiền, kiểm quyền chỉ ở `conventions.md`; bảng và cột chỉ ở `data-model.md`; RPC và REST chỉ ở `api-contract.md`; sự kiện chỉ ở `events.md`.
4. Đầu mỗi file ghi: mục đích (1-2 câu), đối tượng đọc, dòng "Nguồn: mục X của đề" hoặc "Nguồn: quyết định Qn".
5. Viết tiếng Việt; giữ nguyên tiếng Anh cho định danh code (bảng, RPC, sự kiện, trạng thái, mã lỗi).
6. Dùng bảng cho dữ liệu có cấu trúc, câu ngắn cho quy tắc. Quy tắc nào có lý do trong đề thì ghi lý do một dòng.
7. Đánh dấu BẮT BUỘC / NÊN đúng như mục 10 của đề.
8. Sơ đồ dùng Mermaid, phải khớp 100% với bảng tương ứng.
9. Gặp điểm mơ hồ MỚI ngoài danh sách quyết định: không tự quyết; nêu hai cách hiểu, đề xuất một cách, đánh dấu "CẦN XÁC NHẬN", liệt kê trong báo cáo cuối phase.
10. Không tạo lệnh, script hay Makefile chưa tồn tại. Lệnh chưa có thật thì để placeholder `<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>`.

## KẾT THÚC MỖI PHASE

Báo cáo gồm:

- (a) file đã tạo;
- (b) nội dung "bổ sung ngoài đề" và "Gợi ý" đã thêm;
- (c) điểm "CẦN XÁC NHẬN" mới theo quy tắc 9;
- (d) tự kiểm theo "Tiêu chí xong" của phase.

Không tự bắt đầu phase sau.

## QUYẾT ĐỊNH ĐÃ CHỐT

- **Q1 (sự kiện).** Giữ đúng bảng đề: `mini.ord.order.approved`, `mini.stk.reservation.held`, `mini.stk.reservation.failed`. `ord` = order, `stk` = stock. Quy ước `mini.<service>.<thực thể>.<động từ quá khứ>` áp dụng với service viết tắt. Chốt vì đề dùng đúng các tên này ở bảng mục 6.4, phong bì mẫu và AC-07; không có chỗ nào trong đề dùng `mini.order.*` hay `mini.stock.*`.
- **Q2 (AC-04).** Ngày 1 kiểm ở mức SQL bằng vai trò `mini_app`; mức API (`NOT_FOUND`) kiểm khi có `GetOrder` ở ngày 2.
- **Q3 (reservations).** `status` chỉ có `HELD`. `RELEASED` chỉ xuất hiện nếu làm EX-01 và phải ghi là mở rộng.
- **Q4 (PostgreSQL).** Ghim bản vá 18.x cụ thể. Chốt `postgres:18.6`, bản vá mới nhất của PostgreSQL 18 tại ngày 26/10/2026 (đã kiểm release notes của PostgreSQL và tag trên Docker Hub). [bổ sung ngoài đề]
- **Q5 (order_lines).** Thêm cột `currency_code CHAR(3)` để đúng quy tắc tiền mục 4.3 (cột tiền kèm `currency_code`). Chốt vì `order_lines` có hai cột tiền `unit_price`, `line_amount`. [bổ sung ngoài đề]
- **Q6 (khóa chính).** `stock_items`: `(tenant_id, sku)` theo đề. `reservation_lines`: `(tenant_id, reservation_id, sku)`. [bổ sung ngoài đề]
- **Q7 (mã đơn).** Sinh `DH-000001` bằng bảng đếm theo tenant (có RLS), khóa dòng `FOR UPDATE` trong cùng giao dịch tạo đơn; không dùng `MAX()+1`. Tên bảng: `order_counters (tenant_id, last_number)`. Chốt vì `MAX()+1` cho hai giao dịch song song cùng một số và vi phạm `UNIQUE (tenant_id, code)`. [bổ sung ngoài đề]
- **Q8 (GetItems).** BFF không gọi `GetItems`. Service order truyền lại metadata tenant khi gọi stock. Không thêm cơ chế xác thực riêng; đây là quy ước của bài.
- **Q9 (UUID tenant).** A = `11111111-1111-1111-1111-111111111111`, B = `22222222-2222-2222-2222-222222222222`.
- **Q10 (mã lý do lỗi).** Chỉ hai mã lý do có tên: `SELF_APPROVAL`, `INVALID_TRANSITION`. Lỗi khác chỉ có mã gRPC và message.
- **Q11 (lý do từ chối).** 10-500 ký tự, đếm theo ký tự Unicode (rune), không theo byte.
- **Q12 (inbox / JetStream).** `inbox.consumer`: `stock.reserve`, `order.reservation_held`, `order.reservation_failed`. Tên durable consumer thay "." bằng "-" (ví dụ `stock-reserve`) vì tên consumer JetStream không được chứa dấu chấm. Stream `MINI`, subject `mini.>`.
- **Q13 (lỗi đánh máy).** "AC-05 đến AC08" ở mục 8 hiểu là AC-05 đến AC-08; ghi chú, không sửa file nguồn.
- **Q14 (mã lỗi JSON của BFF).** Lỗi có mã lý do (Q10) thì `code` là mã lý do. Lỗi không có mã lý do thì `code` là tên mã gRPC (ví dụ `"NOT_FOUND"`, `"INVALID_ARGUMENT"`). JSON lỗi luôn có `code` và `message`. [bổ sung ngoài đề]
- **Q15 (danh tính người dùng).** `x-user-id` mang tên người dùng mẫu dạng chuỗi (`an`, `binh`, `chi`, `dung`), giống giá trị header `X-Dev-User`. Cột `created_by`, `decided_by` lưu đúng chuỗi đó. Không cấp UUID cho người dùng. [bổ sung ngoài đề]
- **Q16 (dữ liệu cho AC-04 ở mức SQL).** Test tự chèn đơn của tenant A vào CSDL riêng cho test, rồi truy vấn dưới tenant B bằng vai trò `mini_app`. Dữ liệu mẫu chỉ gồm tenant và mặt hàng, không có đơn. [bổ sung ngoài đề]
- **Q17 (đếm ký tự lý do từ chối).** Đếm rune nguyên trạng theo Q11, không chuẩn hóa NFC/NFD trước khi đếm, vì chuẩn hóa cần thư viện ngoài bảng phiên bản. [bổ sung ngoài đề]
- **Q18 (PostgreSQL).** Một máy chủ PostgreSQL trong `deploy/compose.yaml`, hai schema `mini_order` và `mini_stock`. [bổ sung ngoài đề]
- **Q19 (nhãn quy tắc).** Quy tắc ở mục 4, 8, 9 của đề viết dạng mệnh lệnh nhưng không có trong checklist mục 10 vẫn gắn BẮT BUỘC, và ghi "—" ở cột "Mục 10". Chỉ hai dòng NÊN của mục 10 mới gắn NÊN. Riêng `'use client'`, có hai dòng: "chỉ đặt ở component có tương tác" (mục 4.8) là BẮT BUỘC; "đặt sâu nhất có thể" (mục 10) là NÊN.
- **Q20 (khuôn service).** `stock` theo đúng khuôn `services/order/` ở mục 4.2. `bff` giữ `cmd/server/main.go` và `internal/adapter/`, không có `domain/`, `repo/`, `migrations/`, `queries/`, `sqlc.yaml`, vì `bff` không có CSDL và không tính toán. [bổ sung ngoài đề]
- **Q21 (độ dài hàm).** Hàm Go < 40 dòng, chặt hơn mức ≤ 60 dòng của đề. Đạt < 40 thì tự đạt ≤ 60. Giới hạn file giữ ≤ 500 dòng theo đề; không thêm mức < 1000 dòng vì mức đó lỏng hơn đề. [bổ sung ngoài đề]
- **Q22 (kiểu cột).** Cột đề không nêu kiểu: mã và tên (`code`, `status`, `sku`, `name`, `reject_reason`) dùng `text`; số lượng (`quantity`, `on_hand`, `reserved`) dùng `integer`; `last_number` dùng `bigint`. Mọi cột `NOT NULL`, trừ `decided_by` và `reject_reason` (để `NULL` khi đơn chưa được quyết định). [bổ sung ngoài đề]
- **Q23 (vai trò migration).** Migration chạy bằng vai trò chủ bảng riêng `mini_owner`, không phải superuser, không dùng `postgres`. [bổ sung ngoài đề]
- **Q24 (vai trò service).** Hai service dùng chung một vai trò `mini_app`, đúng tên của đề. Việc mỗi service chỉ đọc schema của mình dựa vào code, không dựa vào CSDL. [bổ sung ngoài đề]
- **Q25 (field response gRPC).** Đơn: `id`, `code`, `status`, `created_by`, `decided_by`, `reject_reason`, `total_amount`, `lines[]`, `created_at`, `updated_at`. Dòng đơn: `sku`, `quantity`, `unit_price`, `line_amount`. `GetOrder`, `CreateOrder`, `ApproveOrder`, `RejectOrder` trả đơn kèm `allowed_actions`. `ListOrders` trả `orders[]` (tối thiểu `id`, `code`, `created_by`, `total_amount`, `status`), xếp `created_at` giảm dần. `ListItems` trả `sku`, `name`, `unit_price`, `on_hand`, `reserved`, `available`. `GetItems` trả `sku`, `unit_price`. [bổ sung ngoài đề]
- **Q26 (kiểu trạng thái trong proto).** `status` và `allowed_actions` là `string` trong proto, giữ nguyên giá trị như đề (`PENDING_APPROVAL`, `APPROVE`, `REJECT`…); không dùng enum proto. [bổ sung ngoài đề]
- **Q27 (mã hàng không có).** `GetItems` chỉ trả các mã tìm thấy; usecase `CreateOrder` so khớp và trả `NOT_FOUND`. [bổ sung ngoài đề]
- **Q28 (kiểm đầu vào).** Adapter gRPC trả `INVALID_ARGUMENT` khi: `lines` rỗng; có `quantity <= 0`; một `sku` lặp lại trong cùng đơn; lý do từ chối sai độ dài. [bổ sung ngoài đề]
- **Q29 (chi tiết lỗi).** Mã lý do gửi bằng message riêng `ErrorDetail { string reason = 1; }` khai trong `proto/mini/order/v1/`, đính vào status gRPC làm chi tiết lỗi; BFF đọc `reason` từ chi tiết lỗi. Không dùng `google.rpc.ErrorInfo` để khỏi thêm module `google.golang.org/genproto/googleapis/rpc` ngoài bảng phiên bản (mục 4.1). [bổ sung ngoài đề]
- **Q30 (tự từ chối).** Người lập gọi `RejectOrder` trên đơn của mình thì trả `PERMISSION_DENIED`, lý do `SELF_APPROVAL`. [bổ sung ngoài đề]
- **Q31 (thứ tự kiểm khi duyệt/từ chối).** Kiểm quyền → kiểm tự duyệt → kiểm chuyển trạng thái. [bổ sung ngoài đề]
- **Q32 (message lỗi).** Service đặt message tiếng Việt trong status gRPC; BFF chuyển nguyên. [bổ sung ngoài đề]
- **Q33 (`GET /api/me`).** Trả `{ "userId": ..., "tenantId": ..., "capabilities": [...] }`, với `userId` theo Q15 và `tenantId` theo Q9. BFF trả lời từ bảng người dùng mẫu, không gọi gRPC. [bổ sung ngoài đề]
- **Q34 (trạng thái màn `/`).** Màn `/` dùng danh sách 4 người dùng cố định trong code, không gọi BFF, nên không có trạng thái đang tải, rỗng, lỗi. Ba trạng thái áp cho `/orders`, `/orders/new`, `/orders/[id]`. [bổ sung ngoài đề]
- **Q35 (loại test AC-01, AC-02).** AC-02 (`PERMISSION_DENIED`): unit test ở `usecase`. AC-01 "giá lấy từ kho": unit test ở `usecase` với `StockService` giả. AC-01 "client gửi giá bị bỏ qua": test ở handler BFF. AC-02 "web không hiện nút Duyệt cho `an`": kiểm bằng tay khi demo, hoặc bằng EX-04 nếu làm. [bổ sung ngoài đề]
- **Q36 (NATS cho test tích hợp).** Test tích hợp dùng một máy chủ NATS riêng, không dùng chung với demo; cổng chỉ mở trên `127.0.0.1`. [bổ sung ngoài đề]
- **Q37 (EX-01, mở rộng).** Chỉ áp dụng khi làm EX-01; không tính vào số liệu tự kiểm. Đơn có thêm trạng thái `CANCELLED`; hành động "Hủy" chuyển `RESERVED` → `CANCELLED`, cần quyền `order.approve`. RPC `CancelOrder(id)`, REST `POST /api/orders/{id}/cancel`; `allowed_actions` có `"CANCEL"` khi đơn ở `RESERVED` và người gọi có `order.approve`. Sự kiện `mini.ord.order.cancelled` có `data` = `orderId` (không mang tiền; kho tìm phiếu qua `UNIQUE (tenant_id, order_id)`). Kho nghe bằng consumer `stock.release` (durable `stock-release`): khóa `stock_items` `FOR UPDATE` theo `sku`, trừ `reserved` theo `reservation_lines`, đổi `reservations.status` sang `RELEASED`, cùng giao dịch với inbox. Không có sự kiện trả lời. [bổ sung ngoài đề]
- **Q38 (hàm `lib/data`).** `getMe()`, `listOrders()`, `createOrder()`, `getOrder()`, `approveOrder()`, `rejectOrder()`, `listStockItems()`, mỗi hàm ứng một endpoint REST. `listOrders()`, `approveOrder()` là ví dụ của đề. [bổ sung ngoài đề]
- **Q39 (dữ liệu mẫu).** Tenant A: `BAN-01` "Bàn làm việc" 1.250.000 ₫, `on_hand` 10; `GHE-01` "Ghế xoay" 850.000 ₫, `on_hand` 20; `DEN-01` "Đèn bàn" 320.000 ₫, `on_hand` 1 (mặt hàng chỉ còn 1 cái). Tenant B: `BAN-01` "Bàn họp" 2.500.000 ₫, `on_hand` 5; `TU-01` "Tủ hồ sơ" 1.800.000 ₫, `on_hand` 3. Mọi mặt hàng `reserved` = 0, `currency_code` = `VND`. `BAN-01` có ở cả hai tenant để thử cách ly tenant. [bổ sung ngoài đề]
- **Q40 (plugin sinh code proto).** `protoc-gen-go` v1.36.11 (cùng bản `google.golang.org/protobuf` trong `go.mod`) và `protoc-gen-go-grpc` v1.6.2, chạy cục bộ qua `buf.gen.yaml`. [bổ sung ngoài đề]
- **Q41 (khởi tạo CSDL).** Script `deploy/initdb/01-roles.sh` tạo ba vai trò và CSDL `mini` thuộc `mini_owner`. `mini_owner` có `CREATEDB` để test tích hợp tự tạo và xóa CSDL riêng mà không cần superuser. Schema `mini_order`, `mini_stock` do migration tạo. Bảng version goose: `goose_mini_order`, `goose_mini_stock` (schema `public`). Dữ liệu mẫu Q39 là migration `services/stock/migrations/00002_seed.sql`. [bổ sung ngoài đề]
- **Q42 (bố cục code thêm).** `pkg/tenant/tenantgrpc`: interceptor đọc metadata và hàm truyền lại metadata khi gọi service khác. `internal/testdb`: hỗ trợ test tích hợp (tạo CSDL, migrate, xóa). `services/<svc>/internal/adapter/moneyconv`: đổi tiền chuỗi ↔ `int64` xu. Mã sqlc sinh ở `internal/adapter/repo/sqlcdb`. Query đọc tiền bằng `::text` để không qua số thực. [bổ sung ngoài đề]

## SỐ LIỆU ĐỂ TỰ KIỂM

| Hạng mục | Số lượng |
|---|---|
| AC | 9 (AC-01..09) |
| EX | 5 (EX-01..05) |
| RPC | 7 (OrderService 5 + StockService 2) |
| Endpoint REST của BFF | 7 |
| Sự kiện | 3 |
| Route web | 4 |
| Dòng checklist mục 10 | 24 (Chung 5, Go 5, CSDL 4, Sự kiện và nghiệp vụ 4, BFF và web 4, Hạ tầng 2) |
