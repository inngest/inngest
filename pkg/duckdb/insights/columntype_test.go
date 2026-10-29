package insights

import (
	"os/exec"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/stretchr/testify/require"
)

func TestDuckDBToColumnType(t *testing.T) {
	cases := []struct {
		duckdbType string
		want       ColumnType
	}{
		{"VARCHAR", ColumnTypeString},
		{"CHAR", ColumnTypeString},
		{"BPCHAR", ColumnTypeString},
		{"TEXT", ColumnTypeString},
		{"UUID", ColumnTypeString},
		{"TINYINT", ColumnTypeNumber},
		{"SMALLINT", ColumnTypeNumber},
		{"INTEGER", ColumnTypeNumber},
		{"BIGINT", ColumnTypeNumber},
		{"HUGEINT", ColumnTypeNumber},
		{"UBIGINT", ColumnTypeNumber},
		{"FLOAT", ColumnTypeNumber},
		{"DOUBLE", ColumnTypeNumber},
		{"DECIMAL(10,2)", ColumnTypeNumber},
		{"BOOLEAN", ColumnTypeBoolean},
		{"DATE", ColumnTypeDatetime},
		{"TIME", ColumnTypeDatetime},
		{"TIMESTAMP", ColumnTypeDatetime},
		{"TIMESTAMP_MS", ColumnTypeDatetime},
		{"TIMESTAMP_NS", ColumnTypeDatetime},
		{"TIMESTAMP_S", ColumnTypeDatetime},
		{"TIMESTAMPTZ", ColumnTypeDatetime},
		{"INTERVAL", ColumnTypeDatetime},
		{"JSON", ColumnTypeJSON},
		{"VARCHAR[]", ColumnTypeJSON},
		{"STRUCT(key VARCHAR, id VARCHAR)[]", ColumnTypeJSON},
		{"MAP(VARCHAR, INTEGER)", ColumnTypeJSON},
		{"BLOB", ColumnTypeString},
		// Aliases and parameterized forms, as a query's own CAST can
		// spell them (each checked with typeof(NULL::<alias>)).
		{"INT", ColumnTypeNumber},
		{"int8", ColumnTypeNumber},
		{"LONG", ColumnTypeNumber},
		{"REAL", ColumnTypeNumber},
		{"NUMERIC", ColumnTypeNumber},
		{"NUMERIC(10,2)", ColumnTypeNumber},
		{"DOUBLE PRECISION", ColumnTypeNumber},
		{"BIGNUM", ColumnTypeNumber},
		{"BOOL", ColumnTypeBoolean},
		{"STRING", ColumnTypeString},
		{"VARCHAR(10)", ColumnTypeString},
		{"BYTEA", ColumnTypeString},
		{"ENUM('a', 'b')", ColumnTypeString},
		{"DATETIME", ColumnTypeDatetime},
		{"TIME WITH TIME ZONE", ColumnTypeDatetime},
		{"TIMETZ", ColumnTypeDatetime},
		{"TIMESTAMP WITH TIME ZONE", ColumnTypeDatetime},
		{"INTEGER[3]", ColumnTypeJSON},
		{"UNION(a INTEGER, b VARCHAR)", ColumnTypeJSON},
		// Any spacing/casing DuckDB accepts in a CAST (each checked with
		// typeof(NULL::<type>)).
		{"STRUCT (a INT)", ColumnTypeJSON},
		{"struct(a int)", ColumnTypeJSON},
		{"MAP (VARCHAR, INT)", ColumnTypeJSON},
		{"UNION (a INT, b VARCHAR)", ColumnTypeJSON},
		{"ENUM ('a', 'b')", ColumnTypeString},
		{"DECIMAL (10, 2)", ColumnTypeNumber},
		{"VARCHAR (10)", ColumnTypeString},
		{"INTEGER []", ColumnTypeJSON},
		{"INTEGER [ 3 ]", ColumnTypeJSON},
		{"STRUCT (a INT) []", ColumnTypeJSON},
		{"  double\n\tprecision ", ColumnTypeNumber},
		{"timestamp  with   time zone", ColumnTypeDatetime},
		{"CHARACTER VARYING", ColumnTypeString},
		{"char varying(10)", ColumnTypeString},
		{"CHARACTER", ColumnTypeString},
		{"NATIONAL CHARACTER VARYING", ColumnTypeString},
		{"GUID", ColumnTypeString},
		{"INT64", ColumnTypeNumber},
		{"INTEGRAL", ColumnTypeNumber},
		{"OID", ColumnTypeNumber},
		{"DEC(10,2)", ColumnTypeNumber},
		{"ROW(a INT)", ColumnTypeJSON},
		{"INTERVAL YEAR", ColumnTypeDatetime},
		{"INTERVAL DAY TO SECOND", ColumnTypeDatetime},
		{"BIT", ColumnTypeUnknown},
		{"SOMETHING_UNRECOGNIZED", ColumnTypeUnknown},
	}
	for _, c := range cases {
		t.Run(c.duckdbType, func(t *testing.T) {
			if got := DuckDBToColumnType(c.duckdbType); got != c.want {
				t.Errorf("DuckDBToColumnType(%q) = %v, want %v", c.duckdbType, got, c.want)
			}
		})
	}
}

// sqlStandardTypeAliases are type spellings DuckDB's parser accepts that
// duckdb_types() doesn't list.
var sqlStandardTypeAliases = []string{
	"CHARACTER", "CHARACTER VARYING", "CHAR VARYING", "CHARACTER VARYING(10)",
	"NCHAR", "NCHAR VARYING", "NATIONAL CHARACTER", "NATIONAL CHAR",
	"NATIONAL CHARACTER VARYING", "NATIONAL CHAR VARYING",
	"DOUBLE PRECISION", "TIME WITHOUT TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE",
	"INTERVAL YEAR", "INTERVAL DAY TO SECOND", "ROW(a INT)", "FLOAT(10)", "TIMESTAMP(3)",
}

// TestDuckDBTypeAliasesMatchDuckDB checks every type name duckdb_types()
// lists (plus sqlStandardTypeAliases) buckets the same as the canonical
// name a real duckdb reports for it (typeof(NULL::<alias>)). Names that
// need type parameters (LIST, MAP, ...) error in DuckDB and are skipped.
func TestDuckDBTypeAliasesMatchDuckDB(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	db, err := driver.Open(t.Context(), driver.Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	rows, err := db.QueryContext(t.Context(), "SELECT DISTINCT type_name FROM duckdb_types() ORDER BY type_name")
	require.NoError(t, err)
	var names []string
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		names = append(names, n)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	require.NotEmpty(t, names)

	for _, alias := range append(names, sqlStandardTypeAliases...) {
		t.Run(alias, func(t *testing.T) {
			var canonical string
			if err := db.QueryRowContext(t.Context(), "SELECT typeof(NULL::"+alias+")").Scan(&canonical); err != nil {
				t.Skipf("not castable on its own: %v", err)
			}
			require.Equal(t, DuckDBToColumnType(canonical), DuckDBToColumnType(alias),
				"%q is %q in duckdb", alias, canonical)
		})
	}
}
