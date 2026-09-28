package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

var (
	openOnce sync.Once
	openDB   *sql.DB
	// openErr preserves a failed singleton initialization so later Open
	// calls return the original error instead of proceeding with a nil
	// connection. sync.Once never reruns, so without this a first failure
	// (e.g. lock held past the busy timeout) would panic on conn.Ping.
	openErr error
)

// persistedBusyTimeoutMillis bounds how long a persisted connection waits on
// a locked database before returning SQLITE_BUSY. Connection-local, so it
// must be applied to every pooled connection via the DSN.
const persistedBusyTimeoutMillis = 5000

type Options struct {
	// Persist indicates that the sqlite db should persist data to disk.
	// This can be used for the Dev server, testing, and single-node services.
	Persist bool
	// ForTest indicates that the database handler is created for testing
	// purposes. By default database handlers are singletons, but when this flag
	// is enabled, each call creates a temporary in-memory handler.
	ForTest bool
	// Directory is the path at which the SQLite database should be stored.
	Directory string
}

func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	var (
		conn *sql.DB
		err  error
	)

	l := logger.StdlibLogger(ctx)
	if opts.Persist {
		if opts.ForTest {
			conn, err = openPersisted(opts)
		} else {
			openOnce.Do(func() {
				openDB, openErr = openPersisted(opts)
			})
			conn, err = openDB, openErr
		}
		l = l.With("db", "sqlite", "mode", "persisted")
	} else {
		if opts.ForTest {
			conn, err = openTemporaryMemory()
		} else {
			openOnce.Do(func() {
				openDB, openErr = sql.Open("sqlite", "file:inngest?mode=memory&cache=shared")
			})
			conn, err = openDB, openErr
		}
		l = l.With("db", "sqlite", "mode", "memory")
	}
	l.Info("initialized database")

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	if err := Migrate(ctx, conn); err != nil {
		return nil, err
	}
	l.Info("ran database migrations")

	return conn, nil
}

func Migrate(ctx context.Context, conn *sql.DB) error {
	migrationsFS, err := fs.Sub(MigrationsFS, "migrations")
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, conn, migrationsFS)
	if err != nil {
		return err
	}

	_, err = provider.Up(ctx)
	return err
}

func openPersisted(opts Options) (*sql.DB, error) {
	dir := consts.DefaultInngestConfigDir
	if opts.Directory != "" {
		dir = opts.Directory
	}
	// Resolve to an absolute path unconditionally. A relative path would
	// become the URI authority (file://.inngest/...) which SQLite rejects,
	// and the default config dir is relative.
	if !filepath.IsAbs(dir) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		dir = filepath.Join(wd, dir)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, err
		}
	}

	file := filepath.Join(dir, consts.SQLiteDbFileName)

	conn, err := sql.Open("sqlite", persistedDSN(file))
	if err != nil {
		return nil, err
	}

	// sql.Open is lazy: force a connection so DSN errors surface here with
	// the database path attached, then verify the effective settings. A
	// filesystem without WAL support silently keeps DELETE mode instead of
	// failing, which would lose the concurrency fix without warning.
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open persisted sqlite database %q: %w", file, err)
	}
	if err := verifyPersistedSettings(file, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

// persistedDSN builds the file: URI for a persisted database. Every pooled
// connection configures itself through DSN _pragma parameters: database/sql
// opens connections lazily, so one-time PRAGMAs via Exec would miss
// replacement connections. Journal mode persists in the database header
// (migrating existing DELETE-mode databases on first open); busy timeout and
// synchronous mode are connection-local and reapplied per connection by the
// driver. The driver applies busy_timeout before the other PRAGMAs, so
// concurrent connections racing the initial journal-mode switch wait instead
// of failing. Shared cache is intentionally absent: it is discouraged with
// WAL and increases lock contention. Synchronous stays FULL: the vendored
// engine already defaults WAL synchronization to FULL
// (SQLITE_DEFAULT_WAL_SYNCHRONOUS=2), and the explicit pin keeps that
// durability independent of engine build flags rather than relying on them.
func persistedDSN(file string) string {
	params := url.Values{}
	params.Add("_pragma", "journal_mode(WAL)")
	params.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", persistedBusyTimeoutMillis))
	params.Add("_pragma", "synchronous(FULL)")
	// Root the path so it never becomes the URI authority: a relative path
	// or a Windows drive path (C:/...) would otherwise parse as
	// file://<authority>/..., which SQLite rejects. url.URL also escapes
	// URI-sensitive characters in the path.
	uriPath := "/" + strings.TrimPrefix(filepath.ToSlash(file), "/")
	return (&url.URL{
		Scheme:   "file",
		Path:     uriPath,
		RawQuery: params.Encode(),
	}).String()
}

// verifyPersistedSettings checks the effective journal mode, busy timeout,
// and synchronous mode on a pooled connection. All pooled connections share
// the same DSN, so verifying one verifies the pool's configuration.
func verifyPersistedSettings(file string, conn *sql.DB) error {
	var journalMode string
	if err := conn.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return fmt.Errorf("verify persisted sqlite database %q journal mode: %w", file, err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf(
			"persisted sqlite database %q did not enter WAL journal mode (got %q); "+
				"the filesystem may not support WAL",
			file, journalMode,
		)
	}

	var busyTimeout int
	if err := conn.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		return fmt.Errorf("verify persisted sqlite database %q busy timeout: %w", file, err)
	}
	if busyTimeout != persistedBusyTimeoutMillis {
		return fmt.Errorf(
			"persisted sqlite database %q has unexpected busy timeout %d, want %d",
			file, busyTimeout, persistedBusyTimeoutMillis,
		)
	}

	// Synchronous FULL reads back as 2 (0=OFF, 1=NORMAL, 2=FULL, 3=EXTRA).
	var synchronous int
	if err := conn.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		return fmt.Errorf("verify persisted sqlite database %q synchronous mode: %w", file, err)
	}
	if synchronous != 2 {
		return fmt.Errorf(
			"persisted sqlite database %q has unexpected synchronous mode %d, want FULL (2)",
			file, synchronous,
		)
	}

	return nil
}

func openTemporaryMemory() (*sql.DB, error) {
	dbName := fmt.Sprintf("sqlite_%s", strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
	return sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
}
