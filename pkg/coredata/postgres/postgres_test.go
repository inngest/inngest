package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/coredata"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	pg "gocloud.dev/postgres"
)

var PostgresTestPort uint32 = 5439
var InMemoryPostgresURI string = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", PostgresTestPort)

func setupPostgres(t *testing.T) func(t *testing.T) {
	defaultConfig := embeddedpostgres.DefaultConfig()
	c := defaultConfig.Port(PostgresTestPort)
	db := embeddedpostgres.NewDatabase(c)
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}

	return func(t *testing.T) {
		if err := db.Stop(); err != nil {
			t.Fatal(err)
		}
	}
}

func acquireReadWriter(ctx context.Context, t *testing.T) (coredata.ReadWriter, error) {
	pgrw, err := New(context.Background(), InMemoryPostgresURI)
	return pgrw, err
}

func connect(ctx context.Context, t *testing.T) *sql.DB {
	db, err := pg.Open(ctx, InMemoryPostgresURI)
	require.NoError(t, err)
	return db
}

func runMigrations(db *sql.DB, t *testing.T) {
	if err := goose.Up(db, "./migrations"); err != nil {
		t.Fatal(err)
	}
}

// setup prepares each test and exposes different interfaces
func setup(t *testing.T) (coredata.ReadWriter, *sql.DB, func(t *testing.T)) {
	teardown := setupPostgres(t)
	pgrw, err := acquireReadWriter(context.Background(), t)
	require.NoError(t, err)
	db := connect(context.Background(), t)
	runMigrations(db, t)
	return pgrw, db, teardown
}

func TestPostgresConnection(t *testing.T) {
	teardown := setupPostgres(t)
	defer teardown(t)

	_, err := New(context.Background(), InMemoryPostgresURI)
	require.NoError(t, err)
}

func TestGooseMirgrations(t *testing.T) {
	teardown := setupPostgres(t)
	defer teardown(t)

	db := connect(context.Background(), t)

	err := goose.Up(db, "./migrations")
	require.NoError(t, err)
}

func TestActionVersion_single(t *testing.T) {
	pgrw, db, teardown := setup(t)
	defer teardown(t)

	actionDSN := "test-action-step-1"
	v := uint(1)
	config := "<config>"

	_, err := db.ExecContext(context.Background(),
		`INSERT INTO action_versions (action_dsn, version_major, version_minor, config, valid_from)
		VALUES ($1, $2, $3, $4, $5)`,
		actionDSN, v, v, config, time.Now())
	require.NoError(t, err)

	av, err := pgrw.ActionVersion(context.Background(), actionDSN, &inngest.VersionConstraint{
		Major: &v,
		Minor: &v,
	})
	require.NoError(t, err)
	fmt.Println("ActionVersion")

	require.Equal(t, actionDSN, av.DSN)
	require.Equal(t, v, av.Version.Major)
	require.Equal(t, v, av.Version.Minor)
	require.Equal(t, config, av.Config)
}

func TestActionVersion_latest_version(t *testing.T) {
	pgrw, db, teardown := setup(t)
	defer teardown(t)

	actionDSN := "test-action-step-1"
	v1 := uint(1)
	v2 := uint(2)
	config := "<config>"

	_, err := db.ExecContext(context.Background(),
		`INSERT INTO action_versions (action_dsn, version_major, version_minor, config, valid_from)
		VALUES ($1, $2, $3, $4, $5)`,
		actionDSN, v1, v1, config, time.Now())
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO action_versions (action_dsn, version_major, version_minor, config, valid_from)
		VALUES ($1, $2, $3, $4, $5)`,
		actionDSN, v1, v2, config, time.Now())
	require.NoError(t, err)

	av, err := pgrw.ActionVersion(context.Background(), actionDSN, &inngest.VersionConstraint{
		Major: &v1,
	})
	require.NoError(t, err)
	fmt.Println("ActionVersion")

	require.Equal(t, actionDSN, av.DSN)
	require.Equal(t, v1, av.Version.Major)
	require.Equal(t, v2, av.Version.Minor)
	require.Equal(t, config, av.Config)
}

// func TestFunctions(t *testing.T) {
// 	pgrw, err := acquirePostgresReadWriter(context.Background(), t)
// 	require.NoError(t, err)

// 	fns, err := pgrw.Functions(context.Background())
// 	require.NoError(t, err)
// 	fmt.Println(fns)
// }

// func TestFunctionsScheduled(t *testing.T) {
// 	pgrw, err := acquirePostgresReadWriter(context.Background(), t)
// 	require.NoError(t, err)

// 	fns, err := pgrw.FunctionsScheduled(context.Background())
// 	require.NoError(t, err)
// 	fmt.Println(fns)
// }

// func TestFunctionsByTrigger(t *testing.T) {
// 	pgrw, err := acquirePostgresReadWriter(context.Background(), t)
// 	require.NoError(t, err)

// 	eventName := "test.event"
// 	fns, err := pgrw.FunctionsByTrigger(context.Background(), eventName)
// 	require.NoError(t, err)
// 	fmt.Println("Matching test.event:")
// 	fmt.Println(fns)
// }
