package base_cqrs

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	dbpostgres "github.com/inngest/inngest/pkg/db/postgres"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
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

func New(ctx context.Context, opts BaseCQRSOptions) (*sql.DB, error) {
	if opts.PostgresURI != "" {
		if !strings.HasPrefix(opts.PostgresURI, "postgres://") && !strings.HasPrefix(opts.PostgresURI, "postgresql://") {
			if u, parseErr := url.Parse(opts.PostgresURI); parseErr == nil {
				return nil, fmt.Errorf("unsupported database URL: %s", u.Redacted())
			}
			return nil, fmt.Errorf("unsupported database URL format")
		}

		return dbpostgres.Open(ctx, dbpostgres.Options{
			URI:     opts.PostgresURI,
			ForTest: opts.ForTest,
		})
	}

	return dbsqlite.Open(ctx, dbsqlite.Options{
		Persist:   opts.Persist,
		ForTest:   opts.ForTest,
		Directory: opts.Directory,
	})
}
