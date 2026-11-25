package driver

import (
	"context"
	"fmt"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/duckdbtest"
	"github.com/stretchr/testify/require"
)

// TestQuackAppendersFlushAgainAfterCancelledFlush: a Flush whose ctx ends
// abandons the session it ran on (see quack.Session.Abandoned), which is
// then disconnected. The rows stay buffered, and the next Flush has to run
// on a fresh session (w/ the appender's catalog selected again) rather than
// fail on the dead one for the rest of the appender's life.
func TestQuackAppendersFlushAgainAfterCancelledFlush(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	duckdbtest.RequireQuackExtension(t, binPath)

	for _, conns := range []int{0, 4} {
		t.Run(fmt.Sprintf("QuackConns=%d", conns), func(t *testing.T) {
			addr := EphemeralQuackAddr
			db, err := Open(t.Context(), Options{
				BinaryPath: binPath,
				DBFile:     ":memory:",
				Iceberg:    testIceberg(t),
				QuackAddr:  &addr,
				QuackConns: conns,
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			cancelled, cancel := context.WithCancel(t.Context())
			cancel()

			t.Run("append", func(t *testing.T) {
				_, err := db.ExecContext(t.Context(), "CREATE TABLE "+LakeAlias+".reflush_append_t (k VARCHAR);")
				require.NoError(t, err)

				// LakeAlias as the catalog: the retried Flush only binds
				// main.reflush_append_t if its new session ran the USE too.
				appender, err := NewQuackAppender(t.Context(), db, LakeAlias, "main", "reflush_append_t", []QuackColumnKind{QuackColumnVarchar})
				require.NoError(t, err)
				require.NoError(t, appender.AppendRow("a"))
				require.NoError(t, appender.AppendRow("b"))

				require.ErrorIs(t, appender.Flush(cancelled), context.Canceled)
				require.NoError(t, appender.Flush(t.Context()), "the rows are still buffered, so Flush is retryable")
				require.NoError(t, appender.AppendRow("c"))
				require.NoError(t, appender.Close(t.Context()))

				var n int
				require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM "+LakeAlias+".reflush_append_t;").Scan(&n))
				require.Equal(t, 3, n)
			})

			t.Run("merge", func(t *testing.T) {
				_, err := db.ExecContext(t.Context(), "CREATE TABLE "+LakeAlias+".reflush_merge_t (k VARCHAR, v VARCHAR);")
				require.NoError(t, err)

				appender, err := NewQuackMergeAppender(t.Context(), db, LakeAlias, "main", "reflush_merge_t",
					QuackMergeConfig{On: "t.k = s.k", UpdateSet: "v = s.v"},
					[]QuackMergeColumn{{Name: "k", Kind: QuackColumnVarchar}, {Name: "v", Kind: QuackColumnVarchar}})
				require.NoError(t, err)
				require.NoError(t, appender.AppendRow("a", "1"))

				require.ErrorIs(t, appender.Flush(cancelled), context.Canceled)
				require.NoError(t, appender.Flush(t.Context()), "the rows are still buffered, so Flush is retryable")
				require.NoError(t, appender.AppendRow("a", "2"))
				require.NoError(t, appender.Close(t.Context()))

				var v string
				require.NoError(t, db.QueryRowContext(t.Context(), "SELECT v FROM "+LakeAlias+".reflush_merge_t WHERE k = 'a';").Scan(&v))
				require.Equal(t, "2", v)
			})
		})
	}
}
