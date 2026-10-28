package insights

import "testing"

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
