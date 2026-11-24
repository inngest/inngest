package driver

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAtomicMultiTableCommit: a quack conn (whose lake here is plain duckdb
// tables) commits several tables at once, a jsonlines conn can't run the
// transaction at all.
func TestAtomicMultiTableCommit(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	quackAddr := EphemeralQuackAddr
	for name, c := range map[string]struct {
		quack *string
		want  bool
	}{
		"default":   {nil, true},
		"quackAddr": {&quackAddr, true},
	} {
		t.Run(name, func(t *testing.T) {
			db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:", QuackAddr: c.quack})
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			conn, err := db.Conn(t.Context())
			require.NoError(t, err)
			defer func(c *sql.Conn) { _ = c.Close() }(conn)
			require.Equal(t, c.want, SupportsTransactions(conn))
			require.Equal(t, c.want, AtomicMultiTableCommit(conn))
		})
	}
}
