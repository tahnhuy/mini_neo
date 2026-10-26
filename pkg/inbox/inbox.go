// Package inbox nhận sự kiện không trùng (docs/specs/events.md, mục 5).
// Cả bốn bước nằm trong một giao dịch:
//  1. đặt app.tenant_id bằng tenantId của phong bì (db.InTx làm việc này);
//  2. INSERT INTO inbox ... ON CONFLICT DO NOTHING; không chèn được = đã xử lý, dừng;
//  3. làm việc nghiệp vụ, ghi outbox nếu cần;
//  4. commit, rồi mới ack cho NATS; lỗi thì không ack để NATS gửi lại.
package inbox

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/tahnhuy/mini_neo/pkg/outbox"
	"github.com/tahnhuy/mini_neo/pkg/tenant"
)

// Store là truy vấn inbox của một service (sqlc).
type Store interface {
	// Claim chèn (tenant hiện tại, consumer, eventID); trả false nếu dòng đã có.
	Claim(ctx context.Context, tx pgx.Tx, consumer string, eventID uuid.UUID) (bool, error)
}

// TxRunner mở giao dịch có tenant; bản thật là *db.DB.
type TxRunner interface {
	InTx(ctx context.Context, fn func(tx pgx.Tx) error) error
}

// Handler làm việc nghiệp vụ của một sự kiện, trong giao dịch đã có dòng inbox.
type Handler func(ctx context.Context, tx pgx.Tx, env outbox.Envelope) error

// Processor xử lý sự kiện cho một consumer, ví dụ "stock.reserve".
type Processor struct {
	Consumer string
	Tx       TxRunner
	Store    Store
	Handle   Handler
}

// Process chạy bước 1–4 (trừ ack). Trả nil nghĩa là đã commit, được phép ack.
// Sự kiện trùng cũng trả nil (đã xử lý ở lần trước).
func (p *Processor) Process(ctx context.Context, payload []byte) error {
	env, err := outbox.Decode(payload)
	if err != nil {
		return err
	}
	ctx = tenant.WithIdentity(ctx, tenant.Identity{TenantID: env.TenantID, UserID: "event:" + p.Consumer})
	err = p.Tx.InTx(ctx, func(tx pgx.Tx) error {
		fresh, err := p.Store.Claim(ctx, tx, p.Consumer, env.ID)
		if err != nil {
			return fmt.Errorf("inbox: claim: %w", err)
		}
		if !fresh {
			return nil
		}
		return p.Handle(ctx, tx, env)
	})
	if err != nil {
		return fmt.Errorf("inbox %s, sự kiện %s: %w", p.Consumer, env.ID, err)
	}
	return nil
}
