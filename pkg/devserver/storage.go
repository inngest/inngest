package devserver

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	cqrsmanager "github.com/inngest/inngest/pkg/cqrs/manager"
	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/db/driverhelp"
	dbpostgres "github.com/inngest/inngest/pkg/db/postgres"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/tracing"
	"go.uber.org/fx"
)

// database is the SQLite or Postgres database of the server.
type database interface {
	dbpkg.Adapter
	Helpers() driverhelp.DialectHelpers
}

// newDatabase opens Postgres when StartOpts.PostgresURI is set, and SQLite
// otherwise.  the postgres and sqlite packages keep one connection pool for
// the process, so the pool stays open when the app stops.  a second server
// in the same process, such as in a test, uses the same pool.
func newDatabase(ctx context.Context, opts StartOpts) (database, error) {
	if opts.PostgresURI != "" {
		// Pool settings are in minutes, mirroring the postgres-conn-max-* flags.
		db, err := dbpostgres.Open(ctx, dbpostgres.Options{
			URI:             opts.PostgresURI,
			MaxIdleConns:    opts.PostgresMaxIdleConns,
			MaxOpenConns:    opts.PostgresMaxOpenConns,
			ConnMaxIdleTime: time.Duration(opts.PostgresConnMaxIdleTime) * time.Minute,
			ConnMaxLifetime: time.Duration(opts.PostgresConnMaxLifetime) * time.Minute,
		})
		if err != nil {
			return nil, err
		}
		return dbpostgres.New(db), nil
	}

	db, err := dbsqlite.Open(ctx, dbsqlite.Options{
		Persist:   opts.Persist,
		Directory: opts.SQLiteDir,
	})
	if err != nil {
		return nil, err
	}
	return dbsqlite.New(db), nil
}

type cqrsResult struct {
	fx.Out

	Manager cqrs.Manager
	History history.Driver
}

func newCQRS(db database) cqrsResult {
	return cqrsResult{
		Manager: cqrsmanager.New(db),
		History: base_cqrs.NewHistoryDriver(db),
	}
}

func newTracerProvider(db database) tracing.TracerProvider {
	return tracing.NewSqlcTracerProvider(db.Q())
}
