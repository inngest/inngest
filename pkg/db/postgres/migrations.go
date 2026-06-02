package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
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

func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	if !strings.HasPrefix(opts.URI, "postgres://") && !strings.HasPrefix(opts.URI, "postgresql://") {
		if u, parseErr := url.Parse(opts.URI); parseErr == nil {
			return nil, fmt.Errorf("unsupported database URL: %s", u.Redacted())
		}
		return nil, fmt.Errorf("unsupported database URL format")
	}

	var (
		conn *sql.DB
		err  error
	)

	if opts.ForTest {
		conn, err = sql.Open("pgx", opts.URI)
	} else {
		openOnce.Do(func() {
			openDB, err = sql.Open("pgx", opts.URI)
		})
		conn = openDB
	}

	l := logger.StdlibLogger(ctx).With("db", "postgres")
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
