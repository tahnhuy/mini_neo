# Tiêu chí đạt và chấm bài NEO Mini

> **Mục đích:** Định nghĩa 9 tiêu chí đạt (AC), cách test từng AC, lệnh chạy test, phần làm thêm (EX) và checklist chấm.
>
> **Đối tượng đọc:** Người làm bài (khi viết test và nộp bài), người chấm bài.
>
> **Nguồn:** mục 8, 9, 10 của đề; quyết định Q2, Q3, Q16, Q35, Q36, Q37.

Quy tắc viết test (tên test, build tag, CSDL riêng): xem [../conventions.md](../conventions.md), mục 10. Quy tắc commit và nộp bài: xem [../conventions.md](../conventions.md), mục 11.

## 1. Tiêu chí đạt (AC)

| Mã | Tình huống | Kết quả mong đợi | Loại test | Tầng | Tên test mẫu (gợi ý đặt tên) |
|---|---|---|---|---|---|
| AC-01 | `an` tạo đơn 2 dòng | Giá lấy từ kho; tổng đúng đến đồng; client gửi giá thì bị bỏ qua | Unit dạng bảng (phần tính tổng) | `domain` / `usecase` | `TestCreateOrder_AC01_TotalFromStockPrice` |
| AC-02 | `an` (không có `order.approve`) duyệt đơn | `PERMISSION_DENIED`; web không hiện nút Duyệt cho `an` | Unit | `usecase` | `TestApprove_AC02_MissingCapabilityDenied` |
| AC-03 | `binh` duyệt đơn do chính `binh` lập | `PERMISSION_DENIED`, lý do `SELF_APPROVAL`; đơn giữ nguyên `PENDING_APPROVAL` | Unit dạng bảng | `domain` / `usecase` | `TestApprove_AC03_SelfApprovalDenied` (ví dụ của đề) |
| AC-04 | `dung` (tenant B) xem đơn của tenant A bằng đúng `id` | `NOT_FOUND`. Thử thêm ở mức SQL bằng vai trò `mini_app`: cũng không thấy dòng nào | Tích hợp (PostgreSQL) | SQL; API `GetOrder` | `TestTenantIsolation_AC04_OtherTenantNotFound` |
| AC-05 | `chi` duyệt đơn đủ hàng | Đơn sang `RESERVED`; `reserved` của từng mặt hàng tăng đúng số lượng | Tích hợp (PostgreSQL, NATS) | Đầu-cuối qua outbox/inbox | `TestReserve_AC05_EnoughStockReserved` |
| AC-06 | Duyệt đơn có một dòng thiếu hàng | Đơn sang `OUT_OF_STOCK`; không dòng nào bị giữ | Tích hợp (PostgreSQL, NATS) | Đầu-cuối qua outbox/inbox | `TestReserve_AC06_ShortLineNothingHeld` |
| AC-07 | Cùng một sự kiện `mini.ord.order.approved` được giao 2 lần | Chỉ giữ hàng một lần | Tích hợp (PostgreSQL, NATS) | Consumer `stock.reserve` | `TestInbox_AC07_DuplicateEventReservedOnce` |
| AC-08 | Hai đơn cùng cần mặt hàng chỉ còn 1, xử lý song song | Đúng một đơn `RESERVED`, một đơn `OUT_OF_STOCK`; `reserved` không bao giờ vượt `on_hand`. Chạy test này ít nhất 20 lần liền | Tích hợp (PostgreSQL, NATS); có đối chứng âm (mục 4) | Consumer `stock.reserve` | `TestReserve_AC08_ConcurrentLastItem` |
| AC-09 | Duyệt đơn đang `REJECTED` | `FAILED_PRECONDITION`, lý do `INVALID_TRANSITION` | Unit dạng bảng | `domain` / `usecase` | `TestApprove_AC09_RejectedInvalidTransition` |

- Tên test phải chứa mã AC ([../conventions.md](../conventions.md), mục 10). Ngoài `TestApprove_AC03_SelfApprovalDenied` của đề, các tên ở cột cuối chỉ là gợi ý (ngoài đề).
- AC nào cần đạt ở ngày nào, thứ tự ưu tiên khi thiếu thời gian, ghi chú Q13: xem [tasks.md](tasks.md).

> **Bổ sung ngoài đề (Q35), loại test của AC-01 và AC-02:** Mục 9 chỉ xếp "AC-01 (phần tính tổng)", AC-03, AC-09 vào unit test, còn mục 8 ghi ngày 2 có "unit test AC-01, AC-02, AC-03, AC-09". Các phần còn lại test như sau:
>
> | Phần | Loại test | Tầng |
> |---|---|---|
> | AC-02: `PERMISSION_DENIED` | Unit | `usecase` |
> | AC-01: giá lấy từ kho | Unit, dùng `StockService` giả | `usecase` |
> | AC-01: client gửi giá bị bỏ qua | Test handler | BFF (proto không có trường giá, [api-contract.md](api-contract.md), mục 5) |
> | AC-02: web không hiện nút Duyệt cho `an` | Kiểm bằng tay khi demo, hoặc bằng EX-04 nếu làm | Web |

## 2. AC-04 theo Q2

| Ngày | Mức kiểm | Cách làm |
|---|---|---|
| 1 | SQL | Kết nối bằng vai trò `mini_app` vào CSDL riêng cho test. Test tự chèn một đơn của tenant A (Q16), rồi truy vấn dưới tenant B (`set_config('app.tenant_id', ..., true)`). Kết quả mong đợi: không thấy dòng nào |
| 2 | API | Khi đã có `GetOrder`: `dung` gọi `GetOrder` với đúng `id` đơn của tenant A. Kết quả mong đợi: `NOT_FOUND` |

Vai trò `mini_app` và UUID tenant: xem [data-model.md](data-model.md), mục 6 và [business-rules.md](business-rules.md), mục 4.

## 3. Lệnh chạy test

Lệnh chạy unit test và test tích hợp: xem [README.md](../../README.md), mục 4.

| Loại | Điều kiện |
|---|---|
| Unit | Không cần CSDL |
| Tích hợp | File test đặt build tag `//go:build integration`; cần PostgreSQL và NATS thật |

- Test tích hợp dùng CSDL riêng cho test: tạo mới rồi xóa, không đụng dữ liệu đang demo.
- Lệnh dựng PostgreSQL, NATS cho test và lệnh tạo CSDL test: `<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>`.

> **Bổ sung ngoài đề (Q36):** Test tích hợp dùng một máy chủ NATS riêng, không dùng chung với demo; cổng chỉ mở trên `127.0.0.1` ([../tech-stack.md](../tech-stack.md), mục 5). Lý do: stream `MINI` và tên durable là cố định ([events.md](events.md), mục 6), nên nếu dùng chung NATS, service demo có thể nhận sự kiện của test và đụng vào dữ liệu demo.

## 4. AC-08: chạy lặp và đối chứng âm

| Yêu cầu | Nội dung |
|---|---|
| Chạy lặp | Chạy test AC-08 ít nhất 20 lần liền; cả 20 lần phải xanh |
| Đối chứng âm | Tạm bỏ `FOR UPDATE` ([events.md](events.md), mục 7) thì test phải đỏ. Test không đỏ khi bỏ khóa là test chưa chứng minh được gì |
| Ghi lại kết quả | Ghi cả hai lần chạy vào bảng kết quả ở [README.md](../../README.md), mục 6: lần chạy lặp có khóa (xanh) và lần đối chứng âm không khóa (đỏ), kèm output thật |

Cách test các AC sự kiện: Q45 (**bổ sung ngoài đề**). Kết quả thật ngày 26/10/2026: có khóa xanh 20/20 (4 lượt); bỏ khóa đỏ 20/20 (3 lượt), lỗi `trạng thái hai đơn = map[RESERVED:2]`.

> **Gợi ý (ngoài đề):**
> - Có thể chạy lặp bằng cờ `-count` của `go test`, ví dụ `go test -tags integration -run AC08 -count=20 ./...`. `-count` cũng bỏ qua cache kết quả test.
> - Làm đối chứng âm trên một nhánh tạm: bỏ `FOR UPDATE` trong file query, sinh lại sqlc, chạy test, lưu output, rồi bỏ nhánh. Không merge nhánh này.

## 5. Làm thêm nếu còn thời gian

| Mã | Việc |
|---|---|
| EX-01 | Hủy đơn đã giữ hàng: phát sự kiện hủy đơn (tên: [events.md](events.md), mục 2), kho nhả lượng đã giữ (bước bù trừ) |
| EX-02 | Tắt relay, duyệt đơn, bật lại relay: sự kiện vẫn đến, đơn vẫn sang `RESERVED` |
| EX-03 | Thủ kho điều chỉnh tồn, không được làm `on_hand < reserved` |
| EX-04 | Một kịch bản Playwright chạy luồng đủ trên web |
| EX-05 | Thử lại tối đa 5 lần khi consumer lỗi rồi ghi log lỗi |

- **EX-01 là phần mở rộng (Q3):** nếu làm thì thêm giá trị `RELEASED` cho `reservations.status` ([data-model.md](data-model.md), mục 3) và ghi rõ là mở rộng.
- Thiết kế EX-01 đã chốt theo Q37 (**bổ sung ngoài đề**):
  - Trạng thái `CANCELLED`: [business-rules.md](business-rules.md), mục 10.
  - `CancelOrder` và REST: [api-contract.md](api-contract.md), mục 8.
  - `data` của sự kiện hủy và bước nhả hàng: [events.md](events.md), mục 2.

## 6. Checklist chấm

Người chấm chạy lại bài từ `README.md`, rà theo danh sách dưới, rồi hỏi 15–20 phút về code. **Một mục Bắt buộc trượt thì bài chưa đạt.**

### Chung (5)

| # | Nhãn | Mục | Định nghĩa tại |
|---|---|---|---|
| 1 | **BẮT BUỘC** | Bài nộp ghi mã AC đã đạt và lệnh kiểm đã chạy, kèm kết quả thật | [../conventions.md](../conventions.md), mục 11; bảng kết quả: [README.md](../../README.md), mục 6 |
| 2 | **BẮT BUỘC** | Giải thích được code mình nộp khi được hỏi | [../conventions.md](../conventions.md), mục 11, 12 |
| 3 | **BẮT BUỘC** | Không commit mật khẩu, file `.env`; cấu hình đọc từ biến môi trường | [../conventions.md](../conventions.md), mục 9 |
| 4 | **BẮT BUỘC** | File ≤ 500 dòng, hàm ≤ 60 dòng, component React ≤ 200 dòng | [../conventions.md](../conventions.md), mục 3 |
| 5 | **NÊN** | Tên rõ nghĩa, comment giải thích **vì sao** chứ không kể lại code; không để code bị comment | [../conventions.md](../conventions.md), mục 2 |

### Go (5)

| # | Nhãn | Mục | Định nghĩa tại |
|---|---|---|---|
| 6 | **BẮT BUỘC** | `gofmt`, `go vet`, `go test ./...` sạch | [../conventions.md](../conventions.md), mục 2 |
| 7 | **BẮT BUỘC** | `domain` không import pgx, grpc, nats; adapter gRPC không chứa quy tắc nghiệp vụ | [../conventions.md](../conventions.md), mục 1 |
| 8 | **BẮT BUỘC** | Kiểm quyền theo tên quyền, không theo tên người hay vai trò | [../conventions.md](../conventions.md), mục 5 |
| 9 | **BẮT BUỘC** | Tiền không đi qua `float64` ở bất kỳ đâu | [../conventions.md](../conventions.md), mục 4 |
| 10 | **BẮT BUỘC** | Mọi lời gọi gRPC và NATS có timeout; goroutine có đường dừng khi tắt server | [../conventions.md](../conventions.md), mục 2 |

### CSDL (4)

| # | Nhãn | Mục | Định nghĩa tại |
|---|---|---|---|
| 11 | **BẮT BUỘC** | Mọi bảng có `tenant_id` bật `ENABLE` và `FORCE ROW LEVEL SECURITY` | [../conventions.md](../conventions.md), mục 6; [data-model.md](data-model.md), mục 5 |
| 12 | **BẮT BUỘC** | Service chạy bằng `mini_app`, không phải `postgres`; `app.tenant_id` đặt bằng `set_config(..., true)` trong giao dịch | [../conventions.md](../conventions.md), mục 6 |
| 13 | **BẮT BUỘC** | Truy vấn qua sqlc, không ghép chuỗi SQL | [../conventions.md](../conventions.md), mục 6 |
| 14 | **BẮT BUỘC** | Migration đã merge thì không sửa; muốn đổi thì thêm file mới | [../conventions.md](../conventions.md), mục 6 |

### Sự kiện và nghiệp vụ (4)

| # | Nhãn | Mục | Định nghĩa tại |
|---|---|---|---|
| 15 | **BẮT BUỘC** | Outbox ghi cùng giao dịch với thay đổi nghiệp vụ; không chỗ nào trong usecase gọi NATS | [../conventions.md](../conventions.md), mục 7; [events.md](events.md), mục 4 |
| 16 | **BẮT BUỘC** | Inbox ghi cùng giao dịch xử lý; ack sau khi commit | [../conventions.md](../conventions.md), mục 7; [events.md](events.md), mục 5 |
| 17 | **BẮT BUỘC** | Giữ hàng khóa dòng (`FOR UPDATE`, theo thứ tự `sku`) trước khi đọc tồn; tất cả hoặc không gì cả | [../conventions.md](../conventions.md), mục 7; [events.md](events.md), mục 7 |
| 18 | **BẮT BUỘC** | Chặn tự duyệt ở usecase, không chỉ ẩn nút trên web | [../conventions.md](../conventions.md), mục 5 |

### BFF và web (4)

| # | Nhãn | Mục | Định nghĩa tại |
|---|---|---|---|
| 19 | **BẮT BUỘC** | BFF không có CSDL, không tính tổng, không tự quyết quyền | [../conventions.md](../conventions.md), mục 8 |
| 20 | **BẮT BUỘC** | Trang không `fetch` trực tiếp, chỉ gọi hàm trong `lib/data` | [../conventions.md](../conventions.md), mục 8 |
| 21 | **BẮT BUỘC** | Nút theo `allowedActions`; tiền qua `formatMoney()` | [../conventions.md](../conventions.md), mục 4, 8; [web.md](web.md), mục 5 |
| 22 | **NÊN** | Server Component mặc định, `'use client'` đặt sâu nhất có thể | [../conventions.md](../conventions.md), mục 8 |

### Hạ tầng (2)

| # | Nhãn | Mục | Định nghĩa tại |
|---|---|---|---|
| 23 | **BẮT BUỘC** | Cổng compose chỉ mở trên `127.0.0.1`; image ghim phiên bản | [../tech-stack.md](../tech-stack.md), mục 3, 5 |
| 24 | **BẮT BUỘC** | `README.md` của repo có các lệnh đi từ máy trống tới lúc demo được | [../tech-stack.md](../tech-stack.md), mục 5 |

Tổng: 24 dòng (22 BẮT BUỘC, 2 NÊN).

## 7. Kết quả thật khi nộp

Bảng kết quả (AC, đạt/không, lệnh đã chạy, ngày chạy) nằm ở [README.md](../../README.md), mục 6, để người chấm thấy ngay khi chạy lại bài. Cột "Lệnh đã chạy" ghi đúng lệnh đã gõ, và đính kèm output thật.

## 8. Câu hỏi người chấm có thể hỏi

Đề lấy hai ví dụ: "bỏ `FOR UPDATE` thì chuyện gì xảy ra?" và "vì sao không gửi NATS ngay trong usecase?". Bảy câu hỏi và câu trả lời theo lập luận của đề: xem [../conventions.md](../conventions.md), mục 12 "Quyết định thiết kế".
