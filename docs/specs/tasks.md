# Kế hoạch 4 ngày NEO Mini

> **Mục đích:** Checklist việc cần làm theo từng ngày, kèm link spec và mã AC cần đạt cuối mỗi ngày. Đây là nơi duy nhất định nghĩa kế hoạch ngày và thứ tự ưu tiên.
>
> **Đối tượng đọc:** Người làm bài (theo dõi tiến độ hằng ngày), người chấm bài.
>
> **Nguồn:** mục 8, 9 của đề (thêm mục 10 cho phần nộp bài); quyết định Q2, Q13, Q36.

Mỗi người làm cả bài trong repo riêng của mình. Được dùng trợ lý AI, nhưng phải giải thích được từng dòng mình nộp ([../conventions.md](../conventions.md), mục 11). Nội dung từng AC: xem [acceptance.md](acceptance.md), mục 1.

## Thứ tự ưu tiên

Nguyên văn mục 8 của đề:

> Thứ tự ưu tiên khi thiếu thời gian: phần Go và test (ngày 1–3) quan trọng hơn web. Web xấu nhưng đúng luật mục 4.8 vẫn đạt; web đẹp mà thiếu AC-07, AC-08 thì không đạt.
>
> Ngày 1 đáng đầu tư nhất: proto và bảng sai thì ngày 2–3 phải sửa lại cả chuỗi.

## Ngày 1: proto, CSDL, hạ tầng, stock `ListItems`

AC cần đạt: **AC-04 (mức SQL, Q2)**.

- [x] Viết proto cho `OrderService`, `StockService`; `buf lint` sạch; commit mã sinh vào `gen/go/` ([api-contract.md](api-contract.md), mục 1, 2, 7)
- [x] Viết `deploy/compose.yaml` cho PostgreSQL và NATS; `compose up` chạy được ([../tech-stack.md](../tech-stack.md), mục 3, 5)
- [x] Viết migration cho `mini_order` và `mini_stock` ([data-model.md](data-model.md), mục 2, 3, 4)
- [x] Tạo vai trò `mini_app`, `mini_owner`, `mini_relay` ([data-model.md](data-model.md), mục 6)
- [x] Bật RLS cho mọi bảng ([data-model.md](data-model.md), mục 5)
- [x] Nạp dữ liệu mẫu: 2 tenant, mặt hàng mẫu ([business-rules.md](business-rules.md), mục 4)
- [x] Viết `pkg/db` với `db.InTx` ([../conventions.md](../conventions.md), mục 6; [../tech-stack.md](../tech-stack.md), mục 6)
- [x] Service stock trả `ListItems` qua gRPC, kèm interceptor đọc metadata danh tính ([api-contract.md](api-contract.md), mục 1, 6)
- [x] Test AC-04 ở mức SQL bằng `mini_app` ([acceptance.md](acceptance.md), mục 2)

## Ngày 2: service order

AC cần đạt: **AC-01, AC-02, AC-03, AC-09** (unit test); **AC-04 (mức API, Q2)**.

- [x] `CreateOrder`: gọi `StockService.GetItems` lấy giá, sinh mã đơn ([api-contract.md](api-contract.md), mục 1, 2; [data-model.md](data-model.md), mục 2)
- [x] `GetOrder` kèm `allowed_actions` ([api-contract.md](api-contract.md), mục 1)
- [x] `ListOrders` ([api-contract.md](api-contract.md), mục 1, 2)
- [x] `ApproveOrder`, `RejectOrder` với bảng chuyển trạng thái ở `domain` ([business-rules.md](business-rules.md), mục 3; [api-contract.md](api-contract.md), mục 3)
- [x] Unit test AC-01, AC-02, AC-03, AC-09 ([acceptance.md](acceptance.md), mục 1)
- [x] Test AC-04 ở mức API (`GetOrder` trả `NOT_FOUND`) ([acceptance.md](acceptance.md), mục 2)

## Ngày 3: outbox, relay, inbox, giữ hàng

AC cần đạt: **AC-05, AC-06, AC-07, AC-08** (test tích hợp).

**Ghi chú Q13:** Mục 8 của đề ghi ngày 3 là "AC-05 đến AC08". Câu này được hiểu là AC-05 đến AC-08; file nguồn không sửa.

- [ ] Ghi outbox trong cùng giao dịch với thay đổi nghiệp vụ ([events.md](events.md), mục 4)
- [ ] Relay: goroutine, chu kỳ khoảng 500 ms, dừng khi tắt server ([events.md](events.md), mục 4)
- [ ] Stream `MINI` và các durable consumer ([events.md](events.md), mục 6)
- [ ] Inbox: 4 bước trong một giao dịch, ack sau commit ([events.md](events.md), mục 5)
- [ ] Stock giữ hàng cho cả đơn, khóa `FOR UPDATE` theo `sku` ([events.md](events.md), mục 7)
- [ ] Order nhận kết quả, chuyển đơn sang `RESERVED` hoặc `OUT_OF_STOCK`; bỏ qua sự kiện đến muộn ([events.md](events.md), mục 8, 9)
- [ ] Test tích hợp AC-05 đến AC-08, dùng CSDL và NATS riêng cho test ([acceptance.md](acceptance.md), mục 1, 3)
- [ ] Chạy AC-08 ít nhất 20 lần liền và làm đối chứng âm ([acceptance.md](acceptance.md), mục 4)

## Ngày 4: BFF, web, README, nộp

AC cần đạt: phần web của AC-02 (không hiện nút Duyệt cho `an`, [acceptance.md](acceptance.md), mục 1); rà lại toàn bộ AC.

- [ ] BFF: 7 endpoint REST, gắn danh tính, ánh xạ lỗi sang HTTP ([api-contract.md](api-contract.md), mục 4, 5, 6)
- [ ] Bốn màn web ([web.md](web.md), mục 2, 3, 5)
- [ ] `README.md` có lệnh chạy từ máy trống tới demo ([README.md](../../README.md))
- [ ] Tự rà theo checklist mục 10 ([acceptance.md](acceptance.md), mục 6)
- [ ] Nộp bài (mục "Nộp bài" bên dưới)

## Làm thêm

Chỉ làm khi đã xong ngày 1–4. Nội dung từng EX: xem [acceptance.md](acceptance.md), mục 5.

- [ ] EX-01
- [ ] EX-02
- [ ] EX-03
- [ ] EX-04
- [ ] EX-05

## Nộp bài

Quy tắc: xem [../conventions.md](../conventions.md), mục 11.

- [ ] Gửi link repo (hoặc file nén)
- [ ] Ghi danh sách mã AC đã đạt và kết quả thật của các lệnh kiểm vào bảng ở [README.md](../../README.md), mục 6
- [ ] Lịch sử commit nhỏ theo từng bước, không dồn một commit cuối ngày 4
