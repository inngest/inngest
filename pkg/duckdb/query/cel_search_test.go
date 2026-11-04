package query

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

// useInsightsCEL swaps the real pkg/duckdb/insights CEL-to-SQL conversion
// into this package's plugin points for the duration of t.
func useInsightsCEL(t *testing.T) {
	t.Helper()
	prevTable, prevEvent, prevOutput, prevRender := CELEventTableFilters, CELEventFilters, CELOutputFilters, RenderWhereSQL
	CELEventTableFilters = insights.CELEventTableFilters
	CELEventFilters = insights.CELEventFilters
	CELOutputFilters = insights.CELOutputFilters
	RenderWhereSQL = insights.RenderWhereSQL
	t.Cleanup(func() {
		CELEventTableFilters, CELEventFilters, CELOutputFilters, RenderWhereSQL = prevTable, prevEvent, prevOutput, prevRender
	})
}

type celTestRun struct {
	inputs string
	output *string
}

// seedCELRun inserts one completed run with the given inputs/output.
func seedCELRun(t *testing.T, ctx context.Context, m *Manager, accountID, envID uuid.UUID, runID string, queuedAt time.Time, r celTestRun) {
	t.Helper()
	var output any
	if r.output != nil {
		output = json.RawMessage(*r.output)
	}
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, scheduled_at, app_id, app_name, function_id, function_slug, attributes, inputs, status, output)
		 VALUES (?, ?, ?, ?, ?, ?, 'test-app', ?, 'test-fn', '{}', ?, ?, ?);`,
		accountID.String(), envID.String(), runID, queuedAt, queuedAt, uuid.New().String(), uuid.New().String(),
		json.RawMessage(r.inputs), enums.StepStatusCompleted.String(), output,
	)
	require.NoError(t, err)
}

// TestGetTraceRunsCELMatchesExpressionSemantics checks GetTraceRuns and
// GetTraceRunsCount against what each CEL expression actually means, for
// shapes whose SQL translation used to be too narrow, too wide, or missing.
func TestGetTraceRunsCELMatchesExpressionSemantics(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	useInsightsCEL(t)
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID := uuid.New(), uuid.New()
	runs := []celTestRun{
		1: {`[{"name":"x","data":{"a":1,"b":2}}]`, new(`{"data":{"ok":true}}`)},
		2: {`[{"name":"x","data":{"a":1,"b":3}}]`, new(`{"data":{"ok":false}}`)},
		3: {`[{"name":"y","data":{"a":"1","b":2}}]`, new(`{"data":"x"}`)},
		4: {`[{"name":"y","data":{"a":true,"c":"true"}}]`, new(`{"data":null}`)},
		5: {`[{"name":"z","data":{"a":-1,"a.b":"x","s":"xyz","list":[1,2]}}]`, new(`{"error":{"message":"boom"}}`)},
		// A batch: neither event alone has a == 1 && name == "x".
		6: {`[{"name":"x","data":{"a":2}},{"name":"y","data":{"a":1}}]`, new(`{"data":{"ok":true}}`)},
		// A cron run, with no triggering event.
		7: {`[]`, new(`{"data":{"ok":true}}`)},
		8: {`[{"name":"xa","data":{"a":1,"b":1}}]`, nil},
	}
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	for i := 1; i < len(runs); i++ {
		seedCELRun(t, ctx, m, accountID, envID, fmt.Sprintf("run-%02d", i), base.Add(time.Duration(i)*time.Second), runs[i])
	}

	// pkg/run rejects an expression with nothing expr can parse at all
	// (e.g. a bare event.name.startsWith("x")), so shapes like that are
	// ANDed with event.v == null, which every test event satisfies.
	tests := []struct {
		cel  string
		want []int
	}{
		{`!(event.data.a == 1 && event.data.b == 2)`, []int{2, 3, 4, 5, 6, 8}},
		{`!(event.data.a == "x" || event.data.b == 2)`, []int{2, 4, 5, 6, 8}},
		{`event.data.a == event.data.b`, []int{8}},
		{`event.data.a == b"x"`, []int{}},
		{`event.v == null && event.data.a == -1`, []int{5}},
		{`event.v == null && event.data.a in [1, 2]`, []int{1, 2, 6, 8}},
		{`event.v == null && event.data["a.b"] == "x"`, []int{5}},
		{`event.v == null && event.name.startsWith("x")`, []int{1, 2, 6, 8}},
		{`event.v == null && event.data.s.contains("y")`, []int{5}},
		{`event.v == null && size(event.data.list) > 1`, []int{5}},
		{`event.foo == 2`, []int{}},
		{`event.data.a == 1 || event.foo == 2`, []int{1, 2, 6, 8}},
		// Type confusion: "1" isn't 1 and "true" isn't true.
		{`event.data.a == 1`, []int{1, 2, 6, 8}},
		{`event.data.a == true`, []int{4}},
		{`event.data.c == true`, []int{}},
		{`event.data.a != 1`, []int{3, 4, 5, 6}},
		{`event.data.b == null`, []int{4, 5, 6}},
		{`event.name == "x" && event.data.a == 1`, []int{1, 2}},
		{`output.ok == true`, []int{1, 6, 7}},
		{`output.ok != true`, []int{2, 3, 4, 5}},
		{`error.message == "boom"`, []int{5}},
		// pkg/expressions fails the whole expression when any part of it
		// errors (here "x".ok), even under an || that's already true.
		{`output == "x" || output.ok == false`, []int{2}},
		// Like pkg/cqrs/manager, a mixed expression is checked once against
		// the triggering events alone and once against the output alone
		// (the other namespace reads as null each time), so a mixed ||
		// needs both halves and a mixed && never matches. A run w/o
		// output (8) or w/o a triggering event (7) never matches either.
		{`event.name == "x" && output.ok == true`, []int{}},
		{`event.name == "x" || output.ok == true`, []int{1, 6}},
		{`event.name == "xa" || output.ok == true`, []int{}},
		{`event.name == "z" || output.ok == true`, []int{}},
		{`event.name == "z" || error.message == "boom"`, []int{5}},
	}

	for _, test := range tests {
		t.Run(test.cel, func(t *testing.T) {
			opt := cqrs.GetTraceRunOpt{
				Filter: cqrs.GetTraceRunFilter{
					AccountID: accountID, WorkspaceID: envID,
					TimeField: enums.TraceRunTimeQueuedAt,
					From:      base.Add(-time.Minute), Until: base.Add(time.Hour),
					CEL: test.cel,
				},
				Order: []cqrs.GetTraceRunOrder{{Field: enums.TraceRunTimeQueuedAt, Direction: enums.TraceRunOrderAsc}},
				Items: 40,
			}
			got, err := m.GetTraceRuns(ctx, opt)
			require.NoError(t, err)
			ids := []string{}
			for _, r := range got {
				ids = append(ids, r.RunID)
			}
			want := []string{}
			for _, i := range test.want {
				want = append(want, fmt.Sprintf("run-%02d", i))
			}
			require.Equal(t, want, ids)

			count, err := m.GetTraceRunsCount(ctx, opt)
			require.NoError(t, err)
			require.Equal(t, len(test.want), count)
		})
	}
}

// TestGetTraceRunsCELPagesPastNonMatchingCandidates checks that a selective
// CEL filter the SQL can't apply (so every run is a candidate) still fills
// each page and paginates through every match exactly once, even when the
// matches are spread over more candidates than one batch holds.
func TestGetTraceRunsCELPagesPastNonMatchingCandidates(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	useInsightsCEL(t)
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID := uuid.New(), uuid.New()
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	const total = 3 * celMinCandidateBatch
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, scheduled_at, app_id, app_name, function_id, function_slug, attributes, inputs, status)
		 SELECT ?, ?, printf('run-%04d', i), ?::TIMESTAMPTZ + to_milliseconds(i // 2), ?::TIMESTAMPTZ, ?, 'test-app', ?, 'test-fn', '{}',
		        json_array(json_object('name', 'x', 'data', json_object('a', i % 100)))::VARIANT, ?
		 FROM range(?) t(i);`,
		accountID.String(), envID.String(), base, base, uuid.New().String(), uuid.New().String(),
		enums.StepStatusCompleted.String(), total,
	)
	require.NoError(t, err)

	// Negated logic, so the SQL can't narrow the candidates at all.
	cel := `!(event.data.a != 0 || event.name != "x")`
	want := []string{}
	for i := total - 1; i >= 0; i-- {
		if i%100 == 0 {
			want = append(want, fmt.Sprintf("run-%04d", i))
		}
	}

	opt := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField: enums.TraceRunTimeQueuedAt,
			From:      base.Add(-time.Minute), Until: base.Add(time.Hour),
			CEL: cel,
		},
		Items: 4,
	}
	got := []string{}
	for page := 0; page < 10; page++ {
		res, err := m.GetTraceRuns(ctx, opt)
		require.NoError(t, err)
		for _, r := range res {
			got = append(got, r.RunID)
		}
		if uint(len(res)) < opt.Items {
			break
		}
		require.Len(t, res, int(opt.Items))
		opt.Cursor = res[len(res)-1].Cursor
	}
	require.Equal(t, want, got)

	count, err := m.GetTraceRunsCount(ctx, opt)
	require.NoError(t, err)
	require.Equal(t, len(want), count)
}
