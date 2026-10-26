//go:build integration

// AC-04 mức SQL (Q2): dung (tenant B) dùng đúng id đơn của tenant A, kết nối bằng
// mini_app, cũng không thấy dòng nào. Mức API (GetOrder trả NOT_FOUND) làm ở ngày 2.
package repo_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahnhuy/mini_neo/internal/testdb"
	"github.com/tahnhuy/mini_neo/pkg/db"
	"github.com/tahnhuy/mini_neo/pkg/tenant"
)

func ctxFor(t *testing.T, tenantID, user string) context.Context {
	t.Helper()
	return tenant.WithIdentity(context.Background(), tenant.Identity{
		TenantID: uuid.MustParse(tenantID), UserID: user,
	})
}

func TestTenantIsolation_AC04_OtherTenantNotFound(t *testing.T) {
	pool := testdb.NewAppPool(t, testdb.Order)
	store := db.New(pool)
	assertRoleCannotBypassRLS(t, pool)

	orderID := insertOrder(t, store, ctxFor(t, testdb.TenantA, "an"))

	t.Run("tenant A thấy đơn của mình (đối chứng)", func(t *testing.T) {
		if n := countOrder(t, store, ctxFor(t, testdb.TenantA, "an"), orderID); n != 1 {
			t.Fatalf("tenant A đếm được %d đơn, want 1", n)
		}
	})
	t.Run("tenant B không thấy đơn của tenant A", func(t *testing.T) {
		if n := countOrder(t, store, ctxFor(t, testdb.TenantB, "dung"), orderID); n != 0 {
			t.Fatalf("tenant B đếm được %d đơn, want 0", n)
		}
	})
	t.Run("chưa đặt tenant thì truy vấn lỗi", func(t *testing.T) {
		var n int
		err := pool.QueryRow(context.Background(),
			"SELECT count(*) FROM mini_order.orders WHERE id = $1", orderID).Scan(&n)
		if err == nil {
			t.Fatalf("truy vấn không đặt tenant phải lỗi, nhưng trả %d dòng", n)
		}
	})
}

// Chặn kết quả xanh giả: superuser hoặc vai trò BYPASSRLS sẽ bỏ qua RLS.
func assertRoleCannotBypassRLS(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var user string
	var super, bypass bool
	err := pool.QueryRow(context.Background(),
		"SELECT rolname, rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user").
		Scan(&user, &super, &bypass)
	if err != nil {
		t.Fatal(err)
	}
	if user != "mini_app" || super || bypass {
		t.Fatalf("test phải chạy bằng mini_app không bỏ qua RLS; đang là %s (super=%v, bypassrls=%v)", user, super, bypass)
	}
}

func insertOrder(t *testing.T, store *db.DB, ctx context.Context) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	err := store.InTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO mini_order.orders (id, tenant_id, code, status, created_by, total_amount, currency_code)
			VALUES ($1, current_setting('app.tenant_id')::uuid, 'DH-000001', 'PENDING_APPROVAL', 'an', 0, 'VND')`, id)
		return err
	})
	if err != nil {
		t.Fatalf("chèn đơn của tenant A: %v", err)
	}
	return id
}

func countOrder(t *testing.T, store *db.DB, ctx context.Context, id uuid.UUID) int {
	t.Helper()
	var n int
	err := store.InTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT count(*) FROM mini_order.orders WHERE id = $1", id).Scan(&n)
	})
	if err != nil {
		t.Fatalf("đếm đơn: %v", err)
	}
	return n
}
