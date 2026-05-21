package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
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
)

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
				openDB, err = openPersisted(opts)
			})
			conn = openDB
		}
		l = l.With("db", "sqlite", "mode", "persisted")
	} else {
		if opts.ForTest {
			conn, err = openTemporaryMemory()
		} else {
			openOnce.Do(func() {
				openDB, err = sql.Open("sqlite", "file:inngest?mode=memory&cache=shared")
			})
			conn = openDB
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
		if !filepath.IsAbs(opts.Directory) {
			wd, err := os.Getwd()
			if err != nil {
				return nil, err
			}

			dir = filepath.Join(wd, opts.Directory)
		}
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, err
		}
	}

	file := filepath.Join(dir, consts.SQLiteDbFileName)
	return sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared", file))
}

func openTemporaryMemory() (*sql.DB, error) {
	dbName := fmt.Sprintf("sqlite_%s", strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
	return sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
}
