# NEO Mini: Đặt hàng và Giữ kho

> **Mục đích:** Giới thiệu bài thực hành, liệt kê các bước từ máy trống tới demo, lệnh kiểm và bảng kết quả khi nộp bài.
>
> **Đối tượng đọc:** Người làm bài, người chấm bài (người chấm chạy lại bài theo file này).
>
> **Nguồn:** mục 1, 8, 10 của đề.

## 1. Giới thiệu

NEO Mini là bài thực hành 3–4 ngày, mỗi người tự làm toàn bộ bài một mình. Bài dựng lại kiến trúc của dự án NEO ở cỡ rất nhỏ: hai microservice Go (`order`, `stock`) nói chuyện bằng gRPC và sự kiện NATS, một BFF Go, một web Next.js, trên PostgreSQL. Nghiệp vụ là tạo đơn, duyệt đơn và giữ hàng trong kho, có cách ly dữ liệu giữa các công ty (tenant). Làm ít tính năng nhưng làm đúng: một luồng chạy đúng có test tốt hơn năm luồng chạy "gần đúng".

Luồng gọi: `web → bff → order → stock`, sự kiện đi qua NATS. Sơ đồ thành phần đầy đủ: xem [docs/tech-stack.md](docs/tech-stack.md), mục 1.

## 2. Yêu cầu máy

| Công cụ | Phiên bản |
|---|---|
| Go | Theo [docs/tech-stack.md](docs/tech-stack.md), mục 3 |
| Node.js, pnpm | Theo [docs/tech-stack.md](docs/tech-stack.md), mục 3 |
| Buf | Theo [docs/tech-stack.md](docs/tech-stack.md), mục 3 |
| sqlc | Theo [docs/tech-stack.md](docs/tech-stack.md), mục 3 |
| goose | Theo [docs/tech-stack.md](docs/tech-stack.md), mục 3 |
| Docker (chạy `deploy/compose.yaml`) | Đề không ghim |

## 3. Từ máy trống đến demo

Các lệnh dưới đây sẽ được điền khi có code. Không chạy lệnh nào chưa được điền.

**1. Clone repo**

```bash
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

**2. Sao chép cấu hình mẫu.** Cấu hình đọc từ biến môi trường. Sao chép `.env.example` thành `.env`, và không commit `.env` ([docs/conventions.md](docs/conventions.md), mục 9).

```bash
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

> **Gợi ý (ngoài đề):** Đề không nhắc tới `.env.example`. Đây là cách thường dùng để ghi tên các biến môi trường cần có mà không commit giá trị thật.

**3. Chạy PostgreSQL và NATS** (`deploy/compose.yaml`, [docs/tech-stack.md](docs/tech-stack.md), mục 5)

```bash
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

**4. Tạo vai trò CSDL** `mini_owner`, `mini_app`, `mini_relay` ([docs/specs/data-model.md](docs/specs/data-model.md), mục 6)

```bash
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

**5. Chạy migration** bằng goose, với vai trò chủ bảng ([docs/specs/data-model.md](docs/specs/data-model.md), mục 6)

```bash
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

**6. Nạp dữ liệu mẫu** ([docs/specs/business-rules.md](docs/specs/business-rules.md), mục 4)

```bash
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

**7. Sinh code** bằng `buf generate` (proto, [docs/specs/api-contract.md](docs/specs/api-contract.md), mục 7) và `sqlc generate` (truy vấn). Mã sinh từ proto đã được commit vào `gen/go/`; bước này chỉ cần làm khi sửa proto hoặc query.

```bash
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

**8. Chạy `stock`, `order`, `bff`**

```bash
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

**9. Chạy web**

```bash
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

**10. Mở trang** `/` của web, chọn một người dùng mẫu (mục 5).

```text
<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>
```

## 4. Lệnh kiểm

| Kiểm | Lệnh |
|---|---|
| Định dạng | `gofmt -l .` (không in ra file nào là sạch) |
| Phân tích tĩnh | `go vet ./...` |
| Unit test | `go test ./...` |
| Test tích hợp | `go test -tags integration ./...` (điều kiện: [docs/specs/acceptance.md](docs/specs/acceptance.md), mục 3) |
| Lint proto | `buf lint` |

Biến môi trường và bước dựng CSDL, NATS cho test tích hợp: `<LỆNH CẦN ĐIỀN SAU KHI CÓ CODE>`.

`go test ./...`, `go test -tags integration ./...` và `buf lint` là nguyên văn của đề. Đề chỉ ghi tên `gofmt` và `go vet`; cờ `-l .` và đối số `./...` là cách gọi chuẩn của hai công cụ này, không phải script của repo.

## 5. Người dùng mẫu và luồng demo

Bốn người dùng `an`, `binh`, `chi`, `dung`, cùng tenant và quyền của họ: xem [docs/specs/business-rules.md](docs/specs/business-rules.md), mục 4.

Luồng demo đi theo các AC ([docs/specs/acceptance.md](docs/specs/acceptance.md), mục 1):

1. Chọn `an`, tạo đơn 2 dòng; tổng tiền hiện sau khi service trả về (AC-01). Ở chi tiết đơn, `an` không thấy nút Duyệt (AC-02).
2. Chọn `chi`, duyệt đơn đó; bấm "Làm mới" đến khi đơn sang `RESERVED` (AC-05).
3. Chọn `binh`, tạo một đơn rồi tự duyệt; web không hiện nút Duyệt, và nếu gọi API trực tiếp thì nhận lỗi `SELF_APPROVAL` (AC-03).
4. Tạo đơn cần nhiều hơn số còn lại của một mặt hàng, rồi cho `chi` duyệt; đơn sang `OUT_OF_STOCK` (AC-06).
5. Chọn `dung` (tenant B); danh sách không có đơn của tenant A, và mở đúng `id` đơn của tenant A thì báo không tìm thấy (AC-04).

## 6. Kết quả khi nộp

Điền khi nộp. Cột "Lệnh đã chạy" ghi đúng lệnh đã gõ; đính kèm output thật. Quy tắc nộp bài: xem [docs/conventions.md](docs/conventions.md), mục 11. AC-08 cần cả lần chạy lặp lẫn lần đối chứng âm ([docs/specs/acceptance.md](docs/specs/acceptance.md), mục 4).

| AC | Đạt / Không | Lệnh đã chạy | Ngày chạy |
|---|---|---|---|
| AC-01 | | | |
| AC-02 | | | |
| AC-03 | | | |
| AC-04 (SQL) | | | |
| AC-04 (API) | | | |
| AC-05 | | | |
| AC-06 | | | |
| AC-07 | | | |
| AC-08 (≥ 20 lần, có khóa) | | | |
| AC-08 (đối chứng âm, bỏ khóa: phải đỏ) | | | |
| AC-09 | | | |

| Lệnh kiểm (mục 4) | Đạt / Không | Lệnh đã chạy | Ngày chạy |
|---|---|---|---|
| `gofmt` | | | |
| `go vet` | | | |
| `go test ./...` | | | |
| `go test -tags integration ./...` | | | |
| `buf lint` | | | |

## 7. Mục lục docs

| File | Nội dung |
|---|---|
| [docs/source/neo-mini.md](docs/source/neo-mini.md) | Đề bài (nguồn sự thật, không sửa) |
| [AGENTS.md](AGENTS.md) | Quy tắc viết docs và các quyết định đã chốt (Q1–Q39) |
| [docs/tech-stack.md](docs/tech-stack.md) | Kiến trúc, thành phần, phiên bản ghim, cấu trúc repo, hạ tầng |
| [docs/conventions.md](docs/conventions.md) | Quy tắc code, tiền, kiểm quyền, CSDL, web, test, Git; quyết định thiết kế |
| [docs/specs/business-rules.md](docs/specs/business-rules.md) | Luồng nghiệp vụ, trạng thái đơn, người dùng và dữ liệu mẫu |
| [docs/specs/data-model.md](docs/specs/data-model.md) | Bảng, cột, RLS, vai trò CSDL |
| [docs/specs/api-contract.md](docs/specs/api-contract.md) | gRPC, REST của BFF, mã lỗi, danh tính |
| [docs/specs/events.md](docs/specs/events.md) | Sự kiện, phong bì, outbox, inbox, JetStream, giữ hàng |
| [docs/specs/web.md](docs/specs/web.md) | Bốn màn web, trạng thái, nút hiển thị |
| [docs/specs/acceptance.md](docs/specs/acceptance.md) | AC, EX, checklist chấm |
| [docs/specs/tasks.md](docs/specs/tasks.md) | Kế hoạch 4 ngày, thứ tự ưu tiên, nộp bài |
