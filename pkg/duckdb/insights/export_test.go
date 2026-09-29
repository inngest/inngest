package insights

import (
	"context"
	"database/sql"
	"testing"
)

// CheckRenderedSQL exposes checkRenderedSQL to insights_test.
func CheckRenderedSQL(ctx context.Context, db *sql.DB, query string) error {
	return checkRenderedSQL(ctx, db, query)
}

// DisableRenderedSQLGuard turns off Execute's post-render guard for the rest
// of t, so a test can prove the DuckDB-level sandbox holds on its own.
func DisableRenderedSQLGuard(t *testing.T) {
	guardRenderedSQL = false
	t.Cleanup(func() { guardRenderedSQL = true })
}
