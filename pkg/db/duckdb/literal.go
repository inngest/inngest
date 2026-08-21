package duckdb

import (
	"database/sql/driver"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// timestampLayout is the layout every time.Time is encoded with. Microsecond
// precision is load-bearing, not cosmetic: the spec's append-only,
// no-correlation-state design has readers reconstruct a run's sequencing "from
// timestamps at query time", so a step's scheduled/started/finished rows —
// which in a fast dev run all land inside the same second — must remain
// orderable. DuckDB's TIMESTAMP type is microsecond-precision and round-trips
// this layout exactly (verified against the real binary); it renders a
// zero-fraction timestamp back without the fractional part, which the
// read-side layout list in store.go's asTimestamp already handles.
const timestampLayout = "2006-01-02 15:04:05.000000"

// encodeLiteral converts a bound driver.Value into DuckDB SQL literal text.
// This exists because the stdio/JSON-lines transport has no wire-level
// parameter binding — every bound value must be safely quoted/escaped as
// literal SQL text before being sent to the subprocess. All inputs in this
// phase come from internal batch data, not external user input, but this
// code must still be treated as injection-sensitive.
func encodeLiteral(v driver.Value) (string, error) {
	switch val := v.(type) {
	case nil:
		return "NULL", nil
	case bool:
		if val {
			return "TRUE", nil
		}
		return "FALSE", nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case float64:
		// NaN/±Inf have no bare numeric spelling DuckDB's parser accepts, so
		// they must go over the wire as casts of their special string forms
		// ('NaN'::DOUBLE etc., verified against the real binary). Emitting the
		// Go spellings ("NaN", "+Inf") unquoted would produce invalid SQL —
		// which used to fail silently, and now (see session.exec's stderr
		// correlation) would fail the whole batch loudly instead.
		switch {
		case math.IsNaN(val):
			return "'NaN'::DOUBLE", nil
		case math.IsInf(val, 1):
			return "'Infinity'::DOUBLE", nil
		case math.IsInf(val, -1):
			return "'-Infinity'::DOUBLE", nil
		}
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'", nil
	case []byte:
		var sb strings.Builder
		sb.WriteString("'")
		for _, b := range val {
			fmt.Fprintf(&sb, "\\x%02x", b)
		}
		sb.WriteString("'::BLOB")
		return sb.String(), nil
	case time.Time:
		return "TIMESTAMP '" + val.UTC().Format(timestampLayout) + "'", nil
	default:
		return "", fmt.Errorf("duckdb: unsupported literal type %T", v)
	}
}

// interpolate replaces every "?" positional placeholder in query with its
// corresponding encoded literal, in ordinal order. It returns an error if the
// number of placeholders doesn't match len(args).
func interpolate(query string, args []driver.NamedValue) (string, error) {
	if len(args) == 0 {
		if strings.Count(query, "?") != 0 {
			return "", fmt.Errorf("duckdb: query has placeholders but no args were bound: %q", query)
		}
		return query, nil
	}

	ordered := make([]driver.Value, len(args))
	for _, a := range args {
		if a.Ordinal < 1 || a.Ordinal > len(args) {
			return "", fmt.Errorf("duckdb: arg ordinal %d out of range for %d args", a.Ordinal, len(args))
		}
		ordered[a.Ordinal-1] = a.Value
	}

	var sb strings.Builder
	argIdx := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			if argIdx >= len(ordered) {
				return "", fmt.Errorf("duckdb: query has more placeholders than the %d bound args", len(args))
			}
			literal, err := encodeLiteral(ordered[argIdx])
			if err != nil {
				return "", err
			}
			sb.WriteString(literal)
			argIdx++
			continue
		}
		sb.WriteByte(query[i])
	}
	if argIdx != len(ordered) {
		return "", fmt.Errorf("duckdb: query has %d placeholders but %d args were bound", argIdx, len(ordered))
	}
	return sb.String(), nil
}
