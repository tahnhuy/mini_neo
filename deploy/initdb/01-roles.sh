#!/bin/sh
# Tạo ba vai trò và CSDL "mini" (docs/specs/data-model.md, mục 6).
# Chạy một lần bởi image postgres, bằng superuser, khi volume còn trống.
# Bảng và schema do migration tạo (chạy bằng mini_owner), không tạo ở đây.
set -eu

psql -v ON_ERROR_STOP=1 \
  --username "$POSTGRES_USER" --dbname postgres \
  -v owner_pw="$MINI_OWNER_PASSWORD" \
  -v app_pw="$MINI_APP_PASSWORD" \
  -v relay_pw="$MINI_RELAY_PASSWORD" <<'SQL'
-- Chủ bảng, chạy migration. CREATEDB để test tích hợp tự tạo và xóa CSDL riêng
-- mà không cần superuser.
CREATE ROLE mini_owner LOGIN CREATEDB PASSWORD :'owner_pw';

-- Vai trò thường của service: không superuser, không chủ bảng, chịu RLS.
CREATE ROLE mini_app LOGIN PASSWORD :'app_pw';

-- Relay đọc outbox của mọi tenant nên cần BYPASSRLS; quyền chỉ cấp trên outbox.
CREATE ROLE mini_relay LOGIN BYPASSRLS PASSWORD :'relay_pw';

CREATE DATABASE mini OWNER mini_owner;
SQL
