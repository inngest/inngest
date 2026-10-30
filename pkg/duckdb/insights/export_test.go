package insights

import (
	"context"
	"database/sql"
	"testing"
	"time"
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

// SetExecuteTimeout shrinks Execute's deadline for the rest of t.
func SetExecuteTimeout(t *testing.T, d time.Duration) {
	orig := executeTimeout
	executeTimeout = d
	t.Cleanup(func() { executeTimeout = orig })
}

// MaxResultBytes is Execute's result size limit.
func MaxResultBytes() int64 { return maxResultBytes }
