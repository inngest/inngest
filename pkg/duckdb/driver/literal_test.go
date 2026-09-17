package driver

import (
	"database/sql/driver"
	"encoding/json"
	"math"
	"testing"
	"time"

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
			"TIMESTAMP '2026-01-02 03:04:05.000000'",
		},
		{
			"time.Time with sub-second precision",
			time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC),
			"TIMESTAMP '2026-01-02 03:04:05.123456'",
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

// TestTimestampRoundTripPreservesMicroseconds is the real-binary proof behind
// timestampLayout: three timestamps inside the same second must come back
// distinct and in order, since the dual-write schema's append-only rows carry
// no sequence number and readers order them by timestamp alone.
func TestTimestampRoundTripPreservesMicroseconds(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE ts (label VARCHAR, ts_at TIMESTAMP);")
	require.NoError(t, err)

	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for i, label := range []string{"scheduled", "started", "finished"} {
		_, err = db.ExecContext(t.Context(),
			"INSERT INTO ts (label, ts_at) VALUES (?, ?);",
			label, base.Add(time.Duration(i+1)*time.Microsecond),
		)
		require.NoError(t, err)
	}

	rows, err := db.QueryContext(t.Context(), "SELECT label FROM ts ORDER BY ts_at ASC;")
	require.NoError(t, err)
	defer rows.Close()

	var ordered []string
	for rows.Next() {
		var label string
		require.NoError(t, rows.Scan(&label))
		ordered = append(ordered, label)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"scheduled", "started", "finished"}, ordered)

	// And the distinct-count proves the microseconds weren't truncated away.
	var distinct int
	require.NoError(t,
		db.QueryRowContext(t.Context(), "SELECT count(DISTINCT ts_at) FROM ts;").Scan(&distinct))
	require.Equal(t, 3, distinct)
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
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
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
