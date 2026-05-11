package base_cqrs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/consts"
	dbpostgres "github.com/inngest/inngest/pkg/db/postgres"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/oklog/ulid/v2"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

var (
	o  sync.Once
	db *sql.DB
)

type BaseCQRSOptions struct {
	// Persist indicates that the sqlite db should persist data to disk.
	// This can be used for the Dev server, testing, and single-node services.
	Persist bool
	// ForTest indicates that the database handler is created for testing purposes.
	// By default database handlers are all singletons, but when this flag is enabled, they will create temporary handlers.
	//
	// Only supports with in-memory with sqlite for the moment.
	ForTest bool

	// PostgresURI declares the postgres connection to connect to a postgres database
	PostgresURI string

	// PostgresPool configures the Postgres database/sql connection pool.
	PostgresPool *PostgresPoolOptions

	// The path at which the SQLite database should be stored.
	Directory string
}

func New(ctx context.Context, opts BaseCQRSOptions) (*sql.DB, error) {
	var err error

	l := logger.StdlibLogger(ctx)

	if opts.PostgresURI != "" {
		if !strings.HasPrefix(opts.PostgresURI, "postgres://") && !strings.HasPrefix(opts.PostgresURI, "postgresql://") {
			if u, parseErr := url.Parse(opts.PostgresURI); parseErr == nil {
				return nil, fmt.Errorf("unsupported database URL: %s", u.Redacted())
			}
			return nil, fmt.Errorf("unsupported database URL format")
		}

		if opts.ForTest {
			// For tests, create a new connection each time
			db, err = sql.Open("pgx", opts.PostgresURI)
		} else {
			o.Do(func() {
				db, err = sql.Open("pgx", opts.PostgresURI)
			})
		}
		l = l.With("db", "postgres")
	} else if opts.Persist {
		o.Do(func() {
			// make the dir if it doesn't exist
			dir := consts.DefaultInngestConfigDir
			if opts.Directory != "" {
				dir = opts.Directory
				if !filepath.IsAbs(opts.Directory) {
					wd, err := os.Getwd()
					if err != nil {
						return
					}

					dir = filepath.Join(wd, opts.Directory)
				}
			}

			if _, err := os.Stat(dir); os.IsNotExist(err) {
				err = os.MkdirAll(dir, 0750)
				if err != nil {
					return
				}
			}

			file := filepath.Join(dir, consts.SQLiteDbFileName)

			db, err = sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared", file))
		})
		l = l.With("db", "sqlite", "mode", "persisted")
	} else {
		// In-memory
		if opts.ForTest {
			// initializes a temporary database every time for test purposes
			dbName := fmt.Sprintf("sqlite_%s", strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
			db, err = sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
		} else {
			// initialize the db once
			o.Do(func() {
				db, err = sql.Open("sqlite", "file:inngest?mode=memory&cache=shared")
			})
		}
		l = l.With("db", "sqlite", "mode", "memory")
	}
	if err != nil {
		return nil, err
	}

	l.Info("initialized database")

	if opts.PostgresURI != "" && opts.PostgresPool != nil {
		applyPostgresPoolOptions(db, *opts.PostgresPool)
		l.Info(
			"configured postgres database pool",
			"max_idle_conns", opts.PostgresPool.MaxIdleConns,
			"max_open_conns", opts.PostgresPool.MaxOpenConns,
			"conn_max_idle_time_minutes", opts.PostgresPool.ConnMaxIdleTime,
			"conn_max_lifetime_minutes", opts.PostgresPool.ConnMaxLifetime,
		)
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Run migrations.
	if err := up(db, opts); err != nil {
		return nil, err
	}
	l.Info("ran database migrations")

	return db, err
}

func up(db *sql.DB, opts BaseCQRSOptions) error {
	dialect, migrationsFS, err := gooseConfig(opts)
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(dialect, db, migrationsFS)
	if err != nil {
		return err
	}

	_, err = provider.Up(context.Background())
	return err
}

func gooseConfig(opts BaseCQRSOptions) (goose.Dialect, fs.FS, error) {
	if opts.PostgresURI != "" {
		migrationsFS, err := fs.Sub(dbpostgres.MigrationsFS, "migrations")
		return goose.DialectPostgres, migrationsFS, err
	}

	migrationsFS, err := fs.Sub(dbsqlite.MigrationsFS, "migrations")
	return goose.DialectSQLite3, migrationsFS, err
}
