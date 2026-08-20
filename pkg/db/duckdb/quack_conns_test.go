package duckdb

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestOpenWithQuackConnsRequiresQuackAddr pins the validation: QuackConns
// only means anything once a quack listener actually exists to open
// additional sessions against.
func TestOpenWithQuackConnsRequiresQuackAddr(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		QuackConns: 3,
	})
	require.Error(t, err)
	require.Nil(t, db)
}

// TestOpenWithQuackConnsRunsQueriesConcurrently is the real proof: with
// QuackConns > 1, N goroutines each running a deliberately expensive query
// must genuinely overlap in the subprocess rather than queue behind
// process.exec's mutex the way the default single-connection Open does.
// Each query tracks how many others are in flight at the same time via an
// atomic counter; if the observed peak never exceeds 1, connections are
// still fully serialized and this test fails.
func TestOpenWithQuackConnsRunsQueriesConcurrently(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)
	dir := t.TempDir()

	addr := freeLocalAddr(t)

	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.ducklake"),
			DataPath:    filepath.Join(dir, "data"),
		},
		QuackAddr:  &addr,
		QuackConns: 3,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var inFlight, peak int64
	const workers = 3
	errCh := make(chan error, workers)

	for range workers {
		go func() {
			cur := atomic.AddInt64(&inFlight, 1)
			for {
				p := atomic.LoadInt64(&peak)
				if cur <= p || atomic.CompareAndSwapInt64(&peak, p, cur) {
					break
				}
			}
			defer atomic.AddInt64(&inFlight, -1)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			// A deliberately expensive scan (~0.5-1s locally) so overlapping
			// goroutines have a real window to be observed running at once.
			_, execErr := db.ExecContext(ctx, "SELECT count(*) FROM range(300000000) t(x) WHERE x % 7 = 0;")
			errCh <- execErr
		}()
	}

	for range workers {
		require.NoError(t, <-errCh)
	}

	require.Greater(t, atomic.LoadInt64(&peak), int64(1),
		"at least two queries must have been observed running at the same time; QuackConns did not produce real concurrency")
}
