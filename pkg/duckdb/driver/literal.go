package driver

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/tracing/meta"
)

// timestampLayout is the layout every time.Time is encoded with. Microsecond
// precision is load-bearing, not cosmetic: the spec's append-only,
// no-correlation-state design has readers reconstruct a run's sequencing "from
// timestamps at query time", so a step's scheduled/started/finished rows —
// which in a fast dev run all land inside the same second — must remain
// orderable. DuckDB's TIMESTAMP type is microsecond-precision and round-trips
// this layout exactly (verified against the real binary); it renders a
// zero-fraction timestamp back without the fractional part, which the
// read-side layout list in store.go's AsTimestamp already handles.
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
	case json.RawMessage:
		// Cast through ::JSON rather than encoded as an ordinary VARCHAR
		// literal (the plain string case above): this matters for any
		// VARIANT-typed column (migrations/000001_baseline.sql: attributes,
		// inputs, output, values, links, input, event_data, event_meta).
		// DuckDB's implicit VARCHAR->VARIANT assignment cast does NOT parse
		// a plain string's content — it wraps the whole string as a single
		// VARCHAR-typed variant leaf instead of the structured value
		// callers expect (confirmed against a real binary:
		// `'{"a":1}'::VARCHAR::VARIANT` reports variant_typeof "VARCHAR",
		// not "OBJECT(a)"; going through JSON first,
		// `'{"a":1}'::JSON::VARIANT`, does parse it). A caller with
		// already-marshaled JSON bytes for one of these columns should pass
		// them as json.RawMessage instead of string precisely to opt into
		// this cast — see conn.go's CheckNamedValue for why that requires
		// its own case there too, not just here.
		return "'" + strings.ReplaceAll(string(val), "'", "''") + "'::JSON", nil
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
	case []string:
		// DuckDB's array literal syntax (['a', 'b']) — no cast needed; the
		// target column's declared type (e.g. VARCHAR[]) is enough for
		// DuckDB to accept it, confirmed against a real subprocess over both
		// the stdio and quack transports.
		var sb strings.Builder
		sb.WriteString("[")
		for i, s := range val {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteByte('\'')
			sb.WriteString(strings.ReplaceAll(s, "'", "''"))
			sb.WriteByte('\'')
		}
		sb.WriteString("]")
		return sb.String(), nil
	case meta.EventSessions:
		// DuckDB's list-of-struct literal syntax (`[{'key': 'a', 'id':
		// 'b'}]`) — genuine SQL expression syntax (this is a real
		// interpolated INSERT statement, not a cast-from-string), so no
		// VARCHAR->LIST(STRUCT) casting trick is needed here, unlike
		// cmd/duckdbseed's Appender-based writer (see its sessionsLiteral
		// doc comment), which goes through a different write mechanism
		// that requires one.
		var sb strings.Builder
		sb.WriteString("[")
		for i, s := range val {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("{'key': '")
			sb.WriteString(strings.ReplaceAll(s.Key, "'", "''"))
			sb.WriteString("', 'id': '")
			sb.WriteString(strings.ReplaceAll(s.ID, "'", "''"))
			sb.WriteString("'}")
		}
		sb.WriteString("]")
		return sb.String(), nil
	default:
		return "", fmt.Errorf("duckdb: unsupported literal type %T", v)
	}
}

// interpolate replaces every "?" positional placeholder in query with its
// corresponding encoded literal, in ordinal order. It returns an error if the
// number of placeholders doesn't match len(args). A "?" inside a string
// literal, quoted identifier, comment, or dollar-quoted string is not a
// placeholder — see placeholderOffsets.
func interpolate(query string, args []driver.NamedValue) (string, error) {
	offsets, lexErr := placeholderOffsets(query)
	if len(args) == 0 {
		// With nothing to bind, a query this lexer can't make sense of is
		// passed through untouched so DuckDB reports the real syntax error.
		if lexErr == nil && len(offsets) != 0 {
			return "", fmt.Errorf("duckdb: query has %d placeholders but no args were bound", len(offsets))
		}
		return query, nil
	}
	if lexErr != nil {
		return "", lexErr
	}

	ordered := make([]driver.Value, len(args))
	for _, a := range args {
		if a.Ordinal < 1 || a.Ordinal > len(args) {
			return "", fmt.Errorf("duckdb: arg ordinal %d out of range for %d args", a.Ordinal, len(args))
		}
		ordered[a.Ordinal-1] = a.Value
	}
	if len(offsets) != len(ordered) {
		return "", fmt.Errorf("duckdb: query has %d placeholders but %d args were bound", len(offsets), len(ordered))
	}

	var sb strings.Builder
	prev := 0
	for i, off := range offsets {
		literal, err := encodeLiteral(ordered[i])
		if err != nil {
			return "", err
		}
		sb.WriteString(query[prev:off])
		sb.WriteString(literal)
		prev = off + 1
	}
	sb.WriteString(query[prev:])
	return sb.String(), nil
}

// placeholderOffsets returns the byte offset of every "?" in query that
// DuckDB's parser would see as a positional parameter, skipping the lexical
// contexts where "?" is just text:
//
//   - 'string' literals, with ” as an escaped quote, and E'...' escape
//     strings, where a backslash escapes the next byte
//   - "quoted identifiers", with "" as an escaped quote
//   - -- line comments and /* block comments */, which nest
//   - $$dollar-quoted$$ and $tag$dollar-quoted$tag$ strings
//
// An unterminated construct is an error: interpolating into SQL whose
// quoting we can't follow is exactly how a bound value ends up somewhere it
// shouldn't.
func placeholderOffsets(query string) ([]int, error) {
	var offsets []int
	n := len(query)
	for i := 0; i < n; i++ {
		c := query[i]
		switch {
		case c == '?':
			offsets = append(offsets, i)

		case c == '\'':
			escapes := i > 0 && (query[i-1] == 'E' || query[i-1] == 'e') && (i < 2 || !isIdentByte(query[i-2]))
			end, ok := skipQuoted(query, i, '\'', escapes)
			if !ok {
				return nil, fmt.Errorf("duckdb: unterminated string literal at offset %d", i)
			}
			i = end

		case c == '"':
			end, ok := skipQuoted(query, i, '"', false)
			if !ok {
				return nil, fmt.Errorf("duckdb: unterminated quoted identifier at offset %d", i)
			}
			i = end

		case c == '-' && i+1 < n && query[i+1] == '-':
			end := strings.IndexByte(query[i:], '\n')
			if end < 0 {
				return offsets, nil
			}
			i += end

		case c == '/' && i+1 < n && query[i+1] == '*':
			depth := 1
			j := i + 2
			for ; j < n && depth > 0; j++ {
				switch {
				case query[j] == '/' && j+1 < n && query[j+1] == '*':
					depth++
					j++
				case query[j] == '*' && j+1 < n && query[j+1] == '/':
					depth--
					j++
				}
			}
			if depth > 0 {
				return nil, fmt.Errorf("duckdb: unterminated block comment at offset %d", i)
			}
			i = j - 1

		case c == '$' && (i == 0 || !isIdentByte(query[i-1])):
			tag, ok := dollarTag(query[i:])
			if !ok {
				continue
			}
			end := strings.Index(query[i+len(tag):], tag)
			if end < 0 {
				return nil, fmt.Errorf("duckdb: unterminated dollar-quoted string at offset %d", i)
			}
			i += len(tag) + end + len(tag) - 1
		}
	}
	return offsets, nil
}

// skipQuoted returns the offset of the quote closing the quoted run that
// opens at query[start], treating a doubled quote as an escaped one and,
// when backslashEscapes is set, a backslash as escaping the next byte.
func skipQuoted(query string, start int, quote byte, backslashEscapes bool) (int, bool) {
	for j := start + 1; j < len(query); j++ {
		switch query[j] {
		case '\\':
			if backslashEscapes {
				j++
			}
		case quote:
			if j+1 < len(query) && query[j+1] == quote {
				j++
				continue
			}
			return j, true
		}
	}
	return 0, false
}

// dollarTag reports whether s opens a dollar-quoted string, returning its
// delimiter ("$$" or "$tag$"). A "$" followed by a digit is a $1-style
// parameter, not a tag, since a tag can't start with one.
func dollarTag(s string) (string, bool) {
	for j := 1; j < len(s); j++ {
		c := s[j]
		if c == '$' {
			return s[:j+1], true
		}
		if !isIdentByte(c) || (j == 1 && c >= '0' && c <= '9') {
			return "", false
		}
	}
	return "", false
}

func isIdentByte(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c >= 0x80
}
