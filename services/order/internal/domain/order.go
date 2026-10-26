package domain

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Money là số tiền tính bằng xu (1 đồng = 100 xu), không dùng số thực.
type Money struct {
	Cents    int64
	Currency string
}

// Status là trạng thái đơn (docs/specs/business-rules.md, mục 2).
type Status string

// Năm trạng thái của đơn, giữ nguyên tên như đề.
const (
	StatusPendingApproval Status = "PENDING_APPROVAL"
	StatusApproved        Status = "APPROVED"
	StatusReserved        Status = "RESERVED"
	StatusOutOfStock      Status = "OUT_OF_STOCK"
	StatusRejected        Status = "REJECTED"
)

// Action là hành động người dùng làm được trên đơn, trả về trong allowed_actions.
const (
	ActionApprove = "APPROVE"
	ActionReject  = "REJECT"
)

// Độ dài lý do từ chối, đếm theo rune, không chuẩn hóa (Q11, Q17).
const (
	MinRejectReason = 10
	MaxRejectReason = 500
)

// Line là một dòng đơn. LineAmount = Quantity × UnitPrice.
type Line struct {
	SKU        string
	Quantity   int32
	UnitPrice  Money
	LineAmount Money
}

// Order là một đơn hàng.
type Order struct {
	ID           uuid.UUID
	Code         string
	Status       Status
	CreatedBy    string
	DecidedBy    string
	RejectReason string
	Total        Money
	Lines        []Line
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// LineInput là một dòng khi tạo đơn: mã hàng, số lượng và đơn giá lấy từ kho.
type LineInput struct {
	SKU       string
	Quantity  int32
	UnitPrice Money
}

// NewOrder tạo đơn ở trạng thái PENDING_APPROVAL và tính tiền trong domain:
// line_amount = quantity × unit_price, total_amount = tổng line_amount.
func NewOrder(id uuid.UUID, createdBy string, inputs []LineInput, now time.Time) (Order, error) {
	if len(inputs) == 0 {
		return Order{}, NewError(KindInvalidArgument, "Đơn cần ít nhất một dòng")
	}
	o := Order{ID: id, Status: StatusPendingApproval, CreatedBy: createdBy, CreatedAt: now, UpdatedAt: now}
	o.Total.Currency = inputs[0].UnitPrice.Currency
	for _, in := range inputs {
		if in.UnitPrice.Currency != o.Total.Currency {
			return Order{}, NewError(KindFailedPrecondition, "Các dòng của đơn phải cùng một loại tiền")
		}
		cents, err := lineCents(in)
		if err != nil {
			return Order{}, err
		}
		if o.Total.Cents > MaxCents-cents {
			return Order{}, errAmountTooLarge
		}
		amount := Money{Cents: cents, Currency: o.Total.Currency}
		o.Lines = append(o.Lines, Line{SKU: in.SKU, Quantity: in.Quantity, UnitPrice: in.UnitPrice, LineAmount: amount})
		o.Total.Cents += amount.Cents
	}
	return o, nil
}

// MaxCents là số xu lớn nhất cột NUMERIC(18,2) chứa được (9.999.999.999.999.999,99).
// Kiểm trước khi nhân để int64 không tràn âm thầm ra tổng sai.
const MaxCents int64 = 999_999_999_999_999_999

var errAmountTooLarge = NewError(KindInvalidArgument, "Số tiền của đơn vượt giới hạn cho phép")

// lineCents tính quantity × unit_price; chặn số lượng hay đơn giá âm và tràn số.
func lineCents(in LineInput) (int64, error) {
	if in.Quantity <= 0 {
		return 0, NewError(KindInvalidArgument, fmt.Sprintf("Số lượng của %s phải lớn hơn 0", in.SKU))
	}
	if in.UnitPrice.Cents < 0 {
		return 0, NewError(KindFailedPrecondition, fmt.Sprintf("Đơn giá của %s không hợp lệ", in.SKU))
	}
	if in.UnitPrice.Cents > MaxCents/int64(in.Quantity) {
		return 0, errAmountTooLarge
	}
	return int64(in.Quantity) * in.UnitPrice.Cents, nil
}

// FormatCode tạo mã đơn dạng DH-000001 từ số thứ tự của tenant (Q7).
func FormatCode(n int64) string {
	return fmt.Sprintf("DH-%06d", n)
}

// transitions là bảng chuyển trạng thái (docs/specs/business-rules.md, mục 3).
var transitions = map[Status][]Status{
	StatusPendingApproval: {StatusApproved, StatusRejected},
	StatusApproved:        {StatusReserved, StatusOutOfStock},
}

// CanTransition báo bảng chuyển có cho đi từ from sang to hay không.
func CanTransition(from, to Status) bool {
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

func (o *Order) transition(to Status, now time.Time) error {
	if !CanTransition(o.Status, to) {
		return &Error{
			Kind:    KindFailedPrecondition,
			Reason:  ReasonInvalidTransition,
			Message: fmt.Sprintf("Không thể chuyển đơn từ %s sang %s", o.Status, to),
		}
	}
	o.Status = to
	o.UpdatedAt = now
	return nil
}

// Approve chuyển đơn sang APPROVED. Kiểm quyền và chặn tự duyệt làm ở usecase.
func (o *Order) Approve(decidedBy string, now time.Time) error {
	if err := o.transition(StatusApproved, now); err != nil {
		return err
	}
	o.DecidedBy = decidedBy
	return nil
}

// Reject chuyển đơn sang REJECTED với lý do dài 10–500 ký tự (đếm rune).
func (o *Order) Reject(decidedBy, reason string, now time.Time) error {
	if n := utf8.RuneCountInString(reason); n < MinRejectReason || n > MaxRejectReason {
		return NewError(KindInvalidArgument,
			fmt.Sprintf("Lý do từ chối phải dài %d–%d ký tự", MinRejectReason, MaxRejectReason))
	}
	if err := o.transition(StatusRejected, now); err != nil {
		return err
	}
	o.DecidedBy = decidedBy
	o.RejectReason = reason
	return nil
}

// ApplyReservation nhận kết quả của kho: held = true thì sang RESERVED,
// ngược lại sang OUT_OF_STOCK. Chỉ hợp lệ khi đơn đang APPROVED.
func (o *Order) ApplyReservation(held bool, now time.Time) error {
	if held {
		return o.transition(StatusReserved, now)
	}
	return o.transition(StatusOutOfStock, now)
}

// AllowedActions tính nút được hiện cho người gọi: chỉ khi đơn chờ duyệt,
// người gọi có quyền duyệt và không phải người lập (docs/specs/api-contract.md, mục 1).
func (o Order) AllowedActions(canApprove bool, caller string) []string {
	if o.Status == StatusPendingApproval && canApprove && caller != o.CreatedBy {
		return []string{ActionApprove, ActionReject}
	}
	return []string{}
}
