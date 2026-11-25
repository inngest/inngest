package schema

import (
	"database/sql"
	"testing"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/icebergcatalog/icebergtest"
	"github.com/stretchr/testify/require"
)

type column struct {
	name, typ string
	nullable  bool
}

// TestMatchesDuckDBMigrations guards against drift between this package and
// the lake tables the migrations actually create: same columns, same order,
// same declared types and nullability.
//
// NOTE: an Iceberg table only shows up in DESCRIBE (not
// information_schema/duckdb_columns), and DESCRIBE reports every Iceberg
// column as nullable even though NOT NULL is enforced, so nullability is
// checked against the in-memory mode's plain tables (same DDL minus the
// Iceberg-only clause).
func TestMatchesDuckDBMigrations(t *testing.T) {
	binPath := driver.RequireDuckDBBinary(t)
	open := func(t *testing.T, opts driver.Options, mode duckdb.Mode) *sql.DB {
		opts.BinaryPath, opts.DBFile = binPath, ":memory:"
		db, err := driver.Open(t.Context(), opts)
		if err != nil {
			t.Skipf("duckdb unavailable: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		require.NoError(t, duckdb.Migrate(t.Context(), db, mode))
		return db
	}
	want := func(tbl Table, withNullable bool) []column {
		var cols []column
		for _, c := range tbl.Columns {
			cols = append(cols, column{c.Name, c.PhysicalType(), c.Nullable && withNullable})
		}
		return cols
	}

	t.Run("lake", func(t *testing.T) {
		db := open(t, driver.Options{Iceberg: icebergtest.Start(t)}, duckdb.ModeRawLake)
		for _, tbl := range []Table{Runs, Spans, Events, RunMetadata} {
			t.Run(tbl.Name, func(t *testing.T) {
				rows, err := db.QueryContext(t.Context(), "SELECT column_name, column_type FROM (DESCRIBE "+driver.LakeAlias+"."+tbl.Name+");")
				require.NoError(t, err)
				defer rows.Close()
				var got []column
				for rows.Next() {
					var c column
					require.NoError(t, rows.Scan(&c.name, &c.typ))
					got = append(got, c)
				}
				require.NoError(t, rows.Err())
				require.Equal(t, want(tbl, false), got)
			})
		}
	})

	t.Run("memory", func(t *testing.T) {
		db := open(t, driver.Options{}, duckdb.ModeMemory)
		for _, tbl := range []Table{Runs, Spans, Events, RunMetadata} {
			t.Run(tbl.Name, func(t *testing.T) {
				rows, err := db.QueryContext(t.Context(), `SELECT column_name, data_type, is_nullable = 'YES' FROM information_schema.columns
					WHERE table_schema = 'inngest' AND table_name = ? ORDER BY ordinal_position`, tbl.Name)
				require.NoError(t, err)
				defer rows.Close()
				var got []column
				for rows.Next() {
					var c column
					require.NoError(t, rows.Scan(&c.name, &c.typ, &c.nullable))
					got = append(got, c)
				}
				require.NoError(t, rows.Err())
				require.Equal(t, want(tbl, true), got)
			})
		}
	})
}
