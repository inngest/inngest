package base_cqrs

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/inngest/inngest/pkg/consts"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"
)

var (
	o  sync.Once
	db *sql.DB
)

type BaseCQRSOptions struct {
	// InMemory indicates that the sqlite should run in memory only and not persist data to disk.
	// This is expected to be used for the Dev server and testing.
	InMemory bool
	// ForTest indicates that the database handler is created for testing purposes.
	// By default database handlers are all singletons, but when this flag is enabled, they will create temporary handlers.
	//
	// Only supports with in-memory with sqlite for the moment.
	ForTest bool

	// PostgresURI declares the postgres connection to connect to a postgres database
	PostgresURI string

	// The path at which the SQLite database should be stored.
	Directory string
}

func New(opts BaseCQRSOptions) (*sql.DB, error) {
	var err error

	if opts.PostgresURI != "" {
		if !strings.HasPrefix(opts.PostgresURI, "postgres://") && !strings.HasPrefix(opts.PostgresURI, "postgresql://") {
			if u, parseErr := url.Parse(opts.PostgresURI); parseErr == nil {
				return nil, fmt.Errorf("unsupported database URL: %s", u.Redacted())
			}
			return nil, fmt.Errorf("unsupported database URL format")
		}

		o.Do(func() {
			db, err = sql.Open("pgx", opts.PostgresURI)
		})
	} else if opts.InMemory {
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
	} else {
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
	}

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Run migrations.
	if err := up(db, opts); err != nil {
		return nil, err
	}

	return db, err
}

// FS contains the filesystem of the stdlib, containing all migrations in subdirs
// relative to this package.
//
//go:embed **/**/*.sql
var FS embed.FS

func up(db *sql.DB, opts BaseCQRSOptions) error {
	var (
		err    error
		src    source.Driver
		driver database.Driver
		dbName string
	)

	// Grab the migration driver.
	if opts.PostgresURI != "" {
		src, err = iofs.New(FS, path.Join("migrations", "postgres"))
		if err != nil {
			return err
		}

		dbName = "postgres"
		parsedURL, err := url.Parse(opts.PostgresURI)
		if err != nil {
			return fmt.Errorf("error parsing postgres URI to retrieve DB name: invalid format")
		}

		if parsedURL.Path != "" && parsedURL.Path != "/" {
			// Remove the leading slash
			dbName = parsedURL.Path[1:]
		}

		driver, err = postgres.WithInstance(db, &postgres.Config{
			MigrationsTable: "migrations",
			DatabaseName:    dbName,
		})
		if err != nil {
			return err
		}
	} else {
		src, err = iofs.New(FS, path.Join("migrations", "sqlite"))
		if err != nil {
			return err
		}

		driver, err = sqlite.WithInstance(db, &sqlite.Config{
			MigrationsTable: "migrations",
			NoTxWrap:        true,
		})
		if err != nil {
			return err
		}

		dbName = "file:inngest?mode=memory&cache=shared"
		if !opts.InMemory {
			dbName = fmt.Sprintf("file:%s?cache=shared", fmt.Sprintf("%s/%s", consts.DefaultInngestConfigDir, consts.SQLiteDbFileName))
		}
	}

	m, err := migrate.NewWithInstance("iofs", src, dbName, driver)
	if err != nil {
		return err
	}

	v, dirty, err := m.Version()
	if err != migrate.ErrNilVersion && err != nil {
		return err
	}

	if dirty {
		if err = m.Migrate(v); err != nil {
			return fmt.Errorf("error migrating to version %d resetting dirty: %w", v, err)
		}
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		return nil
	}

	return err
}
