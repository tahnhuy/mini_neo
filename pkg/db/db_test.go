package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Thiếu tenant thì InTx dừng trước khi chạm pool (pool nil ở đây),
// và không chạy hàm nghiệp vụ.
func TestInTxWithoutTenant(t *testing.T) {
	called := false
	err := New(nil).InTx(context.Background(), func(pgx.Tx) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrNoTenant) {
		t.Fatalf("err = %v, want ErrNoTenant", err)
	}
	if called {
		t.Error("hàm nghiệp vụ không được chạy khi thiếu tenant")
	}
}
