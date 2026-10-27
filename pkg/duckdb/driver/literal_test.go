package driver

import (
	"database/sql/driver"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/duckdbtest"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
)

func TestEncodeLiteral(t *testing.T) {
	cases := []struct {
		name  string
		value driver.Value
		want  string
	}{
		{"nil", nil, "NULL"},
		{"bool true", true, "TRUE"},
		{"bool false", false, "FALSE"},
		{"int64", int64(42), "42"},
		{"float64", float64(3.5), "3.5"},
		{"plain string", "hello", "'hello'"},
		{"string with single quote", "O'Brien", "'O''Brien'"},
		{"string with backslash", `a\b`, `'a\b'`},
		{"bytes", []byte{0x01, 0x02}, "'\\x01\\x02'::BLOB"},
		{"float64 NaN", math.NaN(), "'NaN'::DOUBLE"},
		{"float64 +Inf", math.Inf(1), "'Infinity'::DOUBLE"},
		{"float64 -Inf", math.Inf(-1), "'-Infinity'::DOUBLE"},
		{
			"time.Time",
			time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			"TIMESTAMPTZ '2026-01-02 03:04:05.000+00'",
		},
		{
			"time.Time with sub-ms precision is truncated",
			time.Date(2026, 1, 2, 3, 4, 5, 123999999, time.UTC),
			"TIMESTAMPTZ '2026-01-02 03:04:05.123+00'",
		},
		{
			"time.Time in another zone",
			time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("x", -5*3600)),
			"TIMESTAMPTZ '2026-01-02 08:04:05.000+00'",
		},
		{"string slice", []string{"a", "b"}, "['a', 'b']"},
		{"string slice with single quote", []string{"O'Brien"}, "['O''Brien']"},
		{"empty string slice", []string{}, "[]"},
		{
			"event sessions",
			meta.EventSessions{{Key: "customer", ID: "acme"}, {Key: "cart", ID: "123"}},
			"[{'key': 'customer', 'id': 'acme'}, {'key': 'cart', 'id': '123'}]",
		},
		{
			"event sessions with single quote",
			meta.EventSessions{{Key: "O'Brien's cart", ID: "it's-here"}},
			"[{'key': 'O''Brien''s cart', 'id': 'it''s-here'}]",
		},
		{"nil event sessions", meta.EventSessions(nil), "[]"},
		{"empty event sessions", meta.EventSessions{}, "[]"},
		{"JSON raw message", json.RawMessage(`{"a":1}`), `'{"a":1}'::JSON`},
		{"JSON raw message with single quote", json.RawMessage(`{"a":"O'Brien"}`), `'{"a":"O''Brien"}'::JSON`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := encodeLiteral(tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestInterpolateReplacesPositionalPlaceholders(t *testing.T) {
	got, err := interpolate("INSERT INTO t VALUES (?, ?);", []driver.NamedValue{
		{Ordinal: 1, Value: int64(1)},
		{Ordinal: 2, Value: "a"},
	})
	require.NoError(t, err)
	require.Equal(t, "INSERT INTO t VALUES (1, 'a');", got)
}

func TestInterpolateRejectsMismatchedPlaceholderCount(t *testing.T) {
	_, err := interpolate("INSERT INTO t VALUES (?, ?);", []driver.NamedValue{
		{Ordinal: 1, Value: int64(1)},
	})
	require.Error(t, err)
}

// TestTimestampTZBinding is the real-binary proof behind timestampLayout: the
// session is pinned to UTC, a bound time.Time is TIMESTAMPTZ (the columns'
// own type, so filtering one doesn't cast it), and it round trips truncated
// to whole ms.
func TestTimestampTZBinding(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var tz string
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT current_setting('TimeZone');").Scan(&tz))
	require.Equal(t, "UTC", tz)

	at := time.Date(2026, 1, 2, 3, 4, 5, 123000000, time.UTC)

	// The driver can't run EXPLAIN (it DESCRIBEs every query), so this
	// checks the bound type instead: same type as the column -> no cast.
	var typ string
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT typeof(?);", at).Scan(&typ))
	require.Equal(t, "TIMESTAMP WITH TIME ZONE", typ)

	_, err = db.ExecContext(t.Context(), "CREATE TABLE ts (ts_at TIMESTAMPTZ);")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO ts VALUES (?);", at.Add(999*time.Microsecond))
	require.NoError(t, err)

	var got time.Time
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT ts_at FROM ts;").Scan(&got))
	require.True(t, at.Equal(got), "want %s, got %s", at, got)

	var n int
	require.NoError(t, db.QueryRowContext(t.Context(),
		"SELECT count(*) FROM ts WHERE ts_at = ? AND ts_at >= ? - INTERVAL 1 HOUR AND time_bucket(INTERVAL 6 HOUR, ts_at) = ? AND ts_at::DATE = DATE '2026-01-02';",
		at, at, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	).Scan(&n))
	require.Equal(t, 1, n)
}

// TestJSONRawMessageRoundTripsIntoVariantColumn is the real-binary proof
// behind encodeLiteral's json.RawMessage case: a VARIANT column populated
// from a plain Go string (the trap — what every dual-write column looked
// like before this existed) stores the whole string as a single
// VARCHAR-typed variant leaf rather than parsing it, so it decodes right
// back out as a Go string, not the object a caller like duckdbquery's
// asMap expects. Binding the same content as json.RawMessage instead casts
// it through ::JSON (see encodeLiteral's doc comment) and fixes that: it
// decodes to a real map.
func TestJSONRawMessageRoundTripsIntoVariantColumn(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	duckdbtest.RequireQuackExtension(t, binPath)

	quackAddr := EphemeralQuackAddr
	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:", QuackAddr: &quackAddr})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE v (plain VARIANT, viajson VARIANT);")
	require.NoError(t, err)

	jsonText := `{"conversationId":"336khdd91hd","totalTurns":5}`
	_, err = db.ExecContext(t.Context(), "INSERT INTO v (plain, viajson) VALUES (?, ?);", jsonText, json.RawMessage(jsonText))
	require.NoError(t, err)

	var plain, viaJSON any
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT plain, viajson FROM v;").Scan(&plain, &viaJSON))

	require.Equal(t, jsonText, plain, "an ordinary string bound to a VARIANT column is stored verbatim as a VARCHAR-typed variant leaf, not parsed")
	require.Equal(t, map[string]any{
		"conversationId": "336khdd91hd",
		"totalTurns":     int64(5),
	}, viaJSON, "json.RawMessage should cast through ::JSON so the VARIANT column stores (and decodes back to) a real object")
}

// TestSpecialFloatRoundTrip proves NaN/±Inf actually make it into DuckDB as
// the values they claim to be, rather than as invalid SQL tokens.
func TestSpecialFloatRoundTrip(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE f (v DOUBLE);")
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		"INSERT INTO f (v) VALUES (?), (?), (?), (?);",
		math.NaN(), math.Inf(1), math.Inf(-1), 1.5,
	)
	require.NoError(t, err)

	var (
		nans, infs, neginfs, finite int
	)
	require.NoError(t, db.QueryRowContext(t.Context(),
		"SELECT count(*) FROM f WHERE isnan(v);").Scan(&nans))
	require.NoError(t, db.QueryRowContext(t.Context(),
		"SELECT count(*) FROM f WHERE v = 'Infinity'::DOUBLE;").Scan(&infs))
	require.NoError(t, db.QueryRowContext(t.Context(),
		"SELECT count(*) FROM f WHERE v = '-Infinity'::DOUBLE;").Scan(&neginfs))
	require.NoError(t, db.QueryRowContext(t.Context(),
		"SELECT count(*) FROM f WHERE v = 1.5;").Scan(&finite))

	require.Equal(t, 1, nans)
	require.Equal(t, 1, infs)
	require.Equal(t, 1, neginfs)
	require.Equal(t, 1, finite)
}

// TestInterpolateIgnoresQuestionMarksOutsidePlaceholderContexts covers every
// lexical context in which "?" is text rather than a parameter. Each case
// binds exactly one arg, so a scanner that mistook any quoted "?" for a
// placeholder would fail on the count mismatch or splice the literal into
// the wrong place.
func TestInterpolateIgnoresQuestionMarksOutsidePlaceholderContexts(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  string
	}{
		{"string literal", `SELECT '?', ?`, `SELECT '?', 'x'`},
		{"doubled quote escape", `SELECT 'it''s ?', ?`, `SELECT 'it''s ?', 'x'`},
		{"escape string backslash quote", `SELECT E'\' ?', ?`, `SELECT E'\' ?', 'x'`},
		{"lowercase escape string", `SELECT e'\\', ?`, `SELECT e'\\', 'x'`},
		{"backslash is literal in plain string", `SELECT 'a\', ?`, `SELECT 'a\', 'x'`},
		{"identifier ending in e is not an escape string", `SELECT name'?' , ?`, `SELECT name'?' , 'x'`},
		{"quoted identifier", `SELECT "col?" FROM t WHERE a = ?`, `SELECT "col?" FROM t WHERE a = 'x'`},
		{"doubled quote in identifier", `SELECT "a""?" , ?`, `SELECT "a""?" , 'x'`},
		{"line comment", "SELECT ? -- trailing ?\n", "SELECT 'x' -- trailing ?\n"},
		{"line comment at eof", "SELECT ? -- ?", "SELECT 'x' -- ?"},
		{"block comment", `SELECT /* ? */ ?`, `SELECT /* ? */ 'x'`},
		{"nested block comment", `SELECT /* a /* ? */ ? */ ?`, `SELECT /* a /* ? */ ? */ 'x'`},
		{"dollar quoted", `SELECT $$ ? $$, ?`, `SELECT $$ ? $$, 'x'`},
		{"tagged dollar quoted", `SELECT $q$ ? $$ ? $q$, ?`, `SELECT $q$ ? $$ ? $q$, 'x'`},
		{"positional dollar param is not a tag", `SELECT $1, ?`, `SELECT $1, 'x'`},
		{"quote inside comment", "SELECT ? -- don't\n", "SELECT 'x' -- don't\n"},
		{"comment marker inside string", `SELECT '--', '/*', ?`, `SELECT '--', '/*', 'x'`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := interpolate(tc.query, []driver.NamedValue{{Ordinal: 1, Value: "x"}})
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestInterpolateRejectsUnterminatedConstructs pins that a bound value is
// never spliced into SQL whose quoting the scanner could not follow.
func TestInterpolateRejectsUnterminatedConstructs(t *testing.T) {
	for _, q := range []string{
		`SELECT ?, 'open`,
		`SELECT ?, "open`,
		`SELECT ?, /* open`,
		`SELECT ?, /* a /* b */`,
		`SELECT ?, $$ open`,
		`SELECT ?, E'\'`,
	} {
		t.Run(q, func(t *testing.T) {
			_, err := interpolate(q, []driver.NamedValue{{Ordinal: 1, Value: "x"}})
			require.Error(t, err)
		})
	}
}

// TestInterpolateWithoutArgs covers the no-args path: quoted "?" is fine, a
// real placeholder with nothing bound is an error, and SQL the scanner can't
// lex passes through so DuckDB reports its own syntax error.
func TestInterpolateWithoutArgs(t *testing.T) {
	got, err := interpolate(`SELECT '?' AS q -- ?`, nil)
	require.NoError(t, err)
	require.Equal(t, `SELECT '?' AS q -- ?`, got)

	_, err = interpolate(`SELECT ?`, nil)
	require.Error(t, err)

	got, err = interpolate(`SELECT 'unterminated ?`, nil)
	require.NoError(t, err)
	require.Equal(t, `SELECT 'unterminated ?`, got)
}

// TestInterpolateBoundValueContainingQuestionMark guards against a
// regression where placeholders are found by rescanning the output: a bound
// string containing "?" must not consume the next arg.
func TestInterpolateBoundValueContainingQuestionMark(t *testing.T) {
	got, err := interpolate(`SELECT ?, ?`, []driver.NamedValue{
		{Ordinal: 1, Value: "a?b"},
		{Ordinal: 2, Value: "c"},
	})
	require.NoError(t, err)
	require.Equal(t, `SELECT 'a?b', 'c'`, got)
}
