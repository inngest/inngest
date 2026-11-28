package driver

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWithMaxResultBytes covers both transports: one oversized cell (kept
// under jsonlines' own 4MiB line cap), and a result that only exceeds the
// limit across many rows (quack pages that in over several fetches). Either
// way the statement fails with ErrResultTooLarge, and the same connection
// keeps working afterwards.
func TestWithMaxResultBytes(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	open := map[string]func(t *testing.T) *sql.DB{
		"jsonlines": func(t *testing.T) *sql.DB {
			db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
			require.NoError(t, err)
			return db
		},
		"quack": func(t *testing.T) *sql.DB {
			addr := EphemeralQuackAddr
			db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:", QuackAddr: &addr, QuackConns: 2})
			require.NoError(t, err)
			return db
		},
	}

	for name, openDB := range open {
		t.Run(name, func(t *testing.T) {
			db := openDB(t)
			t.Cleanup(func() { _ = db.Close() })

			conn, err := db.Conn(t.Context())
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })

			limited := WithMaxResultBytes(t.Context(), 1<<20)
			for _, q := range []string{
				"SELECT repeat('a', 3000000) AS big;",
				"SELECT i, repeat(md5(i::VARCHAR), 4) AS s FROM range(100000) r(i);",
			} {
				_, err := conn.QueryContext(limited, q)
				require.ErrorIs(t, err, ErrResultTooLarge, q)
			}

			// Under the limit, and without one at all, queries still work on
			// the same connection.
			var n int
			require.NoError(t, conn.QueryRowContext(limited, "SELECT count(*) FROM range(1000);").Scan(&n))
			require.Equal(t, 1000, n)
			var s string
			require.NoError(t, conn.QueryRowContext(t.Context(), "SELECT repeat('a', 3000000) AS big;").Scan(&s))
			require.Len(t, s, 3000000)
		})
	}
}
