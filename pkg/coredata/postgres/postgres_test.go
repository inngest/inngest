package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	pg "gocloud.dev/postgres"
)

// var PostgresTestPort uint32 = 5439
var PostgresTestPort uint32 = 5438
var InMemoryPostgresURI string = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", PostgresTestPort)

var globalDB *sql.DB
var globalPGRW *ReadWriter

func setupPostgres() (func() error, error) {
	defaultConfig := embeddedpostgres.DefaultConfig()
	c := defaultConfig.Version(embeddedpostgres.V13).Port(PostgresTestPort)
	db := embeddedpostgres.NewDatabase(c)
	if err := db.Start(); err != nil {
		return nil, err
	}

	return func() error {
		return db.Stop()
	}, nil
}

func clearAllData(db *sql.DB) error {
	rows, err := db.Query(`select 'drop table if exists "' || tablename || '" cascade;' from pg_tables where schemaname = 'public';`)
	if err != nil {
		return err
	}
	var cmds []string
	for rows.Next() {
		var cmd string
		err := rows.Scan(&cmd)
		if err != nil {
			return err
		}
		cmds = append(cmds, cmd)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	dropAllTablesCmd := strings.Join(cmds[:], " ")
	_, err = db.Exec(dropAllTablesCmd)
	return err
}

// Set up Postgres and run migrations to prepare the database
func setup() func() error {
	var err error
	teardown, err := setupPostgres()
	if err != nil {
		panic(err)
	}

	globalDB, err = pg.Open(context.Background(), InMemoryPostgresURI)
	if err != nil {
		panic(err)
	}
	err = globalDB.Ping()
	if err != nil {
		panic(err)
	}

	// Reset any state that might be in the db
	err = clearAllData(globalDB)
	if err != nil {
		panic(err)
	}

	err = goose.Up(globalDB, "./migrations")
	if err != nil {
		panic(err)
	}

	globalPGRW, err = New(context.Background(), InMemoryPostgresURI)
	if err != nil {
		panic(err)
	}

	// return teardown
	return func() error {
		err := globalDB.Close()
		if err != nil {
			return err
		}
		err = globalPGRW.Close()
		if err != nil {
			return err
		}
		return teardown()
	}
}

// Setup and teardown postgres for all tests
func TestMain(m *testing.M) {
	teardown := setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func TestPostgresConnection(t *testing.T) {
	pgrw, err := New(context.Background(), InMemoryPostgresURI)
	require.NoError(t, err)
	err = pgrw.Close()
	require.NoError(t, err, "should close connection")
}

func TestActionVersion_exact_version(t *testing.T) {
	actionDSN := "test-action-exact-version-step-1"
	v := uint(1)
	config := "<config>"

	_, err := globalDB.ExecContext(context.Background(),
		`INSERT INTO action_versions (action_dsn, version_major, version_minor, config, valid_from)
		VALUES ($1, $2, $3, $4, $5)`,
		actionDSN, v, v, config, time.Now())
	require.NoError(t, err)

	av, err := globalPGRW.ActionVersion(context.Background(), actionDSN, &inngest.VersionConstraint{
		Major: &v,
		Minor: &v,
	})
	require.NoError(t, err)

	require.Equal(t, actionDSN, av.DSN)
	require.Equal(t, v, av.Version.Major)
	require.Equal(t, v, av.Version.Minor)
	require.Equal(t, config, av.Config)
}

func TestActionVersion_range_exact(t *testing.T) {
	actionDSN := "test-action-range-exact-step-1"
	v1 := uint(1)
	v2 := uint(2)
	v3 := uint(3)
	v4 := uint(4)
	config := "<config>"

	// Create 2 versions with valid from timestamps and one without
	_, err := globalDB.ExecContext(context.Background(),
		`INSERT INTO action_versions (action_dsn, version_major, version_minor, config, valid_from)
		VALUES ($1, $2, $3, $4, $5)`,
		actionDSN, v1, v1, config, time.Now().Add(-60))
	require.NoError(t, err)
	_, err = globalDB.ExecContext(context.Background(),
		`INSERT INTO action_versions (action_dsn, version_major, version_minor, config, valid_from)
		VALUES ($1, $2, $3, $4, $5)`,
		actionDSN, v1, v2, config, time.Now().Add(-30))
	require.NoError(t, err)
	_, err = globalDB.ExecContext(context.Background(),
		`INSERT INTO action_versions (action_dsn, version_major, version_minor, config, valid_from, valid_to)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		actionDSN, v1, v3, config, time.Now().Add(-20), time.Now().Add(-10))
	require.NoError(t, err)
	_, err = globalDB.ExecContext(context.Background(),
		`INSERT INTO action_versions (action_dsn, version_major, version_minor, config, valid_from)
		VALUES ($1, $2, $3, $4, $5)`,
		actionDSN, v1, v4, config, nil)
	require.NoError(t, err)

	// no version specified
	noversion, err := globalPGRW.ActionVersion(context.Background(), actionDSN, &inngest.VersionConstraint{})
	require.NoError(t, err)
	require.Equal(t, actionDSN, noversion.DSN)
	require.Equal(t, v1, noversion.Version.Major)
	require.Equal(t, v2, noversion.Version.Minor)
	require.Equal(t, config, noversion.Config)

	// major version specified
	majorversion, err := globalPGRW.ActionVersion(context.Background(), actionDSN, &inngest.VersionConstraint{
		Major: &v1,
	})
	require.NoError(t, err)
	require.Equal(t, actionDSN, majorversion.DSN)
	require.Equal(t, v1, majorversion.Version.Major)
	require.Equal(t, v2, majorversion.Version.Minor)
	require.Equal(t, config, majorversion.Config)

	// exact version, marked valid
	exactvalid, err := globalPGRW.ActionVersion(context.Background(), actionDSN, &inngest.VersionConstraint{
		Major: &v1,
		Minor: &v1,
	})
	require.NoError(t, err)
	require.Equal(t, actionDSN, exactvalid.DSN)
	require.Equal(t, v1, exactvalid.Version.Major)
	require.Equal(t, v1, exactvalid.Version.Minor)
	require.Equal(t, config, exactvalid.Config)

	// exact version, has been marked invalid
	exactinvalid, err := globalPGRW.ActionVersion(context.Background(), actionDSN, &inngest.VersionConstraint{
		Major: &v1,
		Minor: &v3,
	})
	require.NoError(t, err)
	require.Equal(t, actionDSN, exactinvalid.DSN)
	require.Equal(t, v1, exactinvalid.Version.Major)
	require.Equal(t, v3, exactinvalid.Version.Minor)
	require.Equal(t, config, exactinvalid.Config)

	// exact version, not yet valid
	exactunpublished, err := globalPGRW.ActionVersion(context.Background(), actionDSN, &inngest.VersionConstraint{
		Major: &v1,
		Minor: &v4,
	})
	require.NoError(t, err)
	require.Equal(t, actionDSN, exactunpublished.DSN)
	require.Equal(t, v1, exactunpublished.Version.Major)
	require.Equal(t, v4, exactunpublished.Version.Minor)
	require.Equal(t, config, exactunpublished.Config)
}

func TestCreateActionVersion_single(t *testing.T) {
	actionDSN := "test-create-action-single-step-1"

	av, err := globalPGRW.CreateActionVersion(context.Background(), inngest.ActionVersion{
		DSN: actionDSN,
		Version: &inngest.VersionInfo{
			Major: uint(1),
			Minor: uint(1),
		},
	})
	require.NoError(t, err)
	require.Equal(t, actionDSN, av.DSN)
	require.Equal(t, uint(1), av.Version.Major)
	require.Equal(t, uint(1), av.Version.Minor)
	require.Containsf(t, av.Config, actionDSN, "config should contain dsn")
	require.Nil(t, av.ValidFrom)
	require.Nil(t, av.ValidTo)
	require.NotNil(t, av.CreatedAt)

	// Fetch from the db
	v := uint(1)
	fromdb, err := globalPGRW.ActionVersion(context.Background(), actionDSN, &inngest.VersionConstraint{
		Major: &v,
		Minor: &v,
	})
	require.NoError(t, err)
	require.Equal(t, actionDSN, fromdb.DSN)
}

func TestCreateActionVersion_multiple(t *testing.T) {
	actionDSN := "test-create-action-multiple-step-1"

	av1, err := globalPGRW.CreateActionVersion(context.Background(), inngest.ActionVersion{
		DSN: actionDSN,
		Version: &inngest.VersionInfo{
			Major: uint(1),
			Minor: uint(1),
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint(1), av1.Version.Major)
	require.Equal(t, uint(1), av1.Version.Minor)

	av2, err := globalPGRW.CreateActionVersion(context.Background(), inngest.ActionVersion{
		DSN: actionDSN,
		Version: &inngest.VersionInfo{
			Major: uint(1),
			Minor: uint(2),
		},
	})
	require.NoError(t, err, "should allow actions with different versions")
	require.Equal(t, uint(1), av2.Version.Major)
	require.Equal(t, uint(2), av2.Version.Minor)
}

func TestCreateActionVersion_without_version(t *testing.T) {
	actionDSN := "test-create-action-without-version-step-1"

	_, err := globalPGRW.CreateActionVersion(context.Background(), inngest.ActionVersion{
		DSN: actionDSN,
	})
	require.ErrorContains(t, err, "version must not be empty")
}

func TestCreateActionVersion_reject_duplicate(t *testing.T) {
	av := inngest.ActionVersion{
		DSN: "test-create-action-duplicate-step-1",
		Version: &inngest.VersionInfo{
			Major: uint(1),
			Minor: uint(2),
		},
	}

	_, err := globalPGRW.CreateActionVersion(context.Background(), av)
	require.NoError(t, err)

	_, err = globalPGRW.CreateActionVersion(context.Background(), av)
	require.Error(t, err)
	require.ErrorContains(t, err, "existing action version")
}

func TestUpdateActionVersion_enable_new(t *testing.T) {
	actionDSN := "test-update-action-enable-new-step-1"
	versionInfo := &inngest.VersionInfo{
		Major: uint(1),
		Minor: uint(10),
	}

	av, err := globalPGRW.CreateActionVersion(context.Background(), inngest.ActionVersion{
		DSN:     actionDSN,
		Version: versionInfo,
	})
	require.NoError(t, err)
	require.Equal(t, actionDSN, av.DSN)
	require.Nil(t, av.ValidFrom)

	updated, err := globalPGRW.UpdateActionVersion(context.Background(), actionDSN, *versionInfo, true)
	require.NoError(t, err)
	require.Equal(t, actionDSN, updated.DSN)
	require.NotNil(t, updated.ValidFrom)
	require.Nil(t, updated.ValidTo)
}

func TestUpdateActionVersion_dont_enable(t *testing.T) {
	actionDSN := "test-update-action-dont-enable-new-step-1"
	versionInfo := &inngest.VersionInfo{
		Major: uint(1),
		Minor: uint(10),
	}

	av, err := globalPGRW.CreateActionVersion(context.Background(), inngest.ActionVersion{
		DSN:     actionDSN,
		Version: versionInfo,
	})
	require.NoError(t, err)
	require.Equal(t, actionDSN, av.DSN)
	require.Nil(t, av.ValidFrom)

	updated, err := globalPGRW.UpdateActionVersion(context.Background(), actionDSN, *versionInfo, false)
	require.NoError(t, err)
	require.Equal(t, actionDSN, updated.DSN)
	require.Nil(t, updated.ValidFrom, "should not have been enabled")
	require.Nil(t, updated.ValidTo)
}

func TestUpdateActionVersion_disable(t *testing.T) {
	actionDSN := "test-update-action-disable-new-step-1"
	versionInfo := &inngest.VersionInfo{
		Major: uint(1),
		Minor: uint(10),
	}

	av, err := globalPGRW.CreateActionVersion(context.Background(), inngest.ActionVersion{
		DSN:     actionDSN,
		Version: versionInfo,
	})
	require.NoError(t, err)
	require.Equal(t, actionDSN, av.DSN)
	require.Nil(t, av.ValidFrom)

	// first enable it before disabling
	enabled, err := globalPGRW.UpdateActionVersion(context.Background(), actionDSN, *versionInfo, true)
	require.NoError(t, err)
	require.Equal(t, actionDSN, enabled.DSN)
	require.NotNil(t, enabled.ValidFrom)
	require.Nil(t, enabled.ValidTo)

	// disable it
	disabled, err := globalPGRW.UpdateActionVersion(context.Background(), actionDSN, *versionInfo, false)
	require.NoError(t, err)
	require.Equal(t, actionDSN, disabled.DSN)
	require.NotNil(t, disabled.ValidFrom)
	require.NotNil(t, disabled.ValidTo, "should have been disabled")
}

// func TestFunctions(t *testing.T) {
// 	pgrw, err := acquirePostgresReadWriter(context.Background(), t)
// 	require.NoError(t, err)

// 	fns, err := globalPGRW.Functions(context.Background())
// 	require.NoError(t, err)
// 	fmt.Println(fns)
// }

// func TestFunctionsScheduled(t *testing.T) {
// 	pgrw, err := acquirePostgresReadWriter(context.Background(), t)
// 	require.NoError(t, err)

// 	fns, err := globalPGRW.FunctionsScheduled(context.Background())
// 	require.NoError(t, err)
// 	fmt.Println(fns)
// }

// func TestFunctionsByTrigger(t *testing.T) {
// 	pgrw, err := acquirePostgresReadWriter(context.Background(), t)
// 	require.NoError(t, err)

// 	eventName := "test.event"
// 	fns, err := globalPGRW.FunctionsByTrigger(context.Background(), eventName)
// 	require.NoError(t, err)
// 	fmt.Println("Matching test.event:")
// 	fmt.Println(fns)
// }
