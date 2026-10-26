# Web NEO Mini

> **Mục đích:** Định nghĩa 4 màn web, nội dung từng màn, các trạng thái bắt buộc, và cách web chọn nút hiển thị. Quy tắc code web nằm ở `conventions.md`; file này chỉ link tới.
>
> **Đối tượng đọc:** Người làm bài (ngày 4: web), người chấm bài.
>
> **Nguồn:** mục 4.8, 6.2, 7 của đề; quyết định Q33, Q34, Q38.

REST mà web gọi qua BFF, body lỗi, cookie `dev_user`: xem [api-contract.md](api-contract.md). Trạng thái đơn: xem [business-rules.md](business-rules.md).

## 1. Quy tắc web

Nhãn, nguồn và nhóm checklist của từng quy tắc nằm ở [../conventions.md](../conventions.md). Bảng dưới chỉ là mục lục:

| Chủ đề | Định nghĩa tại |
|---|---|
| Server Component mặc định; `'use client'` chỉ ở component có tương tác, đặt sâu nhất có thể | [../conventions.md](../conventions.md), mục 8 |
| Trang không `fetch` trực tiếp; mọi lời gọi BFF nằm trong `web/src/lib/data/` | [../conventions.md](../conventions.md), mục 8 |
| Web không tính toán nghiệp vụ | [../conventions.md](../conventions.md), mục 8 |
| Nút theo `allowedActions` | [../conventions.md](../conventions.md), mục 8; cách áp từng màn ở mục 5 dưới đây |
| Tiền qua `formatMoney()`, kiểu `1.250.000 ₫` | [../conventions.md](../conventions.md), mục 4 |
| Giao diện tiếng Việt, chữ viết thẳng trong code | [../conventions.md](../conventions.md), mục 8 |
| Màu, khoảng cách khai một lần trong theme Tailwind; không hex hay `style=` rải rác | [../conventions.md](../conventions.md), mục 8 |
| Component React ≤ 200 dòng | [../conventions.md](../conventions.md), mục 3 |

## 2. Các màn

| Route | Nội dung |
|---|---|
| `/` | Chọn người dùng mẫu (`an`, `binh`, `chi`, `dung`), lưu cookie, chuyển sang `/orders` |
| `/orders` | Danh sách đơn: mã, người lập, tổng tiền, trạng thái; nút "Tạo đơn" nếu có `order.create` |
| `/orders/new` | Bảng mặt hàng (tên, đơn giá, khả dụng), nhập số lượng, bấm "Tạo đơn". Tổng tiền chỉ hiện sau khi service trả về |
| `/orders/[id]` | Chi tiết đơn, các dòng, nút Duyệt hoặc Từ chối theo `allowedActions`; ô nhập lý do khi từ chối. Đơn đang `APPROVED` thì có nút "Làm mới" |

- Cookie ở màn `/` là `dev_user` ([api-contract.md](api-contract.md), mục 6).
- Ở `/orders/new`, cột "khả dụng" hiển thị `available` do API trả; web không tự tính.
- Ở `/orders/new`, tổng tiền lấy từ response của `CreateOrder`; web không tự cộng.
- Nút "Làm mới" để xem kho đã trả lời chưa, vì đơn `APPROVED` đang chờ kho ([business-rules.md](business-rules.md), mục 2).

> **Gợi ý (ngoài đề):** Đề không nói sau khi tạo đơn xong thì đi đâu. Có thể chuyển sang `/orders/[id]` của đơn vừa tạo, nơi đã hiện tổng tiền do service trả về.

## 3. Trạng thái bắt buộc của mỗi màn

| Trạng thái | Hiển thị |
|---|---|
| Đang tải | Có chỉ báo đang tải |
| Rỗng | Có thông báo khi không có dữ liệu (ví dụ chưa có đơn nào) |
| Lỗi | Hiện `message` từ BFF (body lỗi: [api-contract.md](api-contract.md), mục 4) |

> **Bổ sung ngoài đề (Q34):** Màn `/` dùng danh sách 4 người dùng cố định trong code, không gọi BFF, nên không có trạng thái đang tải, rỗng hay lỗi. Ba trạng thái trên áp cho `/orders`, `/orders/new`, `/orders/[id]`.

> **Gợi ý (ngoài đề):** Với Next.js App Router, có thể dùng `loading.tsx` cho trạng thái đang tải và `error.tsx` cho lỗi không lường trước. Lỗi do BFF trả (có `code`, `message`) vẫn nên hiện ngay trong trang.

## 4. Hàm trong `web/src/lib/data/`

Mỗi endpoint REST của BFF ([api-contract.md](api-contract.md), mục 5) có một hàm trong `web/src/lib/data/`.

| Hàm | Endpoint | Nguồn |
|---|---|---|
| `getMe()` | `GET /api/me` | Q38 (**bổ sung ngoài đề**) |
| `listOrders()` | `GET /api/orders` | Ví dụ của đề |
| `createOrder()` | `POST /api/orders` | Q38 (**bổ sung ngoài đề**) |
| `getOrder()` | `GET /api/orders/{id}` | Q38 (**bổ sung ngoài đề**) |
| `approveOrder()` | `POST /api/orders/{id}/approve` | Ví dụ của đề |
| `rejectOrder()` | `POST /api/orders/{id}/reject` | Q38 (**bổ sung ngoài đề**) |
| `listStockItems()` | `GET /api/stock-items` | Q38 (**bổ sung ngoài đề**) |

## 5. Nút hiển thị

| Nút | Màn | Hiện khi | Nguồn |
|---|---|---|---|
| Duyệt | `/orders/[id]` | `allowedActions` có `"APPROVE"` | Mục 4.8, 7 của đề |
| Từ chối (kèm ô nhập lý do) | `/orders/[id]` | `allowedActions` có `"REJECT"` | Mục 4.8, 7 của đề |
| Tạo đơn | `/orders` | Người dùng có `order.create` | Mục 7 của đề |
| Làm mới | `/orders/[id]` | Đơn đang `APPROVED` | Mục 7 của đề |

- Duyệt và Từ chối chỉ theo `allowedActions`. Web không tự suy hai nút này từ quyền, trạng thái hay người lập ([../conventions.md](../conventions.md), mục 8).
- "Tạo đơn" và "Làm mới" là hai trường hợp đề ghi rõ điều kiện ở mục 7, không đi qua `allowedActions`.
- Quyền `order.create` lấy từ `capabilities` của `GET /api/me` (Q33). Web không tự tra bảng người dùng.
- Ẩn nút chỉ là hiển thị. Service vẫn chặn ở usecase ([../conventions.md](../conventions.md), mục 5).
