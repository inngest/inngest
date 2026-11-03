package driver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/duckdbtest"
	"github.com/stretchr/testify/require"
)

func openQuackForStreams(t *testing.T, conns int) *sql.DB {
	t.Helper()
	binPath := RequireDuckDBBinary(t)
	duckdbtest.RequireQuackExtension(t, binPath)
	dir := t.TempDir()
	addr := EphemeralQuackAddr
	db, err := Open(t.Context(), Options{
		BinaryPath: binPath, DBFile: ":memory:",
		DuckLake:  &DuckLakeOptions{CatalogPath: filepath.Join(dir, "catalog.ducklake"), DataPath: filepath.Join(dir, "data")},
		QuackAddr: &addr, QuackConns: conns,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// testStream serves batches in order, optionally sleeping before each.
func testStream(t *testing.T, cols []QuackColumnKind, delay time.Duration, batches ...[][]any) (QuackStream, string) {
	t.Helper()
	id, err := NewQuackStreamID()
	require.NoError(t, err)
	scan, err := QuackStreamScan(id, cols)
	require.NoError(t, err)
	i := 0
	return QuackStream{ID: id, Columns: cols, Next: func(ctx context.Context) ([][]any, error) {
		if i == len(batches) {
			return nil, io.EOF
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
		i++
		return batches[i-1], nil
	}}, scan
}

func rowsOf(start, n int) [][]any {
	out := make([][]any, n)
	for i := range out {
		out[i] = []any{fmt.Sprintf("r%d", start+i), int64(start + i)}
	}
	return out
}

func TestQuackStreamsFeedAQuery(t *testing.T) {
	for _, conns := range []int{1, 2} {
		t.Run(fmt.Sprintf("%d quack conns", conns), func(t *testing.T) {
			testQuackStreamsFeedAQuery(t, openQuackForStreams(t, conns))
		})
	}
}

func testQuackStreamsFeedAQuery(t *testing.T, db *sql.DB) {
	// Pin one conn, so the pooled (second) conn is the one used when there
	// is one.
	if db.Stats().MaxOpenConnections != 1 {
		first, err := db.Conn(t.Context())
		require.NoError(t, err)
		t.Cleanup(func() { _ = first.Close() })
	}
	cols := []QuackColumnKind{QuackColumnVarchar, QuackColumnBigint}

	t.Run("one stream, many batches over the vector size", func(t *testing.T) {
		st, scan := testStream(t, cols, 0, rowsOf(0, 3000), rowsOf(3000, 10), rowsOf(3010, 2050))
		var n, sum int64
		require.NoError(t, db.QueryRowContext(WithQuackStreams(t.Context(), st),
			"SELECT count(*), sum(c1) FROM "+scan).Scan(&n, &sum))
		require.EqualValues(t, 5060, n)
		require.EqualValues(t, 5060*5059/2, sum)
	})

	t.Run("two streams, each read several times through MATERIALIZED CTEs, sources slow", func(t *testing.T) {
		a, scanA := testStream(t, cols, 20*time.Millisecond, rowsOf(0, 100), rowsOf(100, 100))
		b, scanB := testStream(t, cols, 30*time.Millisecond, rowsOf(150, 100))
		q := fmt.Sprintf(`WITH a AS MATERIALIZED (SELECT c0 AS id, c1 AS v FROM %s),
			b AS MATERIALIZED (SELECT c0 AS id, c1 AS v FROM %s)
			SELECT (SELECT count(*) FROM a), (SELECT count(*) FROM b),
			       (SELECT count(*) FROM a JOIN b USING (id)), (SELECT count(*) FROM a WHERE id NOT IN (SELECT id FROM b))`, scanA, scanB)
		var na, nb, both, onlyA int64
		require.NoError(t, db.QueryRowContext(WithQuackStreams(t.Context(), a, b), q).Scan(&na, &nb, &both, &onlyA))
		require.EqualValues(t, []int64{200, 100, 50, 150}, []int64{na, nb, both, onlyA})
	})

	t.Run("an empty stream", func(t *testing.T) {
		st, scan := testStream(t, cols, 0)
		var n int64
		require.NoError(t, db.QueryRowContext(WithQuackStreams(t.Context(), st), "SELECT count(*) FROM "+scan).Scan(&n))
		require.Zero(t, n)
	})

	t.Run("a failing source fails the query with its error", func(t *testing.T) {
		boom := errors.New("source failed")
		st, scan := testStream(t, cols, 0, rowsOf(0, 10))
		next := st.Next
		calls := 0
		st.Next = func(ctx context.Context) ([][]any, error) {
			if calls++; calls == 2 {
				return nil, boom
			}
			return next(ctx)
		}
		var n int64
		err := db.QueryRowContext(WithQuackStreams(t.Context(), st), "SELECT count(*) FROM "+scan).Scan(&n)
		require.ErrorIs(t, err, boom)
	})

	t.Run("a stream the statement never opens doesn't hold the query", func(t *testing.T) {
		st, _ := testStream(t, cols, time.Hour, rowsOf(0, 1))
		start := time.Now()
		var n int64
		require.NoError(t, db.QueryRowContext(WithQuackStreams(t.Context(), st), "SELECT 1").Scan(&n))
		require.EqualValues(t, 1, n)
		require.Less(t, time.Since(start), 10*time.Second)
	})

	t.Run("a failing query stops a blocked source", func(t *testing.T) {
		st, _ := testStream(t, cols, time.Hour, rowsOf(0, 1))
		done := make(chan error, 1)
		go func() {
			_, err := db.QueryContext(WithQuackStreams(t.Context(), st), "SELECT * FROM no_such_table")
			done <- err
		}()
		select {
		case err := <-done:
			require.Error(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("query with a blocked source didn't return")
		}
	})
}
