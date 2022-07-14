package postgres

import (
	"context"
	"database/sql"
	"time"

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
	err = db.Ping()
	if err != nil {
		return rw, err
	}
	rw.db = db
	return rw, nil
}

// TODO Add method to close the db connection

var (
	// functions
	sqlInsertFunction string = `
		INSERT INTO functions (function_id, name)
		VALUES ($1, $2)`

	// function_versions
	sqlFindLatestFunctionVersion string = `
		SELECT f.function_id, COALESCE(version,0)
		FROM functions f
		LEFT JOIN function_versions fv on f.function_id = fv.function_id
		WHERE f.function_id = $1
		ORDER BY version DESC
		LIMIT 1;`
	sqlInsertFunctionVersion string = `
		INSERT INTO function_versions (function_id, version, config, valid_from)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`
	sqlUpdateFunctionVersionValidTo string = `
		UPDATE function_versions
		SET valid_to = $3
		WHERE function_id = $1 and version = $2`

	// function_triggers
	sqlInsertEventTrigger string = `
		INSERT INTO function_triggers (function_id, event_name)
		VALUES ($1, $2)`
	sqlInsertScheduleTrigger string = `
		INSERT INTO function_triggers (function_id, schedule)
		VALUES ($1, $2)`
	sqlDeleteTriggers string = `
		DELETE FROM function_triggers
		WHERE function_id=$1`
)

// CreateFunctionVersion creates the function, ensures function_triggers are up to date,
// and creates a new function version, setting any prior version no longer valid.
func (rw *ReadWriter) CreateFunctionVersion(ctx context.Context, f function.Function, live bool, env string) (function.FunctionVersion, error) {
	// NOTE - We currently have no "draft" functions in the open source Inngest, this is for future draft functionality
	// Every new function version deployed is assumed to be live
	live = true

	var existingFunctionID string
	var existingVersion int
	now := time.Now()

	err := rw.db.QueryRowContext(ctx, sqlFindLatestFunctionVersion, f.ID).
		Scan(&existingFunctionID, &existingVersion)
	if err != nil && err != sql.ErrNoRows {
		return function.FunctionVersion{}, err
	}

	tx, err := rw.db.BeginTx(ctx, nil)
	if err != nil {
		return function.FunctionVersion{}, err
	}
	defer tx.Rollback()

	// We confirm there is no existing row in the functions table before creating one
	if existingFunctionID == "" {
		_, err := tx.ExecContext(ctx, sqlInsertFunction, f.ID, f.Name)
		if err != nil {
			return function.FunctionVersion{}, err
		}
	}

	// For live functions, we must update the function triggers and make the previous version as no longer valid
	if live {
		// Clear any old triggers and create the current ones - this is simpler than finding which ones need to be deleted/created
		_, err := tx.ExecContext(ctx, sqlDeleteTriggers, f.ID)
		if err != nil {
			return function.FunctionVersion{}, err
		}

		// Insert the currently valid triggers
		for _, trigger := range f.Triggers {
			var err error
			if trigger.EventTrigger != nil {
				_, err = tx.ExecContext(ctx, sqlInsertEventTrigger, f.ID, trigger.Event)
			} else if trigger.CronTrigger != nil {
				_, err = tx.ExecContext(ctx, sqlInsertScheduleTrigger, f.ID, trigger.Cron)
			}
			if err != nil {
				return function.FunctionVersion{}, err
			}
		}

		// Make prior version no longer valid if there is a previous version
		if existingVersion != 0 {
			_, err := tx.ExecContext(ctx, sqlUpdateFunctionVersionValidTo, f.ID, existingVersion, now)
			if err != nil {
				return function.FunctionVersion{}, err
			}
		}

	}

	// Create the function version
	config, err := function.MarshalJSON(f)
	if err != nil {
		return function.FunctionVersion{}, err
	}
	fv := function.FunctionVersion{
		FunctionID: f.ID,
		Version:    uint(existingVersion + 1),
		Config:     string(config),
		Function:   f,
	}
	if live {
		fv.ValidFrom = &now
	}

	err = tx.QueryRowContext(ctx, sqlInsertFunctionVersion, f.ID, fv.Version, fv.Config, fv.ValidFrom).
		Scan(&fv.CreatedAt, &fv.UpdatedAt)
	if err != nil || err == sql.ErrNoRows {
		return function.FunctionVersion{}, err
	}

	if err = tx.Commit(); err != nil {
		return function.FunctionVersion{}, err
	}

	return fv, nil
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
