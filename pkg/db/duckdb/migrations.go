package duckdb

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

// viewsSQL is the views and table macros createViews (re)creates on every
// Migrate, see views.sql.
//
//go:embed views.sql
var viewsSQL string

var viewsTmpl = template.Must(template.New("views.sql").Parse(viewsSQL))

// Mode is what kind of catalog Migrate runs against. Goose's ENVSUB can
// only substitute a variable or fall back to a default, never branch on a
// value, so each mode is the set of env vars (modeEnv) the migrations' "--
// +goose ENVSUB ON" blocks substitute: an unset var keeps a statement's
// in-file default. The views (views.sql) branch on the mode directly.
type Mode int

const (
	// ModeMemory is a bare in-memory catalog (devserver w/o --persist): no
	// lake ATTACH, so Migrate creates the "inngest" schema itself and the
	// tables skip their Iceberg-only partitioning/sort/format-version clause.
	ModeMemory Mode = iota
	// ModeRawLake is an Iceberg lake whose tables hold every emission
	// (devserver --persist): the migrations run unchanged.
	ModeRawLake
	// ModeFlatLake is an Iceberg lake whose runs are already one row per run
	// (IE exported from final rows only): as ModeRawLake, but raw_runs
	// doesn't collapse.
	ModeFlatLake
)

func (m Mode) String() string {
	switch m {
	case ModeMemory:
		return "memory"
	case ModeRawLake:
		return "raw_lake"
	case ModeFlatLake:
		return "flat_lake"
	}
	return fmt.Sprintf("Mode(%d)", int(m))
}

// modeEnv is the env vars each Mode sets while migrating.
//
// DUCKDB_ICEBERG_ONLY replaces each table's Iceberg-only clause
// (PARTITIONED BY/WITH), which a plain duckdb table rejects.
// Every occurrence uses the no-colon "${VAR-default}" form, which (unlike
// "${VAR:-default}") only falls back to the default when the var is
// completely unset, so an explicitly empty value wins.
var modeEnv = map[Mode]map[string]string{
	ModeMemory:   {"DUCKDB_ICEBERG_ONLY": ""},
	ModeRawLake:  {},
	ModeFlatLake: {},
}

// migrateMu serializes Migrate: a mode travels as process env vars, so two
// concurrent Migrates in different modes would see each other's.
var migrateMu sync.Mutex

// migrationModeTable records the Mode a catalog was migrated in. Each
// migration only ever runs once, so re-migrating a catalog in another mode
// would silently keep the first mode's tables; Migrate errors instead.
const migrationModeTable = "duckdb_migration_mode"

// Migrate applies every staging-table migration to db, then (re)creates the
// views and table macros over them (see createViews). It's safe to call
// more than once (idempotent) and is run once at subprocess startup before
// dual-write is enabled.
//
// goose v3.27.0 ships no built-in DuckDB dialect (Postgres, SQLite, MySQL,
// and others, but not DuckDB). goose.DialectSQLite3 was tried first, since
// DuckDB is highly SQL/SQLite-compatible, but it doesn't work: the
// SQLite3 dialect's version-table DDL uses "INTEGER PRIMARY KEY
// AUTOINCREMENT", which DuckDB's parser rejects outright (confirmed against
// the real duckdb binary: "Parser Error: syntax error at or near
// AUTOINCREMENT"). So this uses goose.DialectCustom instead, with a
// hand-written database.Store (duckdbStore, in store.go) that generates
// DuckDB-compatible SQL for the goose_db_version bookkeeping table — and
// (see store.go's doc comment) also works around a second, independently
// discovered incompatibility in this driver's row-column ordering.
//
// mode says what db is (see Mode). ModeMemory's db was opened with
// Options.DBFile ":memory:" and Options.Iceberg nil, so there is no lake
// ATTACH to create the "inngest" catalog and Migrate creates a schema of
// that name directly; the others' db was opened with Options.Iceberg set.
func Migrate(ctx context.Context, db *sql.DB, mode Mode) error {
	env, ok := modeEnv[mode]
	if !ok {
		return fmt.Errorf("duckdb: unknown migration mode %d", mode)
	}
	migrateMu.Lock()
	defer migrateMu.Unlock()

	if mode == ModeMemory {
		if _, err := db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+driver.LakeAlias+";"); err != nil {
			return fmt.Errorf("duckdb: creating in-memory schema %q: %w", driver.LakeAlias, err)
		}
	}
	if err := checkMode(ctx, db, mode); err != nil {
		return err
	}
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("duckdb: setting %s: %w", k, err)
		}
		defer func() { _ = os.Unsetenv(k) }()
	}

	migrationsFS, err := fs.Sub(MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("duckdb: reading embedded migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectCustom, db, migrationsFS,
		goose.WithStore(newDuckdbStore(goose.DefaultTablename)),
		// This POC's driver.Conn doesn't implement transactions (see
		// conn.go's Begin, which always errors), so DDL/DML must run
		// directly against the connection rather than inside a goose
		// BeginTx/Commit wrapper.
		goose.WithIsolateDDL(true),
	)
	if err != nil {
		return fmt.Errorf("duckdb: creating goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("duckdb: running migrations: %w", err)
	}
	return createViews(ctx, db, mode)
}

// createViews (re)creates the views and table macros in views.sql. They
// can't live in an Iceberg lake, so they're in a schema of duckdb's own
// database (schema.MacroSchema) and get re-created every Migrate instead of
// being a migration, so a definition change (or a mode change) never needs
// one.
func createViews(ctx context.Context, db *sql.DB, mode Mode) error {
	var b strings.Builder
	if err := viewsTmpl.Execute(&b, struct{ CollapseRuns bool }{mode != ModeFlatLake}); err != nil {
		return fmt.Errorf("duckdb: rendering views: %w", err)
	}
	for _, stmt := range splitStatements(b.String()) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("duckdb: creating views: %w", err)
		}
	}
	return nil
}

// splitStatements splits views.sql into statements: comment lines are
// dropped and a statement ends at a line ending in `;`.
func splitStatements(sqlText string) []string {
	var (
		stmts []string
		cur   []string
	)
	for _, line := range strings.Split(sqlText, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		cur = append(cur, line)
		if strings.HasSuffix(trimmed, ";") {
			stmts = append(stmts, strings.Join(cur, "\n"))
			cur = nil
		}
	}
	return stmts
}

// checkMode records mode as db's migration mode, or errors if db was already
// migrated in another one.
func checkMode(ctx context.Context, db *sql.DB, mode Mode) error {
	if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS "+migrationModeTable+" (mode VARCHAR NOT NULL);"); err != nil {
		return fmt.Errorf("duckdb: creating %s: %w", migrationModeTable, err)
	}
	var got string
	err := db.QueryRowContext(ctx, "SELECT mode FROM "+migrationModeTable+" LIMIT 1;").Scan(&got)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := db.ExecContext(ctx, "INSERT INTO "+migrationModeTable+" VALUES (?);", mode.String()); err != nil {
			return fmt.Errorf("duckdb: recording migration mode: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("duckdb: reading migration mode: %w", err)
	case got != mode.String():
		return fmt.Errorf("duckdb: catalog was migrated in mode %s, not %s", got, mode)
	}
	return nil
}
