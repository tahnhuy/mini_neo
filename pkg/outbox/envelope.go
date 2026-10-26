// Package outbox phát sự kiện an toàn: mã nghiệp vụ ghi phong bì vào bảng outbox
// trong cùng giao dịch với thay đổi nghiệp vụ; relay đọc bảng rồi gửi JetStream
// (docs/specs/events.md, mục 3, 4). Mã nghiệp vụ không bao giờ gọi NATS trực tiếp.
package outbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SchemaVersion của mọi sự kiện trong bài.
const SchemaVersion = 1

// Envelope là phong bì sự kiện, cũng là toàn bộ cột payload của outbox.
// Tên trường JSON đúng như mẫu ở docs/specs/events.md, mục 3.
type Envelope struct {
	ID            uuid.UUID       `json:"id"`
	Type          string          `json:"type"`
	Source        string          `json:"source"`
	TenantID      uuid.UUID       `json:"tenantId"`
	CorrelationID string          `json:"correlationId"`
	CausationID   *string         `json:"causationId"`
	OccurredAt    time.Time       `json:"occurredAt"`
	SchemaVersion int             `json:"schemaVersion"`
	Data          json.RawMessage `json:"data"`
}

// Meta là phần phong bì do bên phát chọn; id, schemaVersion và data do NewEnvelope điền.
type Meta struct {
	Type          string
	Source        string
	TenantID      uuid.UUID
	CorrelationID string     // = orderId cho cả hành trình
	CausationID   *uuid.UUID // id sự kiện gây ra sự kiện này; nil nếu không có
	OccurredAt    time.Time
}

// NewEnvelope tạo phong bì mới với id UUIDv7 và data được mã hóa JSON.
func NewEnvelope(m Meta, data any) (Envelope, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Envelope{}, fmt.Errorf("outbox: sinh id sự kiện: %w", err)
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, fmt.Errorf("outbox: mã hóa data của %s: %w", m.Type, err)
	}
	env := Envelope{
		ID: id, Type: m.Type, Source: m.Source, TenantID: m.TenantID,
		CorrelationID: m.CorrelationID, OccurredAt: m.OccurredAt.UTC(),
		SchemaVersion: SchemaVersion, Data: raw,
	}
	if m.CausationID != nil {
		c := m.CausationID.String()
		env.CausationID = &c
	}
	return env, nil
}

// ErrMalformed: payload không phải phong bì hợp lệ; gửi lại cũng không sửa được.
var ErrMalformed = errors.New("outbox: phong bì không hợp lệ")

// Decode đọc phong bì từ payload của tin NATS.
func Decode(payload []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if env.ID == uuid.Nil || env.TenantID == uuid.Nil || env.Type == "" {
		return Envelope{}, fmt.Errorf("%w: thiếu id, tenantId hoặc type", ErrMalformed)
	}
	return env, nil
}
