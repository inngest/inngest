package driver

import (
	"database/sql"
	"fmt"
	"time"
)

// ScanRowByName scans the current row of rows generically and returns its
// values keyed by column name, for callers that want name-keyed access to a
// query result (e.g. one whose column list isn't fixed at the call site)
// instead of writing out positional destinations.
func ScanRowByName(rows *sql.Rows) (map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	dest := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("duckdb: scan row: %w", err)
	}
	out := make(map[string]any, len(cols))
	for i, c := range cols {
		out[c] = dest[i]
	}
	return out, nil
}

// AsInt64 converts a value read back from this driver into an int64.
// DuckDB's -jsonlines transport surfaces JSON numbers as float64.
func AsInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case float64:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("duckdb: unexpected numeric type %T", v)
	}
}

// AsTimestamp converts a value read back from this driver into a time.Time.
// This driver has no read-side TIMESTAMP-to-time.Time conversion (see
// literal.go, which only handles the write side), so DuckDB's -jsonlines
// TIMESTAMP output arrives here as a plain string.
func AsTimestamp(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case string:
		for _, layout := range []string{"2006-01-02 15:04:05.999999", "2006-01-02 15:04:05", time.RFC3339} {
			if ts, err := time.Parse(layout, t); err == nil {
				return ts, nil
			}
		}
		return time.Time{}, fmt.Errorf("duckdb: unparseable timestamp %q", t)
	default:
		return time.Time{}, fmt.Errorf("duckdb: unexpected timestamp type %T", v)
	}
}
