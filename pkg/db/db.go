// Package db cho mã nghiệp vụ đúng một cách mở giao dịch: InTx.
//
// InTx luôn đặt app.tenant_id bằng set_config(..., true) ngay sau BEGIN,
// nên RLS lọc theo tenant của người gọi (docs/specs/data-model.md, mục 5).
// Không đặt cho cả kết nối, vì kết nối trong pool được dùng lại cho tenant khác.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahnhuy/mini_neo/pkg/tenant"
)

// ErrNoTenant: context không có tenant nên không mở giao dịch.
var ErrNoTenant = errors.New("db: context không có tenant")

// DB bọc pool kết nối của vai trò mini_app.
type DB struct {
	pool *pgxpool.Pool
}

// New tạo DB từ pool có sẵn.
func New(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

// InTx lấy tenant từ ctx, mở giao dịch, đặt app.tenant_id, chạy fn,
// rồi commit nếu fn trả nil, ngược lại rollback.
func (d *DB) InTx(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	id, ok := tenant.FromContext(ctx)
	if !ok {
		return ErrNoTenant
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db: begin: %w", err)
	}
	defer func() {
		if err != nil {
			// Lỗi rollback không che lỗi gốc; Rollback sau Commit là vô hại.
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", id.TenantID.String()); err != nil {
		return fmt.Errorf("db: set tenant: %w", err)
	}
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: commit: %w", err)
	}
	return nil
}
