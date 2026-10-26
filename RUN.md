# Hướng dẫn chạy NEO Mini bằng `run.sh`

> **Mục đích:** Chạy migration, các service (stock, order, bff) và web bằng một lệnh. Hạ tầng Docker bật riêng.
>
> **Đối tượng đọc:** Người làm bài, người chấm bài muốn dựng demo nhanh.
>
> **Nguồn:** [README.md](README.md), mục 3. Các bước thủ công vẫn theo README; file này chỉ mô tả script.

> **Gợi ý (ngoài đề):** `run.sh` là script tiện ích, đề không yêu cầu. Nó gom đúng các lệnh ở README mục 3 (bước 5, 8, 9), không thêm bước nghiệp vụ.

## 1. Chuẩn bị

Cài công cụ theo [README.md](README.md), mục 2 (phiên bản ở [docs/tech-stack.md](docs/tech-stack.md), mục 3). Script cần: `go`, `pnpm`, `goose`, `setsid` (có sẵn trên Linux).

`goose` cài bằng `go install` nằm ở `$(go env GOPATH)/bin`; script tự thêm thư mục này vào `PATH`.

Tạo cấu hình:

```bash
cp .env.example .env
# Đổi mọi giá trị doi-mat-khau-* trong .env, kể cả mật khẩu nằm trong các URL.
```

Nếu chưa có `.env`, script tự sao chép từ `.env.example` rồi dừng để bạn đổi mật khẩu.

## 2. Bật hạ tầng (tự chạy, script không làm)

PostgreSQL và NATS chạy bằng Docker, bật trước khi chạy script:

```bash
docker compose --env-file .env -f deploy/compose.yaml up -d --wait
```

Dừng hạ tầng (dữ liệu trong volume vẫn giữ):

```bash
docker compose --env-file .env -f deploy/compose.yaml down
```

Script chỉ kiểm cổng PostgreSQL (lấy từ `MIGRATE_DATABASE_URL`) và NATS (`NATS_URL`) đã mở chưa; chưa mở thì báo lỗi và dừng.

## 3. Chạy service và web

| Lệnh | Việc làm |
|---|---|
| `./run.sh` | Chạy tất cả, web ở chế độ production (`pnpm build` + `pnpm start`) |
| `./run.sh --dev` | Như trên, web chạy `pnpm dev` (tự tải lại khi sửa code) |

Khi thấy dòng `Mở http://127.0.0.1:3000`, mở trình duyệt và chọn người dùng mẫu (README mục 5).

Ctrl+C dừng stock, order, bff, web. Container Docker không bị ảnh hưởng.

## 4. Script làm gì

1. Nạp `.env`; kiểm đủ các biến `*_DATABASE_URL`; cảnh báo nếu còn mật khẩu mẫu.
2. Kiểm cổng của stock, order, bff, web còn trống.
3. Kiểm PostgreSQL và NATS đang chạy.
4. Chạy migration goose cho order và stock bằng `MIGRATE_DATABASE_URL` (vai trò `mini_owner`). Migration đã chạy rồi thì goose bỏ qua. Dữ liệu mẫu nằm trong migration `00002_seed.sql` của stock.
5. `go build` ba service vào `bin/` (đã có trong `.gitignore`).
6. Bật lần lượt stock → order → bff → web theo luồng gọi, chờ từng cổng mở mới bật tiếp.
7. In log của bốn service ra terminal; log lưu ở `bin/logs/<service>.log`.

Một service thoát giữa chừng thì script dừng toàn bộ.

## 5. Cổng và biến môi trường

| Thành phần | Biến | Mặc định |
|---|---|---|
| stock (gRPC) | `STOCK_GRPC_ADDR` | `127.0.0.1:50052` |
| order (gRPC) | `ORDER_GRPC_ADDR` | `127.0.0.1:50051` |
| bff (HTTP) | `BFF_HTTP_ADDR` | `127.0.0.1:8080` |
| web gọi bff | `BFF_URL` | `http://<BFF_HTTP_ADDR>` |
| web (HTTP) | `WEB_PORT` | `3000` |
| PostgreSQL | trong `MIGRATE_DATABASE_URL` | `127.0.0.1:5432` |
| NATS | `NATS_URL` | `nats://127.0.0.1:4222` |

Đổi cổng bằng cách sửa `.env`, ví dụ thêm `WEB_PORT=3010`. Mọi cổng chỉ mở trên `127.0.0.1`.

## 6. Xử lý lỗi thường gặp

| Thông báo | Cách xử lý |
|---|---|
| `PostgreSQL chưa chạy` / `NATS chưa chạy` | Bật hạ tầng theo mục 2. |
| `Cổng ... đang bị chiếm` | Còn tiến trình của lần chạy trước. Tìm bằng `ss -ltnp \| grep <cổng>` rồi dừng nó, hoặc đổi cổng trong `.env`. |
| `Thiếu công cụ 'goose'` | Cài goose theo [docs/tech-stack.md](docs/tech-stack.md), mục 3. |
| `Thiếu ... trong .env` | So `.env` với `.env.example`, thêm biến còn thiếu. |
| `<service> đã thoát, xem bin/logs/<service>.log` | Đọc log đó. Hay gặp: sai mật khẩu trong URL, CSDL chưa có vai trò. |
| Đăng nhập CSDL thất bại sau khi đổi mật khẩu | `deploy/initdb/01-roles.sh` chỉ chạy khi volume còn trống. Đổi mật khẩu sau lần đầu thì phải tạo lại volume: `docker compose --env-file .env -f deploy/compose.yaml down -v` (**xóa toàn bộ dữ liệu demo**), rồi bật lại hạ tầng. |
| `Corepack is about to download pnpm` | Bình thường ở lần đầu; corepack tải đúng bản pnpm ghim trong `web/package.json`. |

## 7. Chạy test

Script không chạy test. Lệnh kiểm và điều kiện test tích hợp: xem [README.md](README.md), mục 4.
