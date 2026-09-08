package insights

import "strings"

// ColumnType is the GQL-facing type bucket for one result column — see
// docs/plans/010-duckdb-insights-query-layer.md's GQL schema section
// (InsightsColumnType). UUID isn't its own bucket: it prints/compares as a
// string, so it reports STRING. DuckDB's composite types (T[] lists,
// STRUCT/MAP) report JSON: they're nested data a client renders the same
// way as any other JSON value.
type ColumnType int

const (
	ColumnTypeString ColumnType = iota
	ColumnTypeNumber
	ColumnTypeBoolean
	ColumnTypeDatetime
	ColumnTypeJSON
	ColumnTypeUnknown
)

// DuckDBToColumnType maps a *sql.ColumnType.DatabaseTypeName() string (as
// DuckDB's driver reports it) to a ColumnType bucket — see docs/plans/010's
// GQL schema section's mapping table.
func DuckDBToColumnType(duckdbType string) ColumnType {
	t := strings.ToUpper(duckdbType)

	switch {
	case t == "VARCHAR", t == "CHAR", t == "BPCHAR", t == "TEXT", t == "UUID":
		return ColumnTypeString
	case t == "BOOLEAN":
		return ColumnTypeBoolean
	case t == "DATE", t == "TIME", t == "INTERVAL",
		strings.HasPrefix(t, "TIMESTAMP"):
		return ColumnTypeDatetime
	case t == "JSON", strings.HasSuffix(t, "[]"), strings.HasPrefix(t, "STRUCT("), strings.HasPrefix(t, "MAP("):
		return ColumnTypeJSON
	case isNumberType(t):
		return ColumnTypeNumber
	default:
		return ColumnTypeUnknown
	}
}

func isNumberType(t string) bool {
	switch {
	case t == "TINYINT", t == "SMALLINT", t == "INTEGER", t == "BIGINT", t == "HUGEINT",
		t == "UTINYINT", t == "USMALLINT", t == "UINTEGER", t == "UBIGINT", t == "UHUGEINT",
		t == "FLOAT", t == "DOUBLE":
		return true
	case strings.HasPrefix(t, "DECIMAL"):
		return true
	default:
		return false
	}
}
