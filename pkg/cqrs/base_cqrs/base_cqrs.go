package base_cqrs

import (
	"crypto/rand"
	"crypto/tls"
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
	"github.com/inngest/inngest/pkg/azure"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/oklog/ulid/v2"
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

	// The path at which the SQLite database should be stored.
	Directory string
}

func New(opts BaseCQRSOptions) (*sql.DB, error) {
	var err error

	azureAuth := azure.IsAzureAuthEnabled()

	if azureAuth && opts.PostgresURI != "" {
		return nil, fmt.Errorf("cannot use both Azure Workload Identity (AZURE_POSTGRESQL_HOST) and PostgresURI; choose one authentication method")
	}

	if azureAuth {
		// Azure Workload Identity authentication: build connection from
		// individual env vars and use a BeforeConnect hook to inject tokens.
		if opts.ForTest {
			db, err = openAzurePostgres()
		} else {
			o.Do(func() {
				db, err = openAzurePostgres()
			})
		}
	} else if opts.PostgresURI != "" {
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

// openAzurePostgres creates a *sql.DB using Azure Workload Identity authentication.
// It reads connection parameters from AZURE_POSTGRESQL_* env vars and uses the
// pgx stdlib OpenDB with a BeforeConnect hook that injects Azure AD tokens.
func openAzurePostgres() (*sql.DB, error) {
	cfg, err := azure.LoadAzurePostgresConfig()
	if err != nil {
		return nil, err
	}

	// Build pgx ConnConfig by assigning struct fields directly instead of
	// interpolating into a DSN string, avoiding escaping issues with special
	// characters in host/database/user values.
	connConfig, err := pgx.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("failed to create base pgx config: %w", err)
	}
	connConfig.Host = cfg.Host
	connConfig.Port = cfg.Port
	connConfig.Database = cfg.Database
	connConfig.User = cfg.User
	connConfig.TLSConfig = &tls.Config{
		ServerName: cfg.Host,
		MinVersion: tls.VersionTLS12,
	}

	// Set search_path if schema is specified
	if cfg.Schema != "" {
		if connConfig.RuntimeParams == nil {
			connConfig.RuntimeParams = make(map[string]string)
		}
		connConfig.RuntimeParams["search_path"] = cfg.Schema
	}

	beforeConnect, err := azure.NewBeforeConnectHook()
	if err != nil {
		return nil, err
	}

	return stdlib.OpenDB(*connConfig, stdlib.OptionBeforeConnect(beforeConnect)), nil
}

func up(db *sql.DB, opts BaseCQRSOptions) error {
	var (
		err    error
		src    source.Driver
		driver database.Driver
		dbName string
	)

	// Grab the migration driver.
	if opts.PostgresURI != "" || azure.IsAzureAuthEnabled() {
		src, err = iofs.New(FS, path.Join("migrations", "postgres"))
		if err != nil {
			return err
		}

		dbName = "postgres"
		if opts.PostgresURI != "" {
			parsedURL, parseErr := url.Parse(opts.PostgresURI)
			if parseErr != nil {
				return fmt.Errorf("error parsing postgres URI to retrieve DB name: invalid format")
			}
			if parsedURL.Path != "" && parsedURL.Path != "/" {
				dbName = parsedURL.Path[1:]
			}
		} else if azure.IsAzureAuthEnabled() {
			azCfg, cfgErr := azure.LoadAzurePostgresConfig()
			if cfgErr != nil {
				return fmt.Errorf("error loading Azure PostgreSQL config for migration DB name: %w", cfgErr)
			}
			// Database is guaranteed non-empty here because
			// LoadAzurePostgresConfig validates it as a required field.
			dbName = azCfg.Database
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
		if opts.Persist {
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
