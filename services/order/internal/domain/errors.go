// Package domain chứa mô hình nghiệp vụ của service đơn: đơn, dòng đơn, tiền,
// bảng chuyển trạng thái. Không import pgx, grpc, nats.
package domain

// Kind là loại lỗi nghiệp vụ; adapter gRPC đổi Kind sang mã gRPC.
type Kind int

// Các loại lỗi, khớp bảng mã lỗi ở docs/specs/api-contract.md, mục 3.
const (
	KindInvalidArgument Kind = iota + 1
	KindUnauthenticated
	KindPermissionDenied
	KindNotFound
	KindFailedPrecondition
)

// Mã lý do có tên (Q10): chỉ hai mã này.
const (
	ReasonSelfApproval      = "SELF_APPROVAL"
	ReasonInvalidTransition = "INVALID_TRANSITION"
)

// Error là lỗi nghiệp vụ. Message là tiếng Việt, BFF chuyển nguyên cho web (Q32).
type Error struct {
	Kind    Kind
	Reason  string // rỗng nếu lỗi không có mã lý do
	Message string
}

func (e *Error) Error() string {
	if e.Reason != "" {
		return e.Reason + ": " + e.Message
	}
	return e.Message
}

// NewError tạo lỗi không có mã lý do.
func NewError(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}
