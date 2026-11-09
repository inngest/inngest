package insights_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestExecuteJSONColumns checks the lake's JSON text columns still come
// back from Insights as JSON: typed JSON, decoded to their values, and
// readable w/ ->>, dot access and json_* (in product and raw mode).
func TestExecuteJSONColumns(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	accountID, envID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil).String()
	insertStepSpan(t, db, accountID, envID, runID, "span-1", "step-1", 2, time.Now())
	_, err := db.ExecContext(ctx,
		`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts, event_meta)
		 VALUES (?, ?, 'internal-1', now(), 'api', 'evt-1', 'app/x', ?, '1', now(), '{}');`,
		accountID.String(), envID.String(), json.RawMessage(`{"user":{"id":"u1","n":12345678901},"tags":["a","b"]}`),
	)
	require.NoError(t, err)

	run := func(t *testing.T, tr *insights.TranspileResult) *insights.Result {
		t.Helper()
		res, err := insights.Execute(ctx, db, tr)
		require.NoError(t, err)
		require.Len(t, res.Rows, 1)
		return res
	}

	t.Run("columns", func(t *testing.T) {
		tr, err := insights.Transpile("SELECT data, attributes FROM events, steps", accountID, envID)
		require.NoError(t, err)
		res := run(t, tr)
		require.Equal(t, insights.ColumnTypeJSON, res.Columns[0].Type)
		require.Equal(t, insights.ColumnTypeJSON, res.Columns[1].Type)
		require.Equal(t, map[string]any{
			"user": map[string]any{"id": "u1", "n": float64(12345678901)},
			"tags": []any{"a", "b"},
		}, res.Rows[0][0])
		require.Equal(t, map[string]any{"_inngest.step.userland.id": "step-1", "_inngest.step.attempt": float64(2)}, res.Rows[0][1])
	})

	t.Run("access", func(t *testing.T) {
		tr, err := insights.Transpile(`SELECT data->>'$.user.id', data.user.id, json_extract_string(data, '$.tags[1]'), data->'user', attributes->>'$."_inngest.step.userland.id"'
			FROM events, steps WHERE data->>'$.user.id' = 'u1' AND step_id = 'step-1' AND step_attempt = 2`, accountID, envID)
		require.NoError(t, err)
		res := run(t, tr)
		require.Equal(t, []any{"u1", "u1", "b", map[string]any{"id": "u1", "n": float64(12345678901)}, "step-1"}, res.Rows[0])
		// Dot access is JSON, like DuckDB's own JSON dot.
		require.Equal(t, insights.ColumnTypeJSON, res.Columns[1].Type)
	})

	t.Run("raw", func(t *testing.T) {
		tr, err := insights.TranspileRaw("SELECT event_data, event_data->>'$.user.id' FROM events WHERE event_id = 'evt-1'")
		require.NoError(t, err)
		res := run(t, tr)
		require.Equal(t, insights.ColumnTypeJSON, res.Columns[0].Type)
		require.Equal(t, "u1", res.Rows[0][1])
	})
}

// TestExecuteJSONDotComparison checks dot access into a JSON column
// compares like the value it holds against a lake w/ JSON text columns:
// as text against a string (rewriteJSONDotComparison), natively against
// a number or boolean, and stays JSON when projected.
func TestExecuteJSONDotComparison(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	accountID, envID := uuid.New(), uuid.New()
	for i, data := range []string{
		`{"user":{"id":"u1","name":"Ann"},"n":5,"ok":true,"z":null,"a.b":"dotted","tags":["a",1]}`,
		`{"user":{"id":"u2"},"n":10,"ok":false}`,
		`{"n":"7"}`,
	} {
		_, err := db.ExecContext(ctx,
			`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts, event_meta)
			 VALUES (?, ?, ?, now(), 'api', ?, 'app/x', ?, '1', now(), '{}');`,
			accountID.String(), envID.String(), fmt.Sprintf("internal-%d", i), fmt.Sprintf("evt-%d", i), json.RawMessage(data),
		)
		require.NoError(t, err)
	}

	ids := func(t *testing.T, where string) []any {
		t.Helper()
		tr, err := insights.Transpile("SELECT id FROM events WHERE "+where+" ORDER BY id", accountID, envID)
		require.NoError(t, err)
		res, err := insights.Execute(ctx, db, tr)
		require.NoError(t, err)
		out := []any{}
		for _, row := range res.Rows {
			out = append(out, row[0])
		}
		return out
	}

	for _, tc := range []struct {
		where string
		want  []any
	}{
		{`data.user.id = 'u1'`, []any{"evt-0"}},
		{`(data.user).id = 'u1'`, []any{"evt-0"}},
		{`events.data.user.id <> 'u1'`, []any{"evt-1"}},
		{`data.user.id IN ('u1', 'u2')`, []any{"evt-0", "evt-1"}},
		{`data.user.id LIKE 'u%'`, []any{"evt-0", "evt-1"}},
		{`data.user.id IS NULL`, []any{"evt-2"}},
		{`data.z IS NULL`, []any{"evt-0", "evt-1", "evt-2"}},
		{`data."a.b" = 'dotted'`, []any{"evt-0"}},
		{`data.n = '7'`, []any{"evt-2"}},
		{`data.n = 5`, []any{"evt-0"}},
		{`data.n > 6`, []any{"evt-1", "evt-2"}},
		{`data.n BETWEEN 4 AND 8`, []any{"evt-0", "evt-2"}},
		{`data.ok = true`, []any{"evt-0"}},
		{`data.user.id = (SELECT 'u2' AS u)`, []any{"evt-1"}},
		{`data.user.id::VARCHAR = 'u1'`, []any{"evt-0"}},
		{`starts_with(data.user.id, 'u')`, []any{"evt-0", "evt-1"}},
		{`regexp_matches(data.user.id, '^u2$')`, []any{"evt-1"}},
		{`data.user.id || '!' = 'u1!'`, []any{"evt-0"}},
		{`lower(data.user.name) = 'ann'`, []any{"evt-0"}},
		{`length(data.user.id) = 2`, []any{"evt-0", "evt-1"}},
	} {
		t.Run(tc.where, func(t *testing.T) {
			require.Equal(t, tc.want, ids(t, tc.where))
		})
	}

	t.Run("derived JSON column", func(t *testing.T) {
		tr, err := insights.Transpile("WITH e AS (SELECT id, json_extract(data, '$.user') AS u FROM events) SELECT id FROM e WHERE u.id = 'u2'", accountID, envID)
		require.NoError(t, err)
		res, err := insights.Execute(ctx, db, tr)
		require.NoError(t, err)
		require.Equal(t, [][]any{{"evt-1"}}, res.Rows)
	})

	t.Run("cast to text", func(t *testing.T) {
		tr, err := insights.Transpile("SELECT data.user.id::VARCHAR AS uid, CAST(data.n AS TEXT) AS n, data.user::VARCHAR AS u, data.tags::VARCHAR AS tags, data.z::VARCHAR AS z, TRY_CAST(data.missing AS VARCHAR) AS missing FROM events WHERE id = 'evt-0'", accountID, envID)
		require.NoError(t, err)
		res, err := insights.Execute(ctx, db, tr)
		require.NoError(t, err)
		for _, c := range res.Columns {
			require.Equal(t, insights.ColumnTypeString, c.Type, c.Name)
		}
		// A string unquoted, anything else its JSON text, a JSON null
		// SQL NULL (a plain cast gives the text "null").
		require.Equal(t, [][]any{{"u1", "5", `{"id":"u1","name":"Ann"}`, `["a",1]`, nil, nil}}, res.Rows)
	})

	t.Run("projection stays JSON, decoded", func(t *testing.T) {
		// Execute decodes a JSON column's values, so a projected dot access
		// already reaches the caller as the native value (a string unquoted),
		// w/ the name DuckDB gives it (its last key).
		tr, err := insights.Transpile("SELECT data.user AS u, data.user.id, data.n, data.ok, data.z, data.missing, data.tags FROM events WHERE data.user.id = 'u1'", accountID, envID)
		require.NoError(t, err)
		res, err := insights.Execute(ctx, db, tr)
		require.NoError(t, err)
		var names []string
		for _, c := range res.Columns {
			names = append(names, c.Name)
			require.Equal(t, insights.ColumnTypeJSON, c.Type, c.Name)
		}
		require.Equal(t, []string{"u", "id", "n", "ok", "z", "missing", "tags"}, names)
		require.Equal(t, [][]any{{map[string]any{"id": "u1", "name": "Ann"}, "u1", float64(5), true, nil, nil, []any{"a", float64(1)}}}, res.Rows)
	})
}

// TestExecuteJSONDotNumeric checks a JSON dot access compared against a
// number reads as a DOUBLE: no rounding to an integer literal's type, no
// failing on a value wider than a decimal literal, and a non-numeric value
// just not matching.
func TestExecuteJSONDotNumeric(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	accountID, envID := uuid.New(), uuid.New()
	for i, n := range []string{`6.4`, `6.6`, `5.5`, `1e2`, `"7"`, `"x"`, `6`} {
		_, err := db.ExecContext(ctx,
			`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts, event_meta)
			 VALUES (?, ?, ?, now(), 'api', ?, 'app/x', ?, '1', now(), '{}');`,
			accountID.String(), envID.String(), fmt.Sprintf("internal-%d", i), fmt.Sprintf("evt-%d", i), json.RawMessage(`{"n":`+n+`}`),
		)
		require.NoError(t, err)
	}

	query := func(t *testing.T, sql string) [][]any {
		t.Helper()
		tr, err := insights.Transpile(sql, accountID, envID)
		require.NoError(t, err)
		res, err := insights.Execute(ctx, db, tr)
		require.NoError(t, err)
		return res.Rows
	}
	ids := func(t *testing.T, where string) []any {
		t.Helper()
		out := []any{}
		for _, row := range query(t, "SELECT id FROM events WHERE "+where+" ORDER BY id") {
			out = append(out, row[0])
		}
		return out
	}

	// evt-0 6.4, evt-1 6.6, evt-2 5.5, evt-3 1e2, evt-4 "7", evt-5 "x", evt-6 6
	for _, tc := range []struct {
		where string
		want  []any
	}{
		{`data.n > 6`, []any{"evt-0", "evt-1", "evt-3", "evt-4"}},
		{`data.n >= 7`, []any{"evt-3", "evt-4"}},
		{`data.n = 6`, []any{"evt-6"}},
		{`data.n > 6.5`, []any{"evt-1", "evt-3", "evt-4"}},
		{`data.n BETWEEN 6 AND 7`, []any{"evt-0", "evt-1", "evt-4", "evt-6"}},
		{`data.n IN (6, 7)`, []any{"evt-4", "evt-6"}},
		{`data.n NOT IN (6, 7)`, []any{"evt-0", "evt-1", "evt-2", "evt-3"}},
		{`data.n < (SELECT 6 AS six)`, []any{"evt-2"}},
		{`data.n = 'x'`, []any{"evt-5"}},
	} {
		t.Run(tc.where, func(t *testing.T) {
			require.Equal(t, tc.want, ids(t, tc.where))
		})
	}

	t.Run("numeric column peer", func(t *testing.T) {
		rows := query(t, "WITH s AS (SELECT 6 AS six) SELECT id FROM events, s WHERE data.n <= s.six ORDER BY id")
		require.Equal(t, [][]any{{"evt-2"}, {"evt-6"}}, rows)
	})

	t.Run("sum and arithmetic", func(t *testing.T) {
		rows := query(t, "SELECT sum(data.n) AS s, max(data.n + 1) AS m FROM events")
		require.InDelta(t, 6.4+6.6+5.5+100+7+6, rows[0][0], 1e-9)
		require.InDelta(t, 101.0, rows[0][1], 1e-9)
	})
}
