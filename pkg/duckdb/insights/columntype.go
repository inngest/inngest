package insights

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// ColumnType is the GQL-facing type bucket for one result column — see
// docs/plans/010-duckdb-insights-query-layer.md's GQL schema section
// (InsightsColumnType). UUID isn't its own bucket: it prints/compares as a
// string, so it reports STRING. DuckDB's composite types (T[] lists,
// STRUCT/MAP/VARIANT) report JSON: they're nested data a client renders the
// same way as any other JSON value. VARIANT specifically is what every
// semi-structured column in this schema (attributes/output/input/
// event_data/etc., migrations/000001_baseline.sql) is actually declared
// as -- DuckDB never reports the JSON extension type name for them.
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
// DuckDB's driver reports it, `STRUCT(a INTEGER)[]`, `DECIMAL(18,3)`) to a
// ColumnType bucket -- see docs/plans/010's GQL schema section's mapping
// table. It parses the string with the same grammar rule a query's own
// casts use, so it classifies exactly like typeNameColumnType does for
// those, and a name the parser can't read is ColumnTypeUnknown.
func DuckDBToColumnType(duckdbType string) ColumnType {
	t, err := parser.ParseType(duckdbType)
	if err != nil {
		return ColumnTypeUnknown
	}
	return typeNameColumnType(t)
}

// typeNameColumnType buckets a parsed type by its base type name
// (TypeName.BaseTypeName, which resolves DuckDB's aliases like INT8, TEXT,
// CHARACTER VARYING and DOUBLE PRECISION). Any list/array dimension makes
// it JSON regardless of the element type.
func typeNameColumnType(t *parser.TypeName) ColumnType {
	if len(t.Arrays) > 0 {
		return ColumnTypeJSON
	}
	switch name := t.BaseTypeName(); {
	case name == "STRUCT", name == "MAP", name == "UNION", name == "LIST", name == "ARRAY",
		name == "JSON", name == "VARIANT":
		return ColumnTypeJSON
	case name == "VARCHAR", name == "UUID", name == "BLOB", name == "ENUM":
		return ColumnTypeString
	case name == "BOOLEAN":
		return ColumnTypeBoolean
	// TIME, TIMESTAMP, TIMESTAMP_MS, TIMESTAMP WITH TIME ZONE, TIME_NS, ...
	case name == "DATE", name == "INTERVAL", strings.HasPrefix(name, "TIME"):
		return ColumnTypeDatetime
	case isNumberType(name):
		return ColumnTypeNumber
	default:
		return ColumnTypeUnknown
	}
}

// String is the same label the GQL layer's InsightsColumnType enum uses
// (models.InsightsColumnType's string values), reused as-is by
// DESCRIBE's short-circuited output (describe.go) so a column's
// reported type reads identically there and in a real query result.
func (t ColumnType) String() string {
	switch t {
	case ColumnTypeString:
		return "STRING"
	case ColumnTypeNumber:
		return "NUMBER"
	case ColumnTypeBoolean:
		return "BOOLEAN"
	case ColumnTypeDatetime:
		return "DATETIME"
	case ColumnTypeJSON:
		return "JSON"
	default:
		return "UNKNOWN"
	}
}

func isNumberType(t string) bool {
	switch t {
	case "TINYINT", "SMALLINT", "INTEGER", "BIGINT", "HUGEINT",
		"UTINYINT", "USMALLINT", "UINTEGER", "UBIGINT", "UHUGEINT",
		"FLOAT", "DOUBLE", "DECIMAL", "BIGNUM":
		return true
	default:
		return false
	}
}
