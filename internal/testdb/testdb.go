// Package testdb tạo CSDL riêng cho mỗi test tích hợp: tạo mới, chạy migration,
// xóa khi test xong. Không đụng CSDL "mini" đang dùng để demo
// (docs/specs/acceptance.md, mục 3).
//
// CSDL do mini_owner tạo và migrate; test chạy bằng mini_app, nên RLS được kiểm
// thật, không bị superuser bỏ qua.
package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" cho goose
	"github.com/pressly/goose/v3"
)

// Migration là một thư mục migration (tính từ gốc repo) và bảng version goose của nó.
type Migration struct {
	Dir   string
	Table string
}

// Order và Stock là migration của hai service, khớp lệnh goose trong README.
var (
	Order = Migration{Dir: "services/order/migrations", Table: "goose_mini_order"}
	Stock = Migration{Dir: "services/stock/migrations", Table: "goose_mini_stock"}
)

// NewAppPool tạo CSDL test, chạy các migration, rồi trả pool kết nối bằng mini_app.
// CSDL bị xóa khi test kết thúc.
func NewAppPool(t testing.TB, migrations ...Migration) *pgxpool.Pool {
	t.Helper()
	ownerURL, appURL := mustEnv(t, "TEST_OWNER_DATABASE_URL"), mustEnv(t, "TEST_APP_DATABASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	name := randomName(t)
	createDatabase(ctx, t, ownerURL, name)
	t.Cleanup(func() { dropDatabase(t, ownerURL, name) })

	for _, m := range migrations {
		migrate(ctx, t, withDatabase(t, ownerURL, name), m)
	}
	pool, err := pgxpool.New(ctx, withDatabase(t, appURL, name))
	if err != nil {
		t.Fatalf("testdb: mở pool mini_app: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func mustEnv(t testing.TB, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("testdb: thiếu biến môi trường %s (xem .env.example và README mục 4)", key)
	}
	return v
}

func randomName(t testing.TB) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("testdb: sinh tên CSDL: %v", err)
	}
	return "mini_test_" + hex.EncodeToString(b)
}

func createDatabase(ctx context.Context, t testing.TB, ownerURL, name string) {
	t.Helper()
	conn, err := pgx.Connect(ctx, ownerURL)
	if err != nil {
		t.Fatalf("testdb: kết nối mini_owner: %v", err)
	}
	defer conn.Close(ctx)
	// Tên CSDL do randomName sinh, vẫn Sanitize vì không truyền được tham số cho DDL.
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("testdb: tạo CSDL %s: %v", name, err)
	}
}

func dropDatabase(t testing.TB, ownerURL, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, ownerURL)
	if err != nil {
		t.Errorf("testdb: kết nối để xóa %s: %v", name, err)
		return
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
		t.Errorf("testdb: xóa CSDL %s: %v", name, err)
	}
}

func migrate(ctx context.Context, t testing.TB, ownerTestURL string, m Migration) {
	t.Helper()
	sqlDB, err := sql.Open("pgx", ownerTestURL)
	if err != nil {
		t.Fatalf("testdb: mở CSDL cho goose: %v", err)
	}
	defer sqlDB.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB,
		os.DirFS(filepath.Join(repoRoot(), m.Dir)), goose.WithTableName(m.Table))
	if err != nil {
		t.Fatalf("testdb: goose %s: %v", m.Dir, err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("testdb: migrate %s: %v", m.Dir, err)
	}
}

func withDatabase(t testing.TB, rawURL, name string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("testdb: URL CSDL không hợp lệ: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

// repoRoot tính từ vị trí file này: internal/testdb/testdb.go.
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// TenantA và TenantB là UUID cố định của hai tenant mẫu (Q9).
const (
	TenantA = "11111111-1111-1111-1111-111111111111"
	TenantB = "22222222-2222-2222-2222-222222222222"
)
