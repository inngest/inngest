package insights

import (
	"database/sql"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/stretchr/testify/require"
)

// newSchemaTestDuckDB duplicates pkg/duckdb/insights/views_test.go's
// (package insights_test) unexported helper of the same shape -- can't be
// imported here since it's package-private in a different package. Closes
// itself via t.Cleanup rather than returning a cleanup func, since every
// caller here is a single top-level test with no need to close early.
func newSchemaTestDuckDB(t *testing.T) *sql.DB {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	dir := t.TempDir()
	db, err := duckdb.Open(t.Context(), duckdb.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &duckdb.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
	})
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := duckdb.Migrate(t.Context(), db, true); err != nil {
		t.Fatalf("migrating duckdb: %v", err)
	}
	return db
}

// TestLogicalTablesMatchLiveMacroSchema proves every logicalTable's
// columnOrder/columns registry (tables.go) exactly matches what its real
// DuckDB macro (pkg/db/duckdb/migrations/000002_insights_views.sql) actually
// projects -- catching a class of drift TestLogicalTablesColumnOrderMatchesColumns
// can't: that test only checks tables.go against itself, so a column added
// (or left, or removed) consistently in both columnOrder and columns but
// never touched in the macro's own SELECT list (or vice versa) still passes
// it, while validate would then accept a column reference the macro doesn't
// actually project -- exactly the "inserted_at" bug this test exists to
// catch a recurrence of. Queries LIMIT 0 and inspects rows.ColumnTypes()
// directly, the same path Execute itself uses (execute.go's
// resultColumns), rather than parsing DESCRIBE's text output.
func TestLogicalTablesMatchLiveMacroSchema(t *testing.T) {
	db := newSchemaTestDuckDB(t)
	accountID, envID := uuid.New(), uuid.New()

	for name, tbl := range logicalTables {
		t.Run(name, func(t *testing.T) {
			rows, err := db.QueryContext(t.Context(),
				"SELECT * FROM inngest."+tbl.view+"(?, ?) LIMIT 0;", accountID.String(), envID.String())
			require.NoError(t, err)
			defer rows.Close()

			colTypes, err := rows.ColumnTypes()
			require.NoError(t, err)

			liveNames := make([]string, len(colTypes))
			for i, ct := range colTypes {
				liveNames[i] = ct.Name()
			}
			require.Equalf(t, tbl.columnOrder, liveNames,
				"table %s: columnOrder must match inngest.%s's actual projected columns, in order", name, tbl.view)

			for _, ct := range colTypes {
				col, ok := tbl.columns[ct.Name()]
				require.Truef(t, ok, "table %s: column %q missing from columns map", name, ct.Name())
				live := DuckDBToColumnType(ct.DatabaseTypeName())
				require.Equalf(t, col.colType, live,
					"table %s: column %q declared as ColumnType %v but inngest.%s reports %v (DuckDB type %s)",
					name, ct.Name(), col.colType, tbl.view, live, ct.DatabaseTypeName())
			}
		})
	}
}
