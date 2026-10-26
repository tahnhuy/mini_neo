#!/usr/bin/env bash
# Chạy các service và web của NEO Mini trên máy cục bộ theo README.md, mục 3:
#   migration → stock → order → bff → web.
#
# Script KHÔNG bật Docker. Bật PostgreSQL và NATS trước bằng lệnh riêng:
#   docker compose --env-file .env -f deploy/compose.yaml up -d --wait
#
# Cách dùng:
#   ./run.sh          chạy tất cả, web ở chế độ production (pnpm build + start)
#   ./run.sh --dev    như trên nhưng web chạy `pnpm dev`
#
# Ctrl+C dừng stock, order, bff, web.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

BIN_DIR="$ROOT/bin"          # /bin/ đã có trong .gitignore
LOG_DIR="$BIN_DIR/logs"
COMPOSE_HINT="docker compose --env-file .env -f deploy/compose.yaml up -d --wait"
PIDS=()

log() { printf '\033[1;34m[run]\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m[run] %s\033[0m\n' "$*" >&2; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || die "Thiếu công cụ '$1' (xem README.md, mục 2)."; }

# Tách "host:port" thành hai phần, rồi kiểm cổng có đang mở không.
port_open() { (exec 3<>"/dev/tcp/${1%:*}/${1##*:}") 2>/dev/null; }

wait_port() { # wait_port <tên> <host:port> <pid> [giây]
  local name=$1 addr=$2 pid=$3 timeout=${4:-60} i
  for ((i = 0; i < timeout * 2; i++)); do
    port_open "$addr" && { log "$name sẵn sàng tại $addr"; return 0; }
    kill -0 "$pid" 2>/dev/null || die "$name đã thoát, xem $LOG_DIR/$name.log"
    sleep 0.5
  done
  die "$name không mở cổng $addr sau ${timeout}s, xem $LOG_DIR/$name.log"
}

ensure_free() { # Tránh chạy chồng lên một lần chạy trước còn sống.
  port_open "$2" && die "Cổng $2 ($1) đang bị chiếm. Dừng tiến trình cũ trước."
  return 0
}

# Lấy "host:port" từ URL dạng scheme://[user:pass@]host:port/...
url_addr() { local x=${1#*://}; x=${x#*@}; echo "${x%%/*}"; }

# Hạ tầng do người dùng tự bật bằng docker compose; ở đây chỉ kiểm đã chạy chưa.
check_infra() {
  local pg nats
  pg=$(url_addr "$MIGRATE_DATABASE_URL")
  nats=$(url_addr "${NATS_URL:-nats://127.0.0.1:4222}")
  port_open "$pg" || die "PostgreSQL chưa chạy tại $pg. Bật bằng: $COMPOSE_HINT"
  port_open "$nats" || die "NATS chưa chạy tại $nats. Bật bằng: $COMPOSE_HINT"
  log "PostgreSQL ($pg) và NATS ($nats) đang chạy."
}

# Mỗi tiến trình chạy trong process group riêng (setsid) để dừng được cả tiến trình con
# (pnpm → next).
start() { # start <tên> <lệnh...>
  local name=$1; shift
  setsid "$@" >"$LOG_DIR/$name.log" 2>&1 &
  PIDS+=("$!")
}

cleanup() {
  trap - EXIT INT TERM
  [[ ${#PIDS[@]} -eq 0 ]] && return
  log "Đang dừng các service..."
  local pid
  for pid in "${PIDS[@]}"; do kill -TERM -- "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
  log "Đã dừng stock, order, bff, web."
}

load_env() {
  if [[ ! -f .env ]]; then
    cp .env.example .env
    die "Đã tạo .env từ .env.example. Đổi mọi giá trị doi-mat-khau-* trong .env rồi chạy lại."
  fi
  grep -q 'doi-mat-khau-' .env && log "Cảnh báo: .env còn mật khẩu mẫu doi-mat-khau-*."
  set -a; . ./.env; set +a
  # goose, buf, sqlc cài bằng `go install` nằm ở GOPATH/bin.
  export PATH="$PATH:$(go env GOPATH)/bin"
  # Biến không bắt buộc: mặc định giống giá trị mặc định trong cmd/server/main.go.
  export STOCK_GRPC_ADDR=${STOCK_GRPC_ADDR:-127.0.0.1:50052}
  export ORDER_GRPC_ADDR=${ORDER_GRPC_ADDR:-127.0.0.1:50051}
  export BFF_HTTP_ADDR=${BFF_HTTP_ADDR:-127.0.0.1:8080}
  export BFF_URL=${BFF_URL:-http://$BFF_HTTP_ADDR}
  # Cổng web (Next.js đọc PORT); mặc định 3000 như README.
  WEB_PORT=${WEB_PORT:-3000}
  export PORT=$WEB_PORT
  local key
  for key in MIGRATE_DATABASE_URL STOCK_DATABASE_URL STOCK_RELAY_DATABASE_URL \
    ORDER_DATABASE_URL ORDER_RELAY_DATABASE_URL; do
    [[ -n ${!key:-} ]] || die "Thiếu $key trong .env (xem .env.example)."
  done
}

migrate() {
  log "Chạy migration (order, stock) bằng mini_owner..."
  goose -dir services/order/migrations -table goose_mini_order postgres "$MIGRATE_DATABASE_URL" up
  goose -dir services/stock/migrations -table goose_mini_stock postgres "$MIGRATE_DATABASE_URL" up
}

build_go() {
  log "Build stock, order, bff..."
  mkdir -p "$BIN_DIR"
  local svc
  for svc in stock order bff; do
    go build -o "$BIN_DIR/$svc" "./services/$svc/cmd/server"
  done
}

start_web() {
  local dev=$1
  (cd web && pnpm install --frozen-lockfile)
  if [[ $dev == 1 ]]; then
    start web bash -c 'cd web && exec pnpm dev'
  else
    log "Build web (pnpm build)..."
    (cd web && pnpm build)
    start web bash -c 'cd web && exec pnpm start'
  fi
  wait_port web "127.0.0.1:$WEB_PORT" "${PIDS[-1]}" 120
}

main() {
  local dev=0
  case "${1:-}" in
    --dev) dev=1 ;;
    "") ;;
    *) die "Tham số không hợp lệ: $1 (dùng: ./run.sh [--dev])" ;;
  esac

  need go; need pnpm; need setsid
  load_env
  need goose

  ensure_free stock "$STOCK_GRPC_ADDR"
  ensure_free order "$ORDER_GRPC_ADDR"
  ensure_free bff "$BFF_HTTP_ADDR"
  ensure_free web "127.0.0.1:$WEB_PORT"

  check_infra
  migrate
  build_go

  mkdir -p "$LOG_DIR"
  trap cleanup EXIT
  trap 'cleanup; exit 130' INT TERM

  # Thứ tự theo luồng gọi: order cần stock, bff cần order.
  start stock "$BIN_DIR/stock"; wait_port stock "$STOCK_GRPC_ADDR" "${PIDS[-1]}"
  start order "$BIN_DIR/order"; wait_port order "$ORDER_GRPC_ADDR" "${PIDS[-1]}"
  start bff "$BIN_DIR/bff";     wait_port bff "$BFF_HTTP_ADDR" "${PIDS[-1]}"
  start_web "$dev"

  log "Mở http://127.0.0.1:$WEB_PORT  (Ctrl+C để dừng). Log: $LOG_DIR/"
  tail -n +1 -F "$LOG_DIR"/{stock,order,bff,web}.log &
  PIDS+=("$!")
  wait -n "${PIDS[@]:0:4}" 2>/dev/null || true
  log "Một service đã thoát, dừng toàn bộ."
}

main "$@"
