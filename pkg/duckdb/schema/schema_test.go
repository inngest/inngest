package schema

import (
	"path/filepath"
	"testing"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/stretchr/testify/require"
)

// TestMatchesDuckDBMigrations guards against drift between this package and
// the lake tables the migrations actually create: same columns, same order,
// same declared types and nullability.
func TestMatchesDuckDBMigrations(t *testing.T) {
	binPath := driver.RequireDuckDBBinary(t)
	dir := t.TempDir()
	db, err := driver.Open(t.Context(), driver.Options{
		BinaryPath: binPath, DBFile: ":memory:",
		DuckLake: &driver.DuckLakeOptions{CatalogPath: filepath.Join(dir, "catalog.ducklake"), DataPath: filepath.Join(dir, "data")},
	})
	if err != nil {
		t.Skipf("duckdb unavailable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, duckdb.Migrate(t.Context(), db, true))

	for _, tbl := range []Table{Runs, Spans, Events} {
		t.Run(tbl.Name, func(t *testing.T) {
			rows, err := db.QueryContext(t.Context(), `SELECT column_name, data_type, is_nullable = 'YES' FROM information_schema.columns
				WHERE table_catalog = 'inngest' AND table_name = ? ORDER BY ordinal_position`, tbl.Name)
			require.NoError(t, err)
			defer rows.Close()
			type column struct {
				name, typ string
				nullable  bool
			}
			var got, want []column
			for rows.Next() {
				var c column
				require.NoError(t, rows.Scan(&c.name, &c.typ, &c.nullable))
				got = append(got, c)
			}
			require.NoError(t, rows.Err())
			for _, c := range tbl.Columns {
				want = append(want, column{c.Name, c.PhysicalType(), c.Nullable})
			}
			require.Equal(t, want, got)
		})
	}
}
