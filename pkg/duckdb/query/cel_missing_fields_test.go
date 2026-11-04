package query

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/run"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// goEventMatches is what pkg/cqrs/manager would decide for one event:
// run.ExpressionHandler.MatchEventExpressions against event.Event.Map().
func goEventMatches(t *testing.T, cel string, evt event.Event) bool {
	t.Helper()
	h, err := run.NewExpressionHandler(t.Context(), run.WithExpressionHandlerExpressions([]string{cel}))
	require.NoError(t, err)
	ok, err := h.MatchEventExpressions(t.Context(), evt)
	require.NoError(t, err)
	return ok
}

// TestEventsCELMissingFields checks event.id/name/v/ts/data predicates on
// events whose fields are empty, which event.Event.Map() exposes as ""/0/
// a missing v rather than as missing keys, so the SQL prefilter can't drop
// them.
func TestEventsCELMissingFields(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	useInsightsCEL(t)
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	evts := []struct{ id, name, v string }{
		1: {"", "a", ""},
		2: {"id-2", "a", "1"},
		3: {"id-3", "", "2"},
	}
	ids := make([]ulid.ULID, len(evts))
	for i := 1; i < len(evts); i++ {
		ids[i] = ulid.MustNew(ulid.Timestamp(now.Add(time.Duration(i)*time.Millisecond)), nil)
		_, err := m.db.ExecContext(ctx,
			`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts)
			 VALUES (?, ?, ?, ?, 'test', ?, ?, '{}'::JSON, ?, ?);`,
			accountID.String(), envID.String(), ids[i].String(), now, evts[i].id, evts[i].name, evts[i].v, now,
		)
		require.NoError(t, err)
	}

	tests := []struct {
		cel  string
		want []int
	}{
		{`event.v != ""`, []int{1, 2, 3}},
		{`event.v != "1"`, []int{1, 3}},
		{`event.v == "1"`, []int{2}},
		{`event.v == ""`, []int{}},
		{`event.v == null`, []int{1}},
		{`event.v != null`, []int{2, 3}},
		{`event.id == ""`, []int{1}},
		{`event.id != "id-2"`, []int{1, 3}},
		{`event.name == ""`, []int{3}},
		{`event.data != null`, []int{1, 2, 3}},
		{`event.ts > 0`, []int{1, 2, 3}},
	}
	for _, test := range tests {
		t.Run(test.cel, func(t *testing.T) {
			res, err := m.GetEvents(ctx, accountID, envID, &cqrs.WorkspaceEventsOpts{
				Limit: 10, Newest: now.Add(time.Minute), Oldest: now.Add(-time.Minute),
				IncludeInternalEvents: true, CEL: test.cel,
			})
			require.NoError(t, err)
			got := map[ulid.ULID]bool{}
			for _, e := range res {
				got[e.ID] = true
			}

			want := map[ulid.ULID]bool{}
			for _, i := range test.want {
				want[ids[i]] = true
			}
			// Sanity check the expectations against the Go evaluator.
			for i := 1; i < len(evts); i++ {
				e := event.Event{ID: evts[i].id, Name: evts[i].name, Version: evts[i].v, Timestamp: now.UnixMilli(), Data: map[string]any{}}
				require.Equal(t, want[ids[i]], goEventMatches(t, test.cel, e), "event %d", i)
			}
			require.Equal(t, want, got)
		})
	}
}

// TestTraceRunsCELMissingEventFields is TestEventsCELMissingFields for run
// inputs, where a missing/null/"" JSON field is what event.Event.Map()
// reads as ""/0/a missing v.
func TestTraceRunsCELMissingEventFields(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	useInsightsCEL(t)
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID := uuid.New(), uuid.New()
	inputs := []string{
		1: `[{"name":"a","data":{}}]`,
		2: `[{"name":"a","id":"id-2","v":"1","ts":5,"data":{}}]`,
		3: `[{"id":"id-3","v":"","ts":null}]`,
		4: `[{"name":"b","v":null,"data":null}]`,
	}
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	for i := 1; i < len(inputs); i++ {
		seedCELRun(t, ctx, m, accountID, envID, fmt.Sprintf("run-%02d", i), base.Add(time.Duration(i)*time.Second), celTestRun{inputs: inputs[i]})
	}

	tests := []struct {
		cel  string
		want []int
	}{
		{`event.v != ""`, []int{1, 2, 3, 4}},
		{`event.v != "1"`, []int{1, 3, 4}},
		{`event.v == "1"`, []int{2}},
		{`event.v == null`, []int{1, 3, 4}},
		{`event.v != null`, []int{2}},
		{`event.id == ""`, []int{1, 4}},
		{`event.id != "id-2"`, []int{1, 3, 4}},
		{`event.name == ""`, []int{3}},
		{`event.name != "a"`, []int{3, 4}},
		{`event.ts == 0`, []int{1, 3, 4}},
		{`event.ts < 10`, []int{1, 2, 3, 4}},
		{`event.data != null`, []int{1, 2, 3, 4}},
		{`event.data == null`, []int{}},
		{`event.data.a == null`, []int{1, 2, 3, 4}},
	}
	for _, test := range tests {
		t.Run(test.cel, func(t *testing.T) {
			wantSet := map[int]bool{}
			for _, i := range test.want {
				wantSet[i] = true
			}
			// Sanity check the expectations against the Go evaluator.
			for i := 1; i < len(inputs); i++ {
				var evts []event.Event
				require.NoError(t, json.Unmarshal([]byte(inputs[i]), &evts))
				require.Equal(t, wantSet[i], goEventMatches(t, test.cel, evts[0]), "run %d", i)
			}

			opt := cqrs.GetTraceRunOpt{
				Filter: cqrs.GetTraceRunFilter{
					AccountID: accountID, WorkspaceID: envID,
					TimeField: enums.TraceRunTimeQueuedAt,
					From:      base.Add(-time.Minute), Until: base.Add(time.Hour),
					CEL: test.cel,
				},
				Order: []cqrs.GetTraceRunOrder{{Field: enums.TraceRunTimeQueuedAt, Direction: enums.TraceRunOrderAsc}},
				Items: 10,
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
		})
	}
}
