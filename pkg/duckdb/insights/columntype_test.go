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
		{"BLOB", ColumnTypeUnknown},
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
