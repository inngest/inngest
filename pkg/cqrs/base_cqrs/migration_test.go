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
	"sort"
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

type schemaColumn struct {
	Name    string
	Type    string
	NotNull bool
	Default string
}

type schemaSnapshot struct {
	Tables  map[string][]schemaColumn
	Indexes map[string][]string
}

func TestBaselineOnFreshSQLite(t *testing.T) {
	db, opts, cleanup := newRawMigrationTestDB(t, migrationDialectSQLite)
	defer cleanup()

	require.NoError(t, up(db, opts))

	assertMatchesExpectedSchema(t, db, migrationDialectSQLite)
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

	assertMatchesExpectedSchema(t, db, migrationDialectSQLite)
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

	assertMatchesExpectedSchema(t, db, migrationDialectPostgres)
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

	assertMatchesExpectedSchema(t, db, migrationDialectPostgres)
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

func assertMatchesExpectedSchema(t *testing.T, db *sql.DB, dialect migrationDialect) {
	t.Helper()

	actual := readApplicationSchemaSnapshot(t, db, dialect)
	expected := expectedSchemaSnapshot(t, dialect)
	require.Equal(t, expected, actual)
}

func expectedSchemaSnapshot(t *testing.T, dialect migrationDialect) schemaSnapshot {
	t.Helper()

	schemaBytes, err := os.ReadFile(path.Join("sqlc", string(dialect), "schema.sql"))
	require.NoError(t, err)

	tables := parseSchemaColumns(t, string(schemaBytes))
	indexes := parseIndexNames(t, string(schemaBytes))
	for tableName := range tables {
		if _, ok := indexes[tableName]; !ok {
			indexes[tableName] = nil
		}
	}

	return schemaSnapshot{
		Tables:  tables,
		Indexes: indexes,
	}
}

func parseSchemaColumns(t *testing.T, schema string) map[string][]schemaColumn {
	t.Helper()

	result := map[string][]schemaColumn{}
	for _, statement := range splitSQLStatements(schema) {
		statement = strings.TrimSpace(statement)
		if !strings.HasPrefix(strings.ToUpper(statement), "CREATE TABLE ") {
			continue
		}

		tableName, definitions := parseCreateTableStatement(t, statement)
		for _, definition := range splitTopLevel(definitions, ',') {
			column, ok := parseSchemaColumnLine(definition)
			if !ok {
				continue
			}
			result[tableName] = append(result[tableName], column)
		}
	}

	return result
}

func parseIndexNames(t *testing.T, schema string) map[string][]string {
	t.Helper()

	result := map[string][]string{}

	for _, statement := range splitSQLStatements(schema) {
		statement = strings.TrimSpace(statement)
		upper := strings.ToUpper(statement)
		if !strings.HasPrefix(upper, "CREATE INDEX ") && !strings.HasPrefix(upper, "CREATE UNIQUE INDEX ") {
			continue
		}

		remainder := statement
		switch {
		case strings.HasPrefix(upper, "CREATE UNIQUE INDEX "):
			remainder = strings.TrimSpace(statement[len("CREATE UNIQUE INDEX "):])
		case strings.HasPrefix(upper, "CREATE INDEX "):
			remainder = strings.TrimSpace(statement[len("CREATE INDEX "):])
		}

		if strings.HasPrefix(strings.ToUpper(remainder), "IF NOT EXISTS ") {
			remainder = strings.TrimSpace(remainder[len("IF NOT EXISTS "):])
		}

		onIdx := strings.Index(strings.ToUpper(remainder), " ON ")
		require.NotEqual(t, -1, onIdx, "invalid CREATE INDEX statement: %q", statement)

		indexName := normalizeIdentifier(remainder[:onIdx])
		tableSection := strings.TrimSpace(remainder[onIdx+len(" ON "):])
		if usingIdx := strings.Index(strings.ToUpper(tableSection), " USING "); usingIdx >= 0 {
			tableSection = strings.TrimSpace(tableSection[:usingIdx])
		}
		tableNameSection := tableSection
		if parenIdx := strings.Index(tableNameSection, "("); parenIdx >= 0 {
			tableNameSection = tableNameSection[:parenIdx]
		}

		tableName := normalizeIdentifier(tableNameSection)
		result[tableName] = append(result[tableName], indexName)
	}

	for table := range result {
		sort.Strings(result[table])
	}
	return result
}

func parseCreateTableStatement(t *testing.T, statement string) (string, string) {
	t.Helper()

	remainder := strings.TrimSpace(statement[len("CREATE TABLE "):])
	if strings.HasPrefix(strings.ToUpper(remainder), "IF NOT EXISTS ") {
		remainder = strings.TrimSpace(remainder[len("IF NOT EXISTS "):])
	}

	openIdx := strings.Index(remainder, "(")
	require.NotEqual(t, -1, openIdx, "invalid CREATE TABLE statement: %q", statement)

	tableName := normalizeIdentifier(remainder[:openIdx])
	body := strings.TrimSpace(remainder[openIdx+1:])
	if strings.HasSuffix(body, ")") {
		body = strings.TrimSpace(body[:len(body)-1])
	}

	return tableName, body
}

func parseSchemaColumnLine(definition string) (schemaColumn, bool) {
	line := strings.TrimSpace(strings.TrimSuffix(definition, ","))
	upper := strings.ToUpper(line)
	if line == "" || strings.HasPrefix(upper, "PRIMARY KEY") || strings.HasPrefix(upper, "UNIQUE") || strings.HasPrefix(upper, "CONSTRAINT") {
		return schemaColumn{}, false
	}

	nameEnd := strings.IndexAny(line, " \t")
	if nameEnd == -1 {
		return schemaColumn{}, false
	}

	name := normalizeIdentifier(line[:nameEnd])
	remainder := strings.TrimSpace(line[nameEnd+1:])
	typeEnd := len(remainder)
	upperRemainder := strings.ToUpper(remainder)
	for _, marker := range []string{" DEFAULT ", " NOT NULL", " PRIMARY KEY", " UNIQUE", " CHECK", " REFERENCES", " CONSTRAINT"} {
		if idx := strings.Index(upperRemainder, marker); idx >= 0 && idx < typeEnd {
			typeEnd = idx
		}
	}

	defaultExpr := ""
	if idx := strings.Index(upperRemainder, " DEFAULT "); idx >= 0 {
		defaultExpr = strings.TrimSpace(remainder[idx+len(" DEFAULT "):])
		upperDefault := strings.ToUpper(defaultExpr)
		for _, marker := range []string{" NOT NULL", " PRIMARY KEY", " UNIQUE", " CHECK", " REFERENCES", " CONSTRAINT"} {
			if end := strings.Index(upperDefault, marker); end >= 0 {
				defaultExpr = strings.TrimSpace(defaultExpr[:end])
				break
			}
		}
	}

	return schemaColumn{
		Name:    name,
		Type:    normalizeType(strings.TrimSpace(remainder[:typeEnd])),
		NotNull: strings.Contains(upperRemainder, " NOT NULL") || strings.Contains(upperRemainder, " PRIMARY KEY"),
		Default: normalizeDefault(defaultExpr),
	}, true
}

func readApplicationSchemaSnapshot(t *testing.T, db *sql.DB, dialect migrationDialect) schemaSnapshot {
	t.Helper()

	tables := readRuntimeSchema(t, db, dialect)
	indexes := map[string][]string{}

	for tableName := range tables {
		indexes[tableName] = readIndexes(t, db, dialect, tableName)
	}

	return schemaSnapshot{
		Tables:  tables,
		Indexes: indexes,
	}
}

func readRuntimeSchema(t *testing.T, db *sql.DB, dialect migrationDialect) map[string][]schemaColumn {
	t.Helper()

	tableNames := readRuntimeTableNames(t, db, dialect)
	result := make(map[string][]schemaColumn, len(tableNames))

	for _, tableName := range tableNames {
		result[tableName] = readRuntimeColumns(t, db, dialect, tableName)
	}

	return result
}

func readRuntimeTableNames(t *testing.T, db *sql.DB, dialect migrationDialect) []string {
	t.Helper()

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
			  AND table_name <> 'goose_db_version'
			ORDER BY table_name
		`
	default:
		query = `
			SELECT name
			FROM sqlite_master
			WHERE type = 'table'
			  AND name NOT LIKE 'sqlite_%'
			  AND name <> 'goose_db_version'
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

	return tableNames
}

func readRuntimeColumns(t *testing.T, db *sql.DB, dialect migrationDialect, tableName string) []schemaColumn {
	t.Helper()

	switch dialect {
	case migrationDialectPostgres:
		rows, err := db.Query(`
			SELECT column_name, data_type, is_nullable, column_default, character_maximum_length
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = $1
			ORDER BY ordinal_position
		`, tableName)
		require.NoError(t, err)
		defer rows.Close()

		var columns []schemaColumn
		for rows.Next() {
			var (
				name          string
				dataType      string
				isNullable    string
				defaultValue  sql.NullString
				maxCharLength sql.NullInt64
			)
			require.NoError(t, rows.Scan(&name, &dataType, &isNullable, &defaultValue, &maxCharLength))
			columns = append(columns, schemaColumn{
				Name:    name,
				Type:    normalizeType(postgresColumnType(dataType, maxCharLength)),
				NotNull: isNullable == "NO",
				Default: normalizeDefault(defaultValue.String),
			})
		}
		require.NoError(t, rows.Err())
		return columns
	default:
		rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, tableName))
		require.NoError(t, err)
		defer rows.Close()

		var columns []schemaColumn
		for rows.Next() {
			var (
				cid          int
				name         string
				dataType     string
				notNull      int
				defaultValue sql.NullString
				primaryKey   int
			)
			require.NoError(t, rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey))
			columns = append(columns, schemaColumn{
				Name:    name,
				Type:    normalizeType(dataType),
				NotNull: notNull == 1 || primaryKey > 0,
				Default: normalizeDefault(defaultValue.String),
			})
		}
		require.NoError(t, rows.Err())
		return columns
	}
}

func readIndexes(t *testing.T, db *sql.DB, dialect migrationDialect, tableName string) []string {
	t.Helper()

	var (
		rows *sql.Rows
		err  error
	)

	switch dialect {
	case migrationDialectPostgres:
		rows, err = db.Query(`
			SELECT i.indexname
			FROM pg_indexes i
			LEFT JOIN pg_constraint c
			  ON c.conindid = (i.schemaname || '.' || i.indexname)::regclass
			  AND c.contype = 'p'
			WHERE i.schemaname = current_schema()
			  AND i.tablename = $1
			  AND c.oid IS NULL
			ORDER BY i.indexname
		`, tableName)
	default:
		rows, err = db.Query(`
			SELECT name
			FROM sqlite_master
			WHERE type = 'index'
			  AND tbl_name = ?
			  AND sql IS NOT NULL
			ORDER BY name
		`, tableName)
	}
	require.NoError(t, err)
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		indexes = append(indexes, name)
	}
	require.NoError(t, rows.Err())
	sort.Strings(indexes)
	return indexes
}

func splitSQLStatements(schema string) []string {
	return splitTopLevel(stripLineComments(schema), ';')
}

func stripLineComments(schema string) string {
	lines := strings.Split(schema, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func splitTopLevel(input string, separator rune) []string {
	var (
		result   []string
		start    int
		depth    int
		inString bool
		prevRune rune
	)

	for idx, r := range input {
		switch r {
		case '\'':
			if prevRune != '\\' {
				inString = !inString
			}
		case '(':
			if !inString {
				depth++
			}
		case ')':
			if !inString && depth > 0 {
				depth--
			}
		}

		if r == separator && !inString && depth == 0 {
			part := strings.TrimSpace(input[start:idx])
			if part != "" {
				result = append(result, part)
			}
			start = idx + 1
		}

		prevRune = r
	}

	if tail := strings.TrimSpace(input[start:]); tail != "" {
		result = append(result, tail)
	}

	return result
}

func normalizeIdentifier(name string) string {
	name = strings.TrimSpace(strings.Trim(name, `"`))
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	return strings.Trim(name, `"`)
}

func postgresColumnType(dataType string, maxCharLength sql.NullInt64) string {
	switch dataType {
	case "character varying":
		if maxCharLength.Valid {
			return fmt.Sprintf("varchar(%d)", maxCharLength.Int64)
		}
		return "varchar"
	case "character":
		if maxCharLength.Valid {
			return fmt.Sprintf("char(%d)", maxCharLength.Int64)
		}
		return "char"
	case "timestamp without time zone":
		return "timestamp"
	case "timestamp with time zone":
		return "timestamptz"
	default:
		return dataType
	}
}

func normalizeType(dataType string) string {
	dataType = strings.ToLower(strings.TrimSpace(dataType))
	dataType = strings.Join(strings.Fields(dataType), " ")
	dataType = strings.ReplaceAll(dataType, "character varying", "varchar")
	dataType = strings.ReplaceAll(dataType, "character(", "char(")
	dataType = strings.ReplaceAll(dataType, "character", "char")
	dataType = strings.ReplaceAll(dataType, "integer", "int")
	dataType = strings.ReplaceAll(dataType, "boolean", "bool")
	dataType = strings.ReplaceAll(dataType, "timestamp without time zone", "timestamp")
	dataType = strings.ReplaceAll(dataType, "timestamp with time zone", "timestamptz")
	return dataType
}

func normalizeDefault(defaultExpr string) string {
	defaultExpr = strings.TrimSpace(defaultExpr)
	for strings.HasPrefix(defaultExpr, "(") && strings.HasSuffix(defaultExpr, ")") {
		defaultExpr = strings.TrimSpace(defaultExpr[1 : len(defaultExpr)-1])
	}
	if idx := strings.Index(defaultExpr, "::"); idx >= 0 {
		defaultExpr = defaultExpr[:idx]
	}
	defaultExpr = strings.TrimSpace(defaultExpr)
	if defaultExpr == "" {
		return ""
	}
	if strings.HasPrefix(defaultExpr, "'") {
		return defaultExpr
	}
	return strings.ToLower(defaultExpr)
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
