package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/internal/cuedefs"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/coredata"
	"github.com/inngest/inngest/pkg/function"
	"github.com/lib/pq"
	pg "gocloud.dev/postgres"
)

func init() {
	registration.RegisterDataStore(func() any { return &Config{} })
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

func New(ctx context.Context, URI string) (*ReadWriter, error) {
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

func (rw ReadWriter) Close() error {
	return rw.db.Close()
}

// TODO Add method to close the db connection

var (
	// action_versions
	sqlFindExactMatchingActionVersion string = `
		SELECT action_dsn, version_major, version_minor, config, valid_from, valid_to, created_at
		FROM action_versions
		WHERE action_dsn = $1 and version_major = $2 and version_minor = $3`
	sqlFindLatestValidMajorActionVersion string = `
		SELECT action_dsn, version_major, version_minor, config, valid_from, valid_to, created_at
		FROM action_versions
		WHERE action_dsn = $1 and version_major = $2 and valid_from is not null and valid_to is null
		ORDER BY version_minor DESC
		LIMIT 1`
	sqlFindLatestValidActionVersion string = `
		SELECT action_dsn, version_major, version_minor, config, valid_from, valid_to, created_at
		FROM action_versions
		WHERE action_dsn = $1 and valid_from is not null and valid_to is null
		ORDER BY version_major, version_minor DESC
		LIMIT 1`
	sqlInsertActionVersion string = `
		INSERT INTO action_versions (action_dsn, version_major, version_minor, config)
		VALUES ($1, $2, $3, $4)
		RETURNING action_dsn, version_major, version_minor, config, valid_from, valid_to, created_at`
	sqlUpdateActionVersionValidFrom string = `
		UPDATE action_versions
		SET valid_from = $4
		WHERE action_dsn = $1 and version_major = $2 and version_minor = $3
		RETURNING action_dsn, version_major, version_minor, config, valid_from, valid_to, created_at`
	sqlUpdateActionVersionValidTo string = `
		UPDATE action_versions
		SET valid_to = $4
		WHERE action_dsn = $1 and version_major = $2 and version_minor = $3
		RETURNING action_dsn, version_major, version_minor, config, valid_from, valid_to, created_at`

	// functions
	sqlInsertFunction string = `
		INSERT INTO functions (function_id, name)
		VALUES ($1, $2)`

	// function_versions
	sqlFindAllLiveFunctionVersions string = `
		SELECT f.function_id, fv.version, fv.config
		FROM functions f
		JOIN function_versions fv on f.function_id = fv.function_id
		WHERE fv.valid_from is not null and fv.valid_to is null`
	sqlFindAllLiveScheduledFunctions string = `
		SELECT fv.function_id, fv.version, fv.config
		FROM function_triggers ft
		JOIN function_versions fv on fv.function_id = ft.function_id and fv.version = ft.version
		WHERE ft.schedule is not null and fv.valid_from is not null and fv.valid_to is null`
	sqlFindAllLiveFunctionsByEvent string = `
		SELECT fv.function_id, fv.version, fv.config
		FROM function_triggers ft
		JOIN function_versions fv on fv.function_id = ft.function_id and fv.version = ft.version
		WHERE ft.event_name = $1 and fv.valid_from is not null and fv.valid_to is null;`
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
		INSERT INTO function_triggers (function_id, version, event_name)
		VALUES ($1, $2, $3)`
	sqlInsertScheduleTrigger string = `
		INSERT INTO function_triggers (function_id, version, schedule)
		VALUES ($1, $2, $3)`
)

// CreateFunctionVersion creates the function, ensures function_triggers are up to date,
// and creates a new function version, setting any prior version no longer valid.
func (rw *ReadWriter) CreateFunctionVersion(ctx context.Context, f function.Function, live bool, env string) (function.FunctionVersion, error) {
	var existingFunctionID string
	var existingVersion int
	now := time.Now()

	err := rw.db.QueryRowContext(ctx, sqlFindLatestFunctionVersion, f.ID).
		Scan(&existingFunctionID, &existingVersion)
	if err != nil && err != sql.ErrNoRows {
		return function.FunctionVersion{}, err
	}

	// Bump the version - existingVersion is 0 if no rows are found (via COALESCE)
	newFunctionVersion := uint(existingVersion + 1)

	// TODO - Diff the existing function vs. the new function and only add new version if it has changed

	tx, err := rw.db.BeginTx(ctx, nil)
	if err != nil {
		return function.FunctionVersion{}, err
	}

	// We confirm there is no existing row in the functions table before creating one
	if existingFunctionID == "" {
		_, err := tx.ExecContext(ctx, sqlInsertFunction, f.ID, f.Name)
		if err != nil {
			return function.FunctionVersion{}, err
		}
	}

	// For live functions, we must make the previous version as no longer valid
	// NOTE - We currently have no "draft" functions in the open source Inngest, this is for future draft functionality
	// Every new function version deployed is assumed to be live
	if live && existingVersion != 0 {
		_, err := tx.ExecContext(ctx, sqlUpdateFunctionVersionValidTo, f.ID, existingVersion, now)
		if err != nil {
			return function.FunctionVersion{}, err
		}
	}

	// Create the function version
	config, err := function.MarshalCUE(f)
	if err != nil {
		return function.FunctionVersion{}, err
	}
	fv := function.FunctionVersion{
		FunctionID: f.ID,
		Version:    newFunctionVersion,
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

	// Create all function_triggers for the new version
	for _, trigger := range f.Triggers {
		var err error
		if trigger.EventTrigger != nil {
			_, err = tx.ExecContext(ctx, sqlInsertEventTrigger, f.ID, newFunctionVersion, trigger.Event)
		} else if trigger.CronTrigger != nil {
			_, err = tx.ExecContext(ctx, sqlInsertScheduleTrigger, f.ID, newFunctionVersion, trigger.Cron)
		}
		if err != nil {
			return function.FunctionVersion{}, err
		}
	}

	if err = tx.Commit(); err != nil {
		if err = tx.Rollback(); err != nil {
			return function.FunctionVersion{}, err
		}
		return function.FunctionVersion{}, err
	}

	return fv, nil
}

func rowsToFunctions(ctx context.Context, rows *sql.Rows) ([]function.Function, error) {
	fns := []function.Function{}

	for rows.Next() {
		fv := function.FunctionVersion{}
		err := rows.Scan(&fv.FunctionID, &fv.Version, &fv.Config)
		if err != nil {
			return []function.Function{}, err
		}
		// Parse the cue string
		fn, err := function.Unmarshal(ctx, []byte(fv.Config), "")
		if err != nil {
			return nil, err
		}
		fns = append(fns, *fn)
	}
	// check any rows during iteration
	err := rows.Err()
	if err != nil {
		return []function.Function{}, err
	}
	return fns, nil
}

func (rw *ReadWriter) Functions(ctx context.Context) ([]function.Function, error) {
	rows, err := rw.db.QueryContext(ctx, sqlFindAllLiveFunctionVersions)
	if err != nil {
		return []function.Function{}, err
	}
	defer rows.Close()
	return rowsToFunctions(ctx, rows)
}
func (rw *ReadWriter) FunctionsScheduled(ctx context.Context) ([]function.Function, error) {
	rows, err := rw.db.QueryContext(ctx, sqlFindAllLiveScheduledFunctions)
	if err != nil {
		return []function.Function{}, err
	}
	defer rows.Close()
	return rowsToFunctions(ctx, rows)
}
func (rw *ReadWriter) FunctionsByTrigger(ctx context.Context, eventName string) ([]function.Function, error) {
	rows, err := rw.db.QueryContext(ctx, sqlFindAllLiveFunctionsByEvent, eventName)
	if err != nil {
		return []function.Function{}, err
	}
	defer rows.Close()
	return rowsToFunctions(ctx, rows)
}

func (rw *ReadWriter) ActionVersion(ctx context.Context, dsn string, version *inngest.VersionConstraint) (client.ActionVersion, error) {
	av := client.ActionVersion{}
	v := inngest.VersionInfo{}

	var row *sql.Row
	if version.Major == nil && version.Minor == nil {
		// No version constraint - get the latest valid
		row = rw.db.QueryRowContext(ctx, sqlFindLatestValidActionVersion, dsn)
	} else if version.Major != nil && version.Minor == nil {
		// No minor version constraint - get the latest valid matching the major version
		row = rw.db.QueryRowContext(ctx, sqlFindLatestValidMajorActionVersion, dsn, version.Major)
	} else if version.Major != nil && version.Minor != nil {
		// Exact constraint - get the exact match
		row = rw.db.QueryRowContext(ctx, sqlFindExactMatchingActionVersion, dsn, version.Major, version.Minor)
	}

	err := row.Scan(&av.DSN, &v.Major, &v.Minor, &av.Config, &av.ValidFrom, &av.ValidTo, &av.CreatedAt)
	if err != nil && err != sql.ErrNoRows {
		return client.ActionVersion{}, err
	}
	if err == sql.ErrNoRows {
		return client.ActionVersion{}, coredata.ErrActionVersionNotFound
	}
	av.Version = &v

	return av, nil
}

func (rw *ReadWriter) Action(ctx context.Context, dsn string, version *inngest.VersionConstraint) (*inngest.ActionVersion, error) {
	av, err := rw.ActionVersion(ctx, dsn, version)
	if err != nil {
		return nil, err
	}

	parsed, err := cuedefs.ParseAction(av.Config)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func (rw *ReadWriter) CreateActionVersion(ctx context.Context, av inngest.ActionVersion) (client.ActionVersion, error) {
	config, err := cuedefs.FormatAction(av)
	if err != nil {
		return client.ActionVersion{}, err
	}

	created := client.ActionVersion{}
	created.ActionVersion = av

	if created.Version == nil {
		return client.ActionVersion{}, errors.New("version must not be empty")
	}

	// NOTE - We do not allow valid_from to be set when creating a version as the client needs to push a container image
	// to the registry before calling UpdateActionVersion
	err = rw.db.QueryRowContext(ctx, sqlInsertActionVersion, av.DSN, av.Version.Major, av.Version.Minor, config).
		Scan(&created.DSN, &created.Version.Major, &created.Version.Minor, &created.Config,
			&created.ValidFrom, &created.ValidTo, &created.CreatedAt)
	if err != nil {
		if err, ok := err.(*pq.Error); ok && err.Code.Name() == "unique_violation" {
			return client.ActionVersion{},
				fmt.Errorf("existing action version found for %s:%d-%d", av.DSN, av.Version.Major, av.Version.Minor)
		}
		return client.ActionVersion{}, err
	}

	return created, nil
}
func (rw *ReadWriter) UpdateActionVersion(ctx context.Context, dsn string, version inngest.VersionInfo, enabled bool) (client.ActionVersion, error) {

	vc := &inngest.VersionConstraint{Major: &version.Major, Minor: &version.Minor}
	existing, err := rw.ActionVersion(ctx, dsn, vc)
	if err != nil {
		return client.ActionVersion{}, errors.New("no existing action version to update")
	}
	// if it's already been enabled, or we should not enable, just return
	if (existing.ValidFrom != nil && enabled) || (existing.ValidFrom == nil && !enabled) {
		return existing, nil
	}

	av := client.ActionVersion{}
	v := inngest.VersionInfo{}

	// Set the valid from or valid to depending on enabled
	if existing.ValidFrom == nil && enabled {
		err = rw.db.QueryRowContext(ctx, sqlUpdateActionVersionValidFrom, dsn, version.Major, version.Minor, time.Now()).
			Scan(&av.DSN, &v.Major, &v.Minor, &av.Config, &av.ValidFrom, &av.ValidTo, &av.CreatedAt)
		if err != nil {
			return client.ActionVersion{}, err
		}
	} else if existing.ValidFrom != nil && !enabled {
		err = rw.db.QueryRowContext(ctx, sqlUpdateActionVersionValidTo, dsn, version.Major, version.Minor, time.Now()).
			Scan(&av.DSN, &v.Major, &v.Minor, &av.Config, &av.ValidFrom, &av.ValidTo, &av.CreatedAt)
		if err != nil {
			return client.ActionVersion{}, err
		}
	}

	// Set the Version
	av.Version = &v
	return av, nil
}
