package domain

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var now = time.Date(2026, 10, 26, 3, 15, 0, 0, time.UTC)

func vnd(dong int64) Money { return Money{Cents: dong * 100, Currency: "VND"} }

var totalCases = []struct {
	name      string
	inputs    []LineInput
	wantTotal int64
	wantLines []int64
}{
	{
		"2 dòng như AC-01",
		[]LineInput{{"BAN-01", 2, vnd(1250000)}, {"GHE-01", 3, vnd(850000)}},
		vnd(5050000).Cents, []int64{vnd(2500000).Cents, vnd(2550000).Cents},
	},
	{
		"giá có phần xu: 0,10 + 0,20 = 0,30",
		[]LineInput{{"A", 1, Money{10, "VND"}}, {"B", 1, Money{20, "VND"}}},
		30, []int64{10, 20},
	},
}

// AC-01 (phần tính tổng): tổng đúng đến đồng, tính bằng xu, không qua số thực.
func TestNewOrder_AC01_TotalExact(t *testing.T) {
	for _, tt := range totalCases {
		t.Run(tt.name, func(t *testing.T) {
			o, err := NewOrder(uuid.New(), "an", tt.inputs, now)
			if err != nil {
				t.Fatal(err)
			}
			if o.Total.Cents != tt.wantTotal || o.Total.Currency != "VND" {
				t.Errorf("total = %+v, want %d VND", o.Total, tt.wantTotal)
			}
			for i, l := range o.Lines {
				if l.LineAmount.Cents != tt.wantLines[i] {
					t.Errorf("dòng %d line_amount = %d, want %d", i, l.LineAmount.Cents, tt.wantLines[i])
				}
			}
			if o.Status != StatusPendingApproval {
				t.Errorf("status = %s, want PENDING_APPROVAL", o.Status)
			}
		})
	}
}

func TestNewOrderRejectsBadInput(t *testing.T) {
	tests := []struct {
		name   string
		inputs []LineInput
		kind   Kind
	}{
		{"không có dòng", nil, KindInvalidArgument},
		{"khác loại tiền", []LineInput{{"A", 1, vnd(1)}, {"B", 1, Money{100, "USD"}}}, KindFailedPrecondition},
		{"số lượng 0", []LineInput{{"A", 0, vnd(1)}}, KindInvalidArgument},
		{"đơn giá âm", []LineInput{{"A", 1, Money{-100, "VND"}}}, KindFailedPrecondition},
		{"một dòng tràn int64", []LineInput{{"A", math.MaxInt32, Money{MaxCents, "VND"}}}, KindInvalidArgument},
		{"tổng vượt NUMERIC(18,2)", []LineInput{{"A", 1, Money{MaxCents, "VND"}}, {"B", 1, Money{1, "VND"}}}, KindInvalidArgument},
	}
	for _, tt := range tests {
		_, err := NewOrder(uuid.New(), "an", tt.inputs, now)
		var de *Error
		if !errors.As(err, &de) || de.Kind != tt.kind {
			t.Errorf("%s: err = %v, want kind %v", tt.name, err, tt.kind)
		}
	}
}

func TestFormatCode(t *testing.T) {
	if got := FormatCode(1); got != "DH-000001" {
		t.Errorf("FormatCode(1) = %s", got)
	}
	if got := FormatCode(123456); got != "DH-123456" {
		t.Errorf("FormatCode(123456) = %s", got)
	}
}

// Bảng chuyển trạng thái: mọi cặp (from, to) ngoài bảng đều bị chặn.
func TestCanTransition(t *testing.T) {
	all := []Status{StatusPendingApproval, StatusApproved, StatusReserved, StatusOutOfStock, StatusRejected}
	allowed := map[[2]Status]bool{
		{StatusPendingApproval, StatusApproved}: true,
		{StatusPendingApproval, StatusRejected}: true,
		{StatusApproved, StatusReserved}:        true,
		{StatusApproved, StatusOutOfStock}:      true,
	}
	for _, from := range all {
		for _, to := range all {
			if got, want := CanTransition(from, to), allowed[[2]Status{from, to}]; got != want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}

// AC-09 ở tầng domain: duyệt đơn đang REJECTED trả FAILED_PRECONDITION, INVALID_TRANSITION.
func TestApprove_AC09_RejectedInvalidTransition(t *testing.T) {
	tests := []Status{StatusRejected, StatusApproved, StatusReserved, StatusOutOfStock}
	for _, from := range tests {
		o := Order{Status: from, CreatedBy: "an"}
		err := o.Approve("chi", now)
		var de *Error
		if !errors.As(err, &de) || de.Kind != KindFailedPrecondition || de.Reason != ReasonInvalidTransition {
			t.Errorf("Approve từ %s: err = %v, want FAILED_PRECONDITION/INVALID_TRANSITION", from, err)
		}
		if o.Status != from || o.DecidedBy != "" {
			t.Errorf("Approve từ %s làm đổi đơn: %+v", from, o)
		}
	}
}

func TestApplyReservation(t *testing.T) {
	tests := []struct {
		from Status
		held bool
		want Status
		ok   bool
	}{
		{StatusApproved, true, StatusReserved, true},
		{StatusApproved, false, StatusOutOfStock, true},
		{StatusPendingApproval, true, StatusPendingApproval, false},
		{StatusReserved, false, StatusReserved, false},
	}
	for _, tt := range tests {
		o := Order{Status: tt.from}
		err := o.ApplyReservation(tt.held, now)
		if (err == nil) != tt.ok || o.Status != tt.want {
			t.Errorf("ApplyReservation(%v) từ %s: status = %s, err = %v", tt.held, tt.from, o.Status, err)
		}
	}
}

func TestReject(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		ok     bool
	}{
		{"9 ký tự", strings.Repeat("a", 9), false},
		{"10 ký tự có dấu (đếm rune, không đếm byte)", "Hết hàng ở", true},
		{"500 ký tự", strings.Repeat("ữ", 500), true},
		{"501 ký tự", strings.Repeat("ữ", 501), false},
	}
	for _, tt := range tests {
		o := Order{Status: StatusPendingApproval, CreatedBy: "an"}
		err := o.Reject("chi", tt.reason, now)
		if (err == nil) != tt.ok {
			t.Errorf("%s: err = %v, ok = %v", tt.name, err, tt.ok)
		}
		if tt.ok && (o.Status != StatusRejected || o.RejectReason != tt.reason || o.DecidedBy != "chi") {
			t.Errorf("%s: đơn sau khi từ chối = %+v", tt.name, o)
		}
	}
}

func TestAllowedActions(t *testing.T) {
	pending := Order{Status: StatusPendingApproval, CreatedBy: "binh"}
	tests := []struct {
		name       string
		order      Order
		canApprove bool
		caller     string
		want       int
	}{
		{"người duyệt khác người lập", pending, true, "chi", 2},
		{"người lập có quyền duyệt", pending, true, "binh", 0},
		{"không có quyền duyệt", pending, false, "an", 0},
		{"đơn đã duyệt", Order{Status: StatusApproved, CreatedBy: "binh"}, true, "chi", 0},
	}
	for _, tt := range tests {
		got := tt.order.AllowedActions(tt.canApprove, tt.caller)
		if len(got) != tt.want || got == nil {
			t.Errorf("%s: AllowedActions = %v, want %d phần tử (không nil)", tt.name, got, tt.want)
		}
	}
}
