package testdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
)

// NewJetStream kết nối NATS riêng cho test (TEST_NATS_URL, Q36) và tạo lại stream
// MINI sạch cho từng test, để sự kiện và durable của test trước không lẫn vào.
// Test dùng hàm này không được chạy song song (không gọi t.Parallel).
func NewJetStream(t testing.TB) jetstream.JetStream {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nc, js, err := outbox.Connect(ctx, mustEnv(t, "TEST_NATS_URL"))
	if err != nil {
		t.Fatalf("testdb: %v (chạy: docker compose --env-file .env -f deploy/compose.yaml --profile test up -d --wait nats-test)", err)
	}
	t.Cleanup(nc.Close)
	if err := js.DeleteStream(ctx, outbox.StreamName); err != nil && !errors.Is(err, jetstream.ErrStreamNotFound) {
		t.Fatalf("testdb: xóa stream cũ: %v", err)
	}
	if err := outbox.EnsureStream(ctx, js); err != nil {
		t.Fatalf("testdb: %v", err)
	}
	return js
}
