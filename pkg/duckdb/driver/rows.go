package driver

import (
	"database/sql/driver"
	"io"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
)

// mapRows adapts a []result.Row into database/sql/driver.Rows. types is
// nil for a result conn.go's ExecContext path produced (no caller ever asks
// ExecContext's driver.Result for column types); QueryContext always
// supplies it (see conn.go), which is what backs *sql.Rows.ColumnTypes()'s
// DatabaseTypeName() — see ColumnTypeDatabaseTypeName below.
type mapRows struct {
	cols  []string
	types []string
	rows  []result.Row
	pos   int
}

func newMapRows(cols []string, types []string, rows []result.Row) *mapRows {
	return &mapRows{cols: cols, types: types, rows: rows}
}

func (r *mapRows) Columns() []string { return r.cols }
func (r *mapRows) Close() error      { return nil }

// ColumnTypeDatabaseTypeName implements
// database/sql/driver.RowsColumnTypeDatabaseTypeName, which is what
// *sql.Rows.ColumnTypes()[i].DatabaseTypeName() calls through to. This is
// the mechanism that lets pkg/duckdb/insights.Execute learn a query's real
// DuckDB column types from the query it already ran, instead of a second
// "DESCRIBE <sql>" query of its own.
func (r *mapRows) ColumnTypeDatabaseTypeName(index int) string {
	if index < 0 || index >= len(r.types) {
		return ""
	}
	return r.types[index]
}

func (r *mapRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.rows) {
		return io.EOF
	}
	cur := r.rows[r.pos]
	for i := range r.cols {
		var val any
		if i < len(cur.Vals) {
			val = cur.Vals[i]
		}
		if i < len(r.types) {
			val = convertTimeValue(r.types[i], val)
		}
		dest[i] = val
	}
	r.pos++
	return nil
}

// timeColumnLayouts maps DuckDB's date/time/timestamp column type names (as
// DESCRIBE reports them, and therefore as query()'s types slice carries
// them) to the Go time layout jsonlines' text serialization of that type can
// be parsed with. DuckDB always writes a variable-precision (0-9 digit)
// fractional-seconds component — present when non-zero, omitted entirely
// when zero (confirmed empirically: "2026-01-01 00:00:00" vs. "...23:56:01.933")
// — so every layout uses ".999999999" (Go's trim-trailing-zeros/
// optional-digit-count fractional-second placeholder) rather than a
// fixed-width ".000". "TIMESTAMP WITH TIME ZONE" isn't in this map because
// its offset format varies in a way no single Go layout token matches (see
// parseDuckDBValue).
var timeColumnLayouts = map[string]string{
	"DATE":         "2006-01-02",
	"TIME":         "15:04:05.999999999",
	"TIMESTAMP":    "2006-01-02 15:04:05.999999999",
	"TIMESTAMP_S":  "2006-01-02 15:04:05.999999999",
	"TIMESTAMP_MS": "2006-01-02 15:04:05.999999999",
	"TIMESTAMP_NS": "2006-01-02 15:04:05.999999999",
}

// timestampTZColumnType is DESCRIBE's name for TIMESTAMPTZ/TIMESTAMP WITH
// TIME ZONE columns, handled separately from timeColumnLayouts — see
// parseDuckDBTimestampTZ.
const timestampTZColumnType = "TIMESTAMP WITH TIME ZONE"

// convertTimeValue converts a jsonlines-decoded value into a time.Time when
// dbType names one of DuckDB's date/time/timestamp types, so a
// database/sql caller can Scan directly into *time.Time (or sql.NullTime)
// instead of unexpectedly getting a raw string back — encoding/json's
// generic decode (decodeOrderedRow) can't tell a TIMESTAMP-typed string
// apart from an ordinary VARCHAR by its JSON shape alone, only dbType (the
// column's real DuckDB type, from query()'s DESCRIBE) can. Values dbType
// doesn't recognize, values already a non-string driver.Value, and nil (SQL
// NULL) all pass through unchanged; a value that fails to parse against its
// own type's layout also passes through unchanged rather than erroring —
// this is a best-effort convenience, not a correctness-load-bearing parser,
// and a Scan destination expecting a string still works with the raw text.
func convertTimeValue(dbType string, val any) any {
	s, ok := val.(string)
	if !ok {
		return val
	}
	if dbType == timestampTZColumnType {
		if t, err := parseDuckDBTimestampTZ(s); err == nil {
			return t
		}
		return val
	}
	layout, ok := timeColumnLayouts[dbType]
	if !ok {
		return val
	}
	if t, err := time.Parse(layout, s); err == nil {
		return t
	}
	return val
}

// parseDuckDBTimestampTZ parses a TIMESTAMP WITH TIME ZONE value's
// jsonlines text. DuckDB omits the UTC offset's minutes component when
// zero ("...23:56:07.43901-04") but includes it otherwise
// ("...09:26:48.980587+05:30") — confirmed empirically switching session
// TimeZone — so no single Go layout token matches both; try the
// minutes-included form first, then the minutes-omitted one.
func parseDuckDBTimestampTZ(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05.999999999-07", s)
}
