package inbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
)

// Thời gian chờ của consumer.
const (
	HandleTimeout = 10 * time.Second // một lần xử lý sự kiện
	AckWait       = 30 * time.Second // chưa ack sau ngần này thì NATS gửi lại
	RetryDelay    = time.Second      // lỗi thì chờ ngần này rồi NATS gửi lại
)

// Subscription gắn một Processor với một durable consumer của stream MINI.
type Subscription struct {
	Durable string // Q12: tên inbox.consumer thay "." bằng "-", ví dụ "stock-reserve"
	Subject string // ví dụ "mini.ord.order.approved"
	Proc    *Processor
	Log     *slog.Logger
}

// Run nhận tin tới khi ctx bị hủy, rồi chờ tin đang xử lý xong mới trả về.
func (s *Subscription) Run(ctx context.Context, js jetstream.JetStream) error {
	cons, err := s.waitConsumer(ctx, js)
	if err != nil {
		return nil // ctx bị hủy trước khi tạo được consumer: server đang tắt
	}
	// PullMaxMessages(1): mỗi tiến trình chỉ giữ một tin chưa xử lý, để nhiều bản
	// chạy song song chia nhau tin thay vì một bản ôm cả lô.
	cc, err := cons.Consume(func(msg jetstream.Msg) { s.handle(ctx, msg) },
		jetstream.PullMaxMessages(1),
		jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
			s.Log.WarnContext(ctx, "consumer gặp lỗi khi nhận tin", "consumer", s.Durable, "err", err)
		}))
	if err != nil {
		return fmt.Errorf("inbox: consume %s: %w", s.Durable, err)
	}
	<-ctx.Done()
	cc.Drain()
	<-cc.Closed()
	return nil
}

// waitConsumer thử tạo durable tới khi được hoặc ctx bị hủy. Không thử lại thì
// NATS chập chờn lúc khởi động làm consumer dừng hẳn trong khi server gRPC vẫn
// chạy, và đơn kẹt ở APPROVED mà không ai biết.
func (s *Subscription) waitConsumer(ctx context.Context, js jetstream.JetStream) (jetstream.Consumer, error) {
	for {
		cons, err := s.ensureConsumer(ctx, js)
		if err == nil {
			return cons, nil
		}
		s.Log.ErrorContext(ctx, "tạo consumer lỗi, sẽ thử lại", "consumer", s.Durable, "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(RetryDelay):
		}
	}
}

func (s *Subscription) ensureConsumer(ctx context.Context, js jetstream.JetStream) (jetstream.Consumer, error) {
	cctx, cancel := context.WithTimeout(ctx, outbox.PublishTimeout)
	defer cancel()
	cons, err := js.CreateOrUpdateConsumer(cctx, outbox.StreamName, jetstream.ConsumerConfig{
		Durable:       s.Durable,
		FilterSubject: s.Subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       AckWait,
	})
	if err != nil {
		return nil, fmt.Errorf("inbox: tạo consumer %s: %w", s.Durable, err)
	}
	return cons, nil
}

// handle: commit xong mới ack. Phong bì hỏng thì Term (gửi lại cũng vô ích);
// lỗi khác thì Nak có trễ để NATS gửi lại.
func (s *Subscription) handle(ctx context.Context, msg jetstream.Msg) {
	hctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), HandleTimeout)
	defer cancel()
	err := s.Proc.Process(hctx, msg.Data())
	switch {
	case err == nil:
		if aerr := msg.Ack(); aerr != nil {
			s.Log.WarnContext(hctx, "ack lỗi, NATS sẽ gửi lại; inbox chặn xử lý trùng", "consumer", s.Durable, "err", aerr)
		}
	case errors.Is(err, outbox.ErrMalformed):
		s.Log.ErrorContext(hctx, "bỏ tin có phong bì hỏng", "consumer", s.Durable, "err", err)
		_ = msg.Term()
	default:
		s.Log.ErrorContext(hctx, "xử lý sự kiện lỗi, chờ gửi lại", "consumer", s.Durable, "err", err)
		_ = msg.NakWithDelay(RetryDelay)
	}
}
