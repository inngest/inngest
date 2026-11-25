package icebergcatalog

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// The package registers no database/sql driver, so its tests do (above).

// openSQLite opens a sqlite catalog DB at path as a program would: WAL +
// busy_timeout, since several servers can share it.
func openSQLite(t *testing.T, path string) *sql.DB {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// openPostgres opens a postgres catalog DB at uri.
func openPostgres(t *testing.T, uri string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", uri)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenValidatesDB(t *testing.T) {
	wh := filepath.Join(t.TempDir(), "warehouse")
	_, err := Open(context.Background(), Config{Dialect: SQLite, Warehouse: wh})
	require.ErrorContains(t, err, "DB is required")
	_, err = Open(context.Background(), Config{DB: openSQLite(t, filepath.Join(t.TempDir(), "c.db")), Dialect: "mysql", Warehouse: wh})
	require.ErrorContains(t, err, `unsupported dialect "mysql"`)
}
