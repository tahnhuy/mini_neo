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
| protoc-gen-go, protoc-gen-go-grpc (plugin của `buf generate`) | Theo [docs/tech-stack.md](docs/tech-stack.md), mục 3 |

Lệnh cài các công cụ Go đúng phiên bản: xem [docs/tech-stack.md](docs/tech-stack.md), mục 3.

## 3. Từ máy trống đến demo

Chạy các lệnh tại thư mục gốc repo, trừ bước web có `cd web`. Cài công cụ theo mục 2 trước khi bắt đầu.

**1. Clone repo**

```bash
git clone https://github.com/tahnhuy/mini_neo.git
cd mini_neo
```

**2. Sao chép cấu hình mẫu.** Cấu hình đọc từ biến môi trường. Sao chép `.env.example` thành `.env`, và không commit `.env` ([docs/conventions.md](docs/conventions.md), mục 9).

```bash
cp .env.example .env
# Đổi mọi giá trị doi-mat-khau-* trong .env, kể cả mật khẩu nằm trong các URL.
```

> **Gợi ý (ngoài đề):** Đề không nhắc tới `.env.example`. Đây là cách thường dùng để ghi tên các biến môi trường cần có mà không commit giá trị thật.

**3. Chạy PostgreSQL và NATS** (`deploy/compose.yaml`, [docs/tech-stack.md](docs/tech-stack.md), mục 5)

```bash
docker compose --env-file .env -f deploy/compose.yaml up -d --wait
```

**4. Tạo vai trò CSDL** `mini_owner`, `mini_app`, `mini_relay` ([docs/specs/data-model.md](docs/specs/data-model.md), mục 6)

```bash
# Tự động: deploy/initdb/01-roles.sh chạy ở lần khởi động đầu (volume còn trống),
# tạo ba vai trò và CSDL "mini" thuộc mini_owner. Kiểm lại:
docker compose --env-file .env -f deploy/compose.yaml exec postgres \
  psql -U postgres -c "select rolname, rolbypassrls, rolcreatedb from pg_roles where rolname like 'mini_%'"
```

**5. Chạy migration** bằng goose, với vai trò chủ bảng ([docs/specs/data-model.md](docs/specs/data-model.md), mục 6)

```bash
set -a; . ./.env; set +a   # nạp biến môi trường từ .env vào shell hiện tại
goose -dir services/order/migrations -table goose_mini_order postgres "$MIGRATE_DATABASE_URL" up
goose -dir services/stock/migrations -table goose_mini_stock postgres "$MIGRATE_DATABASE_URL" up
```

**6. Nạp dữ liệu mẫu** ([docs/specs/business-rules.md](docs/specs/business-rules.md), mục 4)

```bash
# Không có lệnh riêng: dữ liệu mẫu là migration services/stock/migrations/00002_seed.sql,
# đã chạy ở bước 5.
```

**7. Sinh code** bằng `buf generate` (proto, [docs/specs/api-contract.md](docs/specs/api-contract.md), mục 7) và `sqlc generate` (truy vấn). Mã sinh từ proto đã được commit vào `gen/go/`; bước này chỉ cần làm khi sửa proto hoặc query.

```bash
buf generate
(cd services/stock && sqlc generate)
(cd services/order && sqlc generate)
```

**8. Chạy `stock`, `order`, `bff`**

```bash
set -a; . ./.env; set +a
go run ./services/stock/cmd/server   # cửa sổ terminal 1
go run ./services/order/cmd/server   # cửa sổ terminal 2 (nạp .env như trên)
go run ./services/bff/cmd/server     # cửa sổ terminal 3 (nạp .env như trên)
```

**9. Chạy web**

```bash
set -a; . ./.env; set +a
cd web
pnpm install --frozen-lockfile
pnpm build
pnpm start
```

**10. Mở trang** `/` của web, chọn một người dùng mẫu (mục 5).

```text
http://127.0.0.1:3000
```

## 4. Lệnh kiểm

| Kiểm | Lệnh |
|---|---|
| Định dạng | `gofmt -l .` (không in ra file nào là sạch) |
| Phân tích tĩnh | `go vet ./...` |
| Unit test | `go test ./...` |
| Test tích hợp | `go test -tags integration ./...` (điều kiện: [docs/specs/acceptance.md](docs/specs/acceptance.md), mục 3) |
| Lint proto | `buf lint` |
| Kiểu web | `cd web && pnpm typecheck` |
| Build web | `cd web && pnpm build` |

Trước khi chạy test tích hợp: PostgreSQL phải đang chạy (bước 3) và nạp `.env` (`set -a; . ./.env; set +a`). Test tự tạo và xóa CSDL riêng bằng `mini_owner`, chạy truy vấn bằng `mini_app`. Test tích hợp của sự kiện (AC-05 đến AC-08) cần NATS riêng cho test (Q36): `docker compose --env-file .env -f deploy/compose.yaml --profile test up -d --wait nats-test`.

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

Kết quả các ngày trước được giữ nguyên; kiểm Ngày 4 ghi bên dưới. Cột "Lệnh đã chạy" ghi đúng lệnh đã gõ; đính kèm output thật. Quy tắc nộp bài: xem [docs/conventions.md](docs/conventions.md), mục 11. AC-08 cần cả lần chạy lặp lẫn lần đối chứng âm ([docs/specs/acceptance.md](docs/specs/acceptance.md), mục 4).

| AC | Đạt / Không | Lệnh đã chạy | Ngày chạy |
|---|---|---|---|
| AC-01 (phần tổng tiền và giá từ kho) | Đạt | `go test -count=1 -run AC01 -v ./services/order/...` và `go test -count=1 -tags integration -run AC01 -v ./services/order/...` | 2026-10-26 |
| AC-02 (phần `PERMISSION_DENIED`, `allowed_actions` rỗng) | Đạt | `go test -count=1 -run AC02 -v ./services/order/...` | 2026-10-26 |
| AC-03 | Đạt | `go test -count=1 -run AC03 -v ./services/order/...` | 2026-10-26 |
| AC-04 (SQL) | Đạt | `go test -count=1 -tags integration -run AC04 -v ./services/order/...` | 2026-10-26 |
| AC-04 (API) | Đạt | `go test -count=1 -tags integration -run AC04 -v ./services/order/...` | 2026-10-26 |
| AC-05 | Đạt | `go test -count=1 -tags integration -run AC05 -v ./services/order/...` | 2026-10-26 |
| AC-06 | Đạt | `go test -count=1 -tags integration -run AC06 -v ./services/order/...` | 2026-10-26 |
| AC-07 | Đạt | `go test -count=1 -tags integration -run AC07 -v ./services/order/...` | 2026-10-26 |
| AC-08 (≥ 20 lần, có khóa) | Đạt: 20/20 xanh, chạy 4 lượt | `go test -tags integration -run AC08 -count=20 -v ./services/order/...` | 2026-10-26 |
| AC-08 (đối chứng âm, bỏ khóa: phải đỏ) | Đạt: đỏ 20/20, chạy 3 lượt; lỗi `trạng thái hai đơn = map[RESERVED:2]` | Bỏ `FOR UPDATE` trong `LockStockLevels` (`services/stock/queries/events.sql`), `sqlc generate`, rồi chạy lệnh như dòng trên; xong thì khôi phục | 2026-10-26 |
| AC-09 | Đạt | `go test -count=1 -run AC09 -v ./services/order/...` | 2026-10-26 |

| Lệnh kiểm (mục 4) | Đạt / Không | Lệnh đã chạy | Ngày chạy |
|---|---|---|---|
| `gofmt` | Đạt | `gofmt -l .` | 2026-10-26 |
| `go vet` | Đạt | `GOCACHE=/tmp/neo-go-cache go vet ./...` | 2026-10-26 |
| `go test ./...` | Đạt | `GOCACHE=/tmp/neo-go-cache go test -count=1 ./...` | 2026-10-26 |
| `go test -tags integration ./...` | Đạt | `GOCACHE=/tmp/neo-go-cache go test -count=1 -tags integration ./...` | 2026-10-26 |
| `buf lint` | Đạt | `/home/nhathuy/go/bin/buf lint` | 2026-10-26 |

### Kiểm Ngày 4

- AC-01 (client gửi giá): `GOCACHE=/tmp/neo-go-cache go test ./services/bff/...` đạt; handler bỏ trường giá và chuyển đúng dòng sang gRPC.
- Unit test toàn repo: output các package có test đều `ok`; BFF: `ok github.com/tahnhuy/mini_neo/services/bff/internal/adapter/http 0.004s`.
- Test tích hợp chạy lại: `ok` cho adapter gRPC, repo, app của order và adapter gRPC stock; exit code 0.
- `gofmt -l .`, `go vet ./...`, `buf lint`: không có output, exit code 0.

> **Gợi ý (ngoài đề):** `GOCACHE=/tmp/neo-go-cache` chỉ dùng cho môi trường kiểm có thư mục cache mặc định chỉ đọc. Máy cá nhân dùng lệnh ở mục 4. Các lỗi gRPC không có ánh xạ trong đề trả HTTP 500 theo gợi ý của [api-contract.md](docs/specs/api-contract.md), mục 4.

- Web: `COREPACK_HOME=/tmp/neo-corepack pnpm install --frozen-lockfile --offline --store-dir /tmp/neo-pnpm-store`, `COREPACK_HOME=/tmp/neo-corepack pnpm typecheck`, `COREPACK_HOME=/tmp/neo-corepack pnpm build`: exit code 0. Output build:

```text
✓ Compiled successfully in 4.2s
  Finished TypeScript in 1247ms ...
✓ Generating static pages using 7 workers (5/5) in 409ms
Route (app)
┌ ○ /
├ ○ /_not-found
├ ƒ /orders
├ ƒ /orders/[id]
└ ƒ /orders/new
```

Kiểm HTTP thực tế trên bản production (không phải Playwright / EX-04), ngày 2026-10-26:

```text
AC-01 REST: giá client bị bỏ qua, tổng 2100000.00; AC-02 HTML SSR: an không có nút Duyệt/Từ chối; AC-04 REST: tenant B nhận 404
4 route SSR đạt; chi có nút theo allowedActions; duyệt qua REST + sự kiện thật → RESERVED; trang tenant B hiện không tìm thấy
REST: SELF_APPROVAL → 403; từ chối → 200; INVALID_TRANSITION → 409
Server Actions qua form HTML: chọn an → cookie dev_user → /orders; tạo GHE-01 → chi tiết, tổng 850.000 ₫, không nút Duyệt: đạt
```

**Bổ sung ngoài đề (Q14, Q15, Q29, Q32, Q33, Q34, Q38):** BFF và web triển khai các quyết định này theo [hợp đồng](docs/specs/api-contract.md) và [web](docs/specs/web.md), không thêm endpoint. Không làm EX trong Ngày 4.

**Gợi ý (ngoài đề):** Build dùng webpack do worker Turbopack bị môi trường kiểm chặn mở cổng nội bộ. TypeScript và các gói kiểu là công cụ build Next.js, không thêm thư viện nghiệp vụ. Theme dùng các token đã ghi ở [frontend.md](docs/specs/frontend.md#tokens). Lần kiểm ban đầu chưa dùng trình duyệt; kết quả cập nhật theo Figma ghi bên dưới.

### Cập nhật giao diện theo Figma

**Nguồn:** [Figma NEO Mini - Web](https://www.figma.com/design/VuykjXk2pbaOR9PlOGZtLv/Untitled), các frame desktop/mobile của bốn route; [đặc tả frontend](docs/specs/frontend.md). **Bổ sung ngoài đề (giao diện Figma):** font Inter lưu cục bộ kèm giấy phép, header tài khoản/đổi người dùng, thẻ mobile, màu trạng thái, skeleton, form mở/xác nhận/hủy từ chối. Không đổi hợp đồng hoặc thêm nghiệp vụ.

Đã chạy `pnpm typecheck`, `pnpm build` thành công. Kiểm Chromium thực tế ở 1440px và 390px, chụp và xem ảnh đối chiếu; kết quả:

```text
PASS 1440 responsive layouts, SVG 6×6, reject open/cancel, Unicode count, allowedActions
PASS 390 responsive layouts, SVG 6×6, reject open/cancel, Unicode count, allowedActions
PASS real browser: create, reject, approve → RESERVED, tenant isolation
PASS no browser errors
```

Cả năm SVG trạng thái tải từ Figma đã lưu trong `web/public/status`, kiểm file không rỗng và ảnh render đúng 6×6. Không thêm dependency ứng dụng; Playwright dùng tạm ngoài repo để kiểm, chưa coi là bộ test EX-04 được nộp.

Các điểm lệch dữ liệu thiết kế đã biết vẫn áp dụng quyết định của bài: mặt hàng theo Q39; chi tiết đơn chỉ có SKU theo Q25; không thêm nhãn “Sắp hết” khi chưa có quy tắc ngưỡng (UI-05, UI-07, UI-08 trong docs frontend). Không phát sinh điểm CẦN XÁC NHẬN mới.

### Rà 24 dòng checklist mục 10

Thứ tự dòng giữ theo [checklist chấm](docs/specs/acceptance.md); bảng này chỉ ghi bằng chứng, không định nghĩa lại quy tắc.

| Nhóm / dòng | Kết quả rà Ngày 4 |
|---|---|
| Chung 1 | Có bảng AC, lệnh và output thật ở trên |
| Chung 2 | Cần người nộp tự giải thích code khi chấm; không thể xác nhận thay người nộp |
| Chung 3 | `.env` bị ignore; cấu hình BFF/web đọc môi trường; chỉ nộp `.env.example` |
| Chung 4 | File code viết tay ≤ 500 dòng; hàm BFF < 40 dòng; component React ≤ 200 dòng; mã sinh theo công cụ |
| Chung 5 (NÊN) | Tên theo nghiệp vụ; comment giải thích lựa chọn; không có code bị comment trong phần mới |
| Go 1 | `gofmt`, `go vet`, unit test đạt |
| Go 2 | Domain và các adapter cũ giữ cấu trúc; BFF chỉ chuyển giao thức |
| Go 3 | Quyền nghiệp vụ ở usecase; web chỉ kiểm capability tạo đơn theo đề |
| Go 4 | Tiền chuyển nguyên chuỗi; `formatMoney()` dùng `BigInt`, không qua số thực |
| Go 5 | BFF timeout gRPC 5 giây, web 7 giây; server dừng bằng signal và Shutdown |
| CSDL 1–4 | Migration/RLS/sqlc giữ nguyên; test tích hợp chạy lại đạt; không sửa migration |
| Sự kiện và nghiệp vụ 1–4 | Code ngày 3 giữ nguyên; AC-05–08 chạy lại đạt; bằng chứng AC-08 lặp và đối chứng âm ở bảng trên |
| BFF và web 1 | BFF không có CSDL hoặc phép tính tổng/quyền nghiệp vụ |
| BFF và web 2 | `fetch` chỉ trong `web/src/lib/data/index.ts` |
| BFF và web 3 | Hai nút theo `allowedActions`; mọi tiền hiển thị qua `formatMoney()` |
| BFF và web 4 (NÊN) | Trang là Server Component; Client chỉ form/nút và error boundary |
| Hạ tầng 1 | Compose giữ image ghim và cổng loopback |
| Hạ tầng 2 | Mục 3 đã có lệnh chạy đủ stock, order, BFF và web |

Bài nộp được chuẩn bị trong working tree và đóng gói `neo-mini-day4.tar.gz` (không gồm `.env`, dependency cài sẵn hoặc build cache); chưa push hoặc gửi tới người chấm. Lịch sử ngày 1–3 đã có các commit riêng; thay đổi Ngày 4 chưa commit. Không đánh dấu hai việc này là đã làm.

## 7. Mục lục docs

| File | Nội dung |
|---|---|
| [docs/source/neo-mini.md](docs/source/neo-mini.md) | Đề bài (nguồn sự thật, không sửa) |
| [AGENTS.md](AGENTS.md) | Quy tắc viết docs và các quyết định đã chốt (Q1–Q45) |
| [docs/tech-stack.md](docs/tech-stack.md) | Kiến trúc, thành phần, phiên bản ghim, cấu trúc repo, hạ tầng |
| [docs/conventions.md](docs/conventions.md) | Quy tắc code, tiền, kiểm quyền, CSDL, web, test, Git; quyết định thiết kế |
| [docs/specs/business-rules.md](docs/specs/business-rules.md) | Luồng nghiệp vụ, trạng thái đơn, người dùng và dữ liệu mẫu |
| [docs/specs/data-model.md](docs/specs/data-model.md) | Bảng, cột, RLS, vai trò CSDL |
| [docs/specs/api-contract.md](docs/specs/api-contract.md) | gRPC, REST của BFF, mã lỗi, danh tính |
| [docs/specs/events.md](docs/specs/events.md) | Sự kiện, phong bì, outbox, inbox, JetStream, giữ hàng |
| [docs/specs/web.md](docs/specs/web.md) | Bốn màn web, trạng thái, nút hiển thị |
| [docs/specs/acceptance.md](docs/specs/acceptance.md) | AC, EX, checklist chấm |
| [docs/specs/tasks.md](docs/specs/tasks.md) | Kế hoạch 4 ngày, thứ tự ưu tiên, nộp bài |
