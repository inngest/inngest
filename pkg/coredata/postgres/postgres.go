package postgres

import (
	"context"
	"database/sql"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/inngest/client"
	"github.com/inngest/inngest-cli/pkg/config/registration"
	"github.com/inngest/inngest-cli/pkg/coredata"
	"github.com/inngest/inngest-cli/pkg/function"
	pg "gocloud.dev/postgres"
)

func init() {
	registration.RegisterDataStore(&Config{})
}

// Config registers the configuration for the PostgreSQL data store
type Config struct {
	URI string
}

func (c Config) DataStoreName() string {
	return "postgres"
}

func (c Config) ReadWriter(ctx context.Context) (coredata.ReadWriter, error) {
	return New(ctx, c.ConnectionString())
}

func (c Config) ConnectionString() string {
	// TODO Validation and/or combining separate fields into a connection string
	return c.URI
}

type ReadWriter struct {
	db *sql.DB
}

func New(ctx context.Context, URI string) (coredata.ReadWriter, error) {
	rw := &ReadWriter{}
	db, err := pg.Open(ctx, URI)
	if err != nil {
		return nil, err
	}
	rw.db = db
	return rw, nil
}

func (rw *ReadWriter) CreateFunctionVersion(ctx context.Context, f function.Function, live bool, env string) (function.FunctionVersion, error) {
	return function.FunctionVersion{}, nil
}
func (rw *ReadWriter) ActionVersion(ctx context.Context, dsn string, version *inngest.VersionConstraint) (client.ActionVersion, error) {
	return client.ActionVersion{}, nil
}
func (rw *ReadWriter) CreateActionVersion(ctx context.Context, av inngest.ActionVersion) (client.ActionVersion, error) {
	return client.ActionVersion{}, nil
}
func (rw *ReadWriter) UpdateActionVersion(ctx context.Context, dsn string, version inngest.VersionInfo, enabled bool) (client.ActionVersion, error) {
	return client.ActionVersion{}, nil
}
