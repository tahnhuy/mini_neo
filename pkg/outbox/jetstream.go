package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Stream và subject của bài (Q12).
const (
	StreamName    = "MINI"
	StreamSubject = "mini.>"
)

// PublishTimeout cho mỗi lần gửi lên JetStream.
const PublishTimeout = 5 * time.Second

// EnsureStream tạo (hoặc cập nhật) stream MINI gom subject mini.>.
func EnsureStream(ctx context.Context, js jetstream.JetStream) error {
	ctx, cancel := context.WithTimeout(ctx, PublishTimeout)
	defer cancel()
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     StreamName,
		Subjects: []string{StreamSubject},
	})
	if err != nil {
		return fmt.Errorf("outbox: tạo stream %s: %w", StreamName, err)
	}
	return nil
}

// Connect kết nối NATS, mở JetStream và bảo đảm stream MINI tồn tại.
// Bên gọi đóng nc khi tắt server.
func Connect(ctx context.Context, url string) (*nats.Conn, jetstream.JetStream, error) {
	// MaxReconnects(-1): mặc định nats.go bỏ cuộc sau 60 lần (khoảng 2 phút) rồi đóng
	// hẳn kết nối; khi đó relay và consumer hỏng vĩnh viễn dù NATS đã sống lại.
	nc, err := nats.Connect(url, nats.Timeout(PublishTimeout), nats.MaxReconnects(-1))
	if err != nil {
		return nil, nil, fmt.Errorf("outbox: kết nối NATS %s: %w", url, err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("outbox: mở JetStream: %w", err)
	}
	if err := EnsureStream(ctx, js); err != nil {
		nc.Close()
		return nil, nil, err
	}
	return nc, js, nil
}

// JetStreamPublisher cài đặt Publisher bằng JetStream.
type JetStreamPublisher struct {
	JS jetstream.JetStream
}

// Publish gửi payload lên subject, header Nats-Msg-Id = msgID (id sự kiện).
func (p JetStreamPublisher) Publish(ctx context.Context, subject, msgID string, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, PublishTimeout)
	defer cancel()
	if _, err := p.JS.Publish(ctx, subject, payload, jetstream.WithMsgID(msgID)); err != nil {
		return fmt.Errorf("outbox: publish %s: %w", subject, err)
	}
	return nil
}
