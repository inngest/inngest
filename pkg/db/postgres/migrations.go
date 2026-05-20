package postgres

import (
	"context"
	"crypto/tls"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/azure"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

var (
	openOnce sync.Once
	openDB   *sql.DB
)

type Options struct {
	// URI declares the postgres connection to connect to a postgres database.
	URI string
	// ForTest indicates that the database handler is created for testing
	// purposes. By default database handlers are singletons, but when this flag
	// is enabled, each call creates a new connection.
	ForTest bool
}

// openAzurePostgres creates a *sql.DB using Azure Workload Identity authentication.
func openAzurePostgres() (*sql.DB, error) {
	cfg, err := azure.LoadAzurePostgresConfig()
	if err != nil {
		return nil, err
	}

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

func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	azureAuth := azure.IsAzureAuthEnabled()

	if azureAuth && opts.URI != "" {
		return nil, fmt.Errorf("cannot use both Azure Workload Identity (AZURE_POSTGRESQL_HOST) and PostgresURI; choose one authentication method")
	}

	var (
		conn *sql.DB
		err  error
	)

	l := logger.StdlibLogger(ctx).With("db", "postgres")

	if azureAuth {
		if opts.ForTest {
			conn, err = openAzurePostgres()
		} else {
			openOnce.Do(func() {
				openDB, err = openAzurePostgres()
			})
			conn = openDB
		}
		l = l.With("auth", "azure-workload-identity")
	} else {
		if !strings.HasPrefix(opts.URI, "postgres://") && !strings.HasPrefix(opts.URI, "postgresql://") {
			if u, parseErr := url.Parse(opts.URI); parseErr == nil {
				return nil, fmt.Errorf("unsupported database URL: %s", u.Redacted())
			}
			return nil, fmt.Errorf("unsupported database URL format")
		}

		if opts.ForTest {
			conn, err = sql.Open("pgx", opts.URI)
		} else {
			openOnce.Do(func() {
				openDB, err = sql.Open("pgx", opts.URI)
			})
			conn = openDB
		}
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

	provider, err := goose.NewProvider(goose.DialectPostgres, conn, migrationsFS)
	if err != nil {
		return err
	}

	_, err = provider.Up(ctx)
	return err
}
