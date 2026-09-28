package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// persistedSettings asserts the effective per-connection configuration that
// every pooled connection to a persisted database must carry.
func persistedSettings(t *testing.T, conn *sql.DB) {
	t.Helper()

	var journalMode string
	require.NoError(t, conn.QueryRow("PRAGMA journal_mode").Scan(&journalMode))
	assert.Equal(t, "wal", strings.ToLower(journalMode))

	var busyTimeout int
	require.NoError(t, conn.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout))
	assert.Equal(t, persistedBusyTimeoutMillis, busyTimeout)

	// 0=OFF, 1=NORMAL, 2=FULL, 3=EXTRA: FULL preserves pre-WAL durability.
	var synchronous int
	require.NoError(t, conn.QueryRow("PRAGMA synchronous").Scan(&synchronous))
	assert.Equal(t, 2, synchronous)
}

func TestOpenPersistedFresh(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{Persist: true, ForTest: true, Directory: t.TempDir()})
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	persistedSettings(t, conn)
}

func TestOpenPersistedDirectoryWithSpaces(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "dir with spaces & special=chars")
	conn, err := Open(ctx, Options{Persist: true, ForTest: true, Directory: dir})
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	persistedSettings(t, conn)

	// Round-trip a row to prove the escaped path resolves to a usable file.
	_, err = conn.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS path_probe (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, "INSERT INTO path_probe (id) VALUES (1)")
	require.NoError(t, err)
}

func TestOpenPersistedMigratesExistingDeleteMode(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, consts.SQLiteDbFileName)

	// Simulate a pre-WAL database created by the old opener.
	legacy, err := sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared", file))
	require.NoError(t, err)
	_, err = legacy.Exec("CREATE TABLE legacy_probe (id INTEGER PRIMARY KEY, v TEXT)")
	require.NoError(t, err)
	_, err = legacy.Exec("INSERT INTO legacy_probe (id, v) VALUES (1, 'kept')")
	require.NoError(t, err)
	var mode string
	require.NoError(t, legacy.QueryRow("PRAGMA journal_mode").Scan(&mode))
	require.Equal(t, "delete", strings.ToLower(mode))
	require.NoError(t, legacy.Close())

	ctx := context.Background()
	conn, err := Open(ctx, Options{Persist: true, ForTest: true, Directory: dir})
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	persistedSettings(t, conn)

	var v string
	require.NoError(t, conn.QueryRow("SELECT v FROM legacy_probe WHERE id = 1").Scan(&v))
	assert.Equal(t, "kept", v)
}

func TestOpenPersistedConcurrentConnections(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{Persist: true, ForTest: true, Directory: t.TempDir()})
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Pin several pool connections at once and verify each carries the settings.
	const conns = 4
	pinned := make([]*sql.Conn, conns)
	for i := range pinned {
		pinned[i], err = conn.Conn(ctx)
		require.NoError(t, err)
		defer func(c *sql.Conn) { _ = c.Close() }(pinned[i])
	}
	for _, c := range pinned {
		var journalMode string
		require.NoError(t, c.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode))
		assert.Equal(t, "wal", strings.ToLower(journalMode))

		var busyTimeout int
		require.NoError(t, c.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout))
		assert.Equal(t, persistedBusyTimeoutMillis, busyTimeout)
	}
}

func TestOpenPersistedReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	conn, err := Open(ctx, Options{Persist: true, ForTest: true, Directory: dir})
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, "CREATE TABLE reopen_probe (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	reopened, err := Open(ctx, Options{Persist: true, ForTest: true, Directory: dir})
	require.NoError(t, err)
	defer func() { _ = reopened.Close() }()

	persistedSettings(t, reopened)
	var count int
	require.NoError(t, reopened.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE name = 'reopen_probe'",
	).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestOpenPersistedConcurrentReadsWrites(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{Persist: true, ForTest: true, Directory: t.TempDir()})
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	_, err = conn.ExecContext(ctx, "CREATE TABLE rw_probe (id INTEGER PRIMARY KEY, v TEXT)")
	require.NoError(t, err)

	const writers = 4
	const writesEach = 25
	const readers = 4

	var wg sync.WaitGroup
	errCh := make(chan error, writers+readers)

	for w := range writers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := range writesEach {
				_, err := conn.ExecContext(ctx,
					"INSERT INTO rw_probe (v) VALUES (?)", fmt.Sprintf("w%d-%d", w, i))
				if err != nil {
					errCh <- err
					return
				}
			}
		}(w)
	}
	for range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range writesEach {
				var n int
				if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM rw_probe").Scan(&n); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		// Any locking failure here is a regression: WAL plus the busy
		// timeout must let this bounded workload drain.
		assert.NotContains(t, err.Error(), "database is locked")
		assert.NotContains(t, err.Error(), "database table is locked")
		require.NoError(t, err)
	}

	var total int
	require.NoError(t, conn.QueryRow("SELECT COUNT(*) FROM rw_probe").Scan(&total))
	assert.Equal(t, writers*writesEach, total)
}

func TestOpenPersistedReplacementConnections(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{Persist: true, ForTest: true, Directory: t.TempDir()})
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Force the pool to cycle connections; each replacement configures
	// itself from the DSN, so settings must hold every time.
	conn.SetMaxIdleConns(0)
	for range 5 {
		c, err := conn.Conn(ctx)
		require.NoError(t, err)
		var busyTimeout int
		require.NoError(t, c.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout))
		assert.Equal(t, persistedBusyTimeoutMillis, busyTimeout)
		var journalMode string
		require.NoError(t, c.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode))
		assert.Equal(t, "wal", strings.ToLower(journalMode))
		require.NoError(t, c.Close())
	}
}

func TestOpenPersistedInitFailure(t *testing.T) {
	// A regular file where the database directory should be cannot host
	// main.db: the opener must return an actionable error, not panic.
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))

	_, err := Open(context.Background(), Options{
		Persist:   true,
		ForTest:   true,
		Directory: filepath.Join(blocker, "sub"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persisted sqlite")
}

func TestOpenPersistedForTestFailureReturnsError(t *testing.T) {
	// A failed non-singleton open must surface the wrapped error (with the
	// database path) rather than a nil-pointer panic downstream. This is
	// the ForTest path through the same code the openOnce singleton guards.
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))

	conn, err := Open(context.Background(), Options{
		Persist:   true,
		ForTest:   true,
		Directory: filepath.Join(blocker, "sub"),
	})
	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "persisted sqlite")
}

func TestOpenInMemoryRegression(t *testing.T) {
	ctx := context.Background()

	a, err := Open(ctx, Options{Persist: false, ForTest: true})
	require.NoError(t, err)
	defer func() { _ = a.Close() }()
	b, err := Open(ctx, Options{Persist: false, ForTest: true})
	require.NoError(t, err)
	defer func() { _ = b.Close() }()

	// Existing shared-memory test isolation: each opener gets its own database.
	_, err = a.ExecContext(ctx, "CREATE TABLE isolation_probe (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
	var count int
	require.NoError(t, b.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE name = 'isolation_probe'",
	).Scan(&count))
	assert.Equal(t, 0, count)
}
