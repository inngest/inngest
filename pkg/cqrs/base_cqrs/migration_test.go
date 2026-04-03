package base_cqrs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/inngest/inngest/tests/testutil"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*/*.sql
var legacyMigrationsFS embed.FS

type migrationDialect string

const (
	migrationDialectSQLite   migrationDialect = "sqlite"
	migrationDialectPostgres migrationDialect = "postgres"
)

type schemaSnapshot struct {
	Tables  map[string][]string
	Indexes map[string]map[string]string
}

func TestBaselineOnFreshSQLite(t *testing.T) {
	db, opts, cleanup := newRawMigrationTestDB(t, migrationDialectSQLite)
	defer cleanup()

	require.NoError(t, up(db, opts))

	actual := readApplicationSchemaSnapshot(t, db, migrationDialectSQLite)
	expected := expectedTableColumns(t, migrationDialectSQLite)
	require.Equal(t, expected, actual.Tables)
	assertGooseVersionTable(t, db)
}

func TestMigrationIdempotencySQLite(t *testing.T) {
	db, opts, cleanup := newRawMigrationTestDB(t, migrationDialectSQLite)
	defer cleanup()

	require.NoError(t, up(db, opts))
	before := readApplicationSchemaSnapshot(t, db, migrationDialectSQLite)

	require.NoError(t, up(db, opts))
	after := readApplicationSchemaSnapshot(t, db, migrationDialectSQLite)

	require.Equal(t, before, after)
	assertGooseVersionTable(t, db)
}

func TestSchemaMatchesSqlcSQLite(t *testing.T) {
	db, opts, cleanup := newRawMigrationTestDB(t, migrationDialectSQLite)
	defer cleanup()

	require.NoError(t, up(db, opts))

	actual := readApplicationSchemaSnapshot(t, db, migrationDialectSQLite)
	expected := expectedTableColumns(t, migrationDialectSQLite)
	require.Equal(t, expected, actual.Tables)
}

func TestLegacyMigrationThenGooseBaselineIsNoopSQLite(t *testing.T) {
	db, opts, cleanup := newRawMigrationTestDB(t, migrationDialectSQLite)
	defer cleanup()

	require.NoError(t, upLegacy(db, opts))
	before := readApplicationSchemaSnapshot(t, db, migrationDialectSQLite)

	require.NoError(t, up(db, opts))
	after := readApplicationSchemaSnapshot(t, db, migrationDialectSQLite)

	require.Equal(t, before, after)
	assertGooseVersionTable(t, db)
}

func TestBaselineOnFreshPostgres(t *testing.T) {
	db, opts, cleanup := newRawMigrationTestDB(t, migrationDialectPostgres)
	defer cleanup()

	require.NoError(t, up(db, opts))

	actual := readApplicationSchemaSnapshot(t, db, migrationDialectPostgres)
	expected := expectedTableColumns(t, migrationDialectPostgres)
	require.Equal(t, expected, actual.Tables)
	assertGooseVersionTable(t, db)
}

func TestMigrationIdempotencyPostgres(t *testing.T) {
	db, opts, cleanup := newRawMigrationTestDB(t, migrationDialectPostgres)
	defer cleanup()

	require.NoError(t, up(db, opts))
	before := readApplicationSchemaSnapshot(t, db, migrationDialectPostgres)

	require.NoError(t, up(db, opts))
	after := readApplicationSchemaSnapshot(t, db, migrationDialectPostgres)

	require.Equal(t, before, after)
	assertGooseVersionTable(t, db)
}

func TestSchemaMatchesSqlcPostgres(t *testing.T) {
	db, opts, cleanup := newRawMigrationTestDB(t, migrationDialectPostgres)
	defer cleanup()

	require.NoError(t, up(db, opts))

	actual := readApplicationSchemaSnapshot(t, db, migrationDialectPostgres)
	expected := expectedTableColumns(t, migrationDialectPostgres)
	require.Equal(t, expected, actual.Tables)
}

func TestLegacyMigrationThenGooseBaselineIsNoopPostgres(t *testing.T) {
	db, opts, cleanup := newRawMigrationTestDB(t, migrationDialectPostgres)
	defer cleanup()

	require.NoError(t, upLegacy(db, opts))
	before := readApplicationSchemaSnapshot(t, db, migrationDialectPostgres)

	require.NoError(t, up(db, opts))
	after := readApplicationSchemaSnapshot(t, db, migrationDialectPostgres)

	require.Equal(t, before, after)
	assertGooseVersionTable(t, db)
}

func newRawMigrationTestDB(t *testing.T, dialect migrationDialect) (*sql.DB, BaseCQRSOptions, func()) {
	t.Helper()

	ctx := context.Background()

	if dialect == migrationDialectPostgres {
		if os.Getenv(EnvTestDatabase) != string(migrationDialectPostgres) {
			t.Skip("set TEST_DATABASE=postgres to run postgres migration tests")
		}

		pc, err := testutil.StartPostgres(t)
		require.NoError(t, err)

		db, err := sql.Open("pgx", pc.URI)
		require.NoError(t, err)
		require.NoError(t, db.PingContext(ctx))

		cleanup := func() {
			_ = db.Close()
			if err := pc.Terminate(t.Context()); err != nil {
				t.Logf("failed to terminate postgres container: %v", err)
			}
		}
		return db, BaseCQRSOptions{PostgresURI: pc.URI, ForTest: true}, cleanup
	}

	dbName := fmt.Sprintf("sqlite_%s", strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	require.NoError(t, err)
	require.NoError(t, db.PingContext(ctx))

	return db, BaseCQRSOptions{ForTest: true}, func() {
		_ = db.Close()
	}
}

func upLegacy(db *sql.DB, opts BaseCQRSOptions) error {
	var (
		err    error
		src    source.Driver
		driver database.Driver
		dbName string
	)

	if opts.PostgresURI != "" {
		src, err = iofs.New(legacyMigrationsFS, path.Join("migrations", "postgres"))
		if err != nil {
			return err
		}

		dbName = "postgres"
		parsedURL, err := url.Parse(opts.PostgresURI)
		if err != nil {
			return fmt.Errorf("error parsing postgres URI to retrieve DB name: invalid format")
		}
		if parsedURL.Path != "" && parsedURL.Path != "/" {
			dbName = parsedURL.Path[1:]
		}

		driver, err = migratepostgres.WithInstance(db, &migratepostgres.Config{
			MigrationsTable: "migrations",
			DatabaseName:    dbName,
		})
		if err != nil {
			return err
		}
	} else {
		src, err = iofs.New(legacyMigrationsFS, path.Join("migrations", "sqlite"))
		if err != nil {
			return err
		}

		driver, err = migratesqlite.WithInstance(db, &migratesqlite.Config{
			MigrationsTable: "migrations",
			NoTxWrap:        true,
		})
		if err != nil {
			return err
		}

		dbName = "sqlite"
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
		if err := m.Migrate(v); err != nil {
			return fmt.Errorf("error migrating to version %d resetting dirty: %w", v, err)
		}
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		return nil
	}
	return err
}

func expectedTableColumns(t *testing.T, dialect migrationDialect) map[string][]string {
	t.Helper()

	schemaBytes, err := os.ReadFile(path.Join("sqlc", string(dialect), "schema.sql"))
	require.NoError(t, err)

	return parseTableColumns(t, string(schemaBytes))
}

func parseTableColumns(t *testing.T, schema string) map[string][]string {
	t.Helper()

	result := map[string][]string{}

	var currentTable string
	for _, rawLine := range strings.Split(schema, "\n") {
		line := strings.TrimSpace(rawLine)

		if idx := strings.Index(line, "--"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}

		if currentTable == "" {
			if !strings.HasPrefix(line, "CREATE TABLE ") {
				continue
			}

			parts := strings.Fields(line)
			require.GreaterOrEqual(t, len(parts), 3, "invalid CREATE TABLE statement: %q", line)
			currentTable = parts[2]
			result[currentTable] = nil
			continue
		}

		if line == ");" {
			currentTable = ""
			continue
		}

		line = strings.TrimSuffix(line, ",")
		if strings.HasPrefix(line, "PRIMARY KEY") || strings.HasPrefix(line, "UNIQUE") || strings.HasPrefix(line, "CONSTRAINT") {
			continue
		}

		parts := strings.Fields(line)
		require.NotEmpty(t, parts, "invalid column definition: %q", rawLine)
		result[currentTable] = append(result[currentTable], parts[0])
	}

	return result
}

func readApplicationSchemaSnapshot(t *testing.T, db *sql.DB, dialect migrationDialect) schemaSnapshot {
	t.Helper()

	tables := readTableColumns(t, db, dialect)
	indexes := map[string]map[string]string{}

	for tableName := range tables {
		indexes[tableName] = readIndexes(t, db, dialect, tableName)
	}

	return schemaSnapshot{
		Tables:  tables,
		Indexes: indexes,
	}
}

func readTableColumns(t *testing.T, db *sql.DB, dialect migrationDialect) map[string][]string {
	t.Helper()

	result := map[string][]string{}

	var (
		query string
		rows  *sql.Rows
		err   error
	)

	switch dialect {
	case migrationDialectPostgres:
		query = `
			SELECT table_name
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_type = 'BASE TABLE'
			  AND table_name NOT IN ('migrations', 'goose_db_version')
			ORDER BY table_name
		`
	default:
		query = `
			SELECT name
			FROM sqlite_master
			WHERE type = 'table'
			  AND name NOT LIKE 'sqlite_%'
			  AND name NOT IN ('migrations', 'goose_db_version')
			ORDER BY name
		`
	}

	rows, err = db.Query(query)
	require.NoError(t, err)
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tableNames = append(tableNames, name)
	}
	require.NoError(t, rows.Err())

	for _, tableName := range tableNames {
		switch dialect {
		case migrationDialectPostgres:
			columnRows, err := db.Query(`
				SELECT column_name
				FROM information_schema.columns
				WHERE table_schema = current_schema()
				  AND table_name = $1
				ORDER BY ordinal_position
			`, tableName)
			require.NoError(t, err)

			for columnRows.Next() {
				var columnName string
				require.NoError(t, columnRows.Scan(&columnName))
				result[tableName] = append(result[tableName], columnName)
			}
			require.NoError(t, columnRows.Err())
			require.NoError(t, columnRows.Close())
		default:
			columnRows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
			require.NoError(t, err)

			for columnRows.Next() {
				var (
					cid        int
					columnName string
					dataType   string
					notNull    int
					defaultVal sql.NullString
					pk         int
				)
				require.NoError(t, columnRows.Scan(&cid, &columnName, &dataType, &notNull, &defaultVal, &pk))
				result[tableName] = append(result[tableName], columnName)
			}
			require.NoError(t, columnRows.Err())
			require.NoError(t, columnRows.Close())
		}
	}

	return result
}

func readIndexes(t *testing.T, db *sql.DB, dialect migrationDialect, tableName string) map[string]string {
	t.Helper()

	result := map[string]string{}

	switch dialect {
	case migrationDialectPostgres:
		rows, err := db.Query(`
			SELECT indexname, indexdef
			FROM pg_indexes
			WHERE schemaname = current_schema()
			  AND tablename = $1
			ORDER BY indexname
		`, tableName)
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var name, definition string
			require.NoError(t, rows.Scan(&name, &definition))
			result[name] = definition
		}
		require.NoError(t, rows.Err())
	default:
		rows, err := db.Query(`
			SELECT name, sql
			FROM sqlite_master
			WHERE type = 'index'
			  AND tbl_name = ?
			  AND sql IS NOT NULL
			ORDER BY name
		`, tableName)
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var name, definition string
			require.NoError(t, rows.Scan(&name, &definition))
			result[name] = definition
		}
		require.NoError(t, rows.Err())
	}

	return result
}

func assertGooseVersionTable(t *testing.T, db *sql.DB) {
	t.Helper()

	var maxVersion int64
	require.NoError(t, db.QueryRow("SELECT MAX(version_id) FROM goose_db_version").Scan(&maxVersion))
	require.Equal(t, int64(1), maxVersion)

	var baselineRows int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id = 1").Scan(&baselineRows))
	require.Equal(t, 1, baselineRows)
}
