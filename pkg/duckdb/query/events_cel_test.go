package query

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestGetEventsCELMatchesExpressionSemantics is the events-table
// counterpart of TestGetTraceRunsCELMatchesExpressionSemantics, covering
// GetEvents (with its batched pagination), GetEventsCount, and
// GetEventsByExpressions.
func TestGetEventsCELMatchesExpressionSemantics(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	useInsightsCEL(t)
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	evts := []struct {
		name string
		data string
	}{
		1: {"x", `{"a":1,"b":2}`},
		2: {"x", `{"a":1,"b":3}`},
		3: {"y", `{"a":"1","b":2}`},
		4: {"y", `{"a":true,"c":"true"}`},
		5: {"z", `{"a":-1,"a.b":"x","s":"xyz","list":[1,2]}`},
		6: {"xa", `{"a":1,"b":1}`},
	}
	ids := make([]ulid.ULID, len(evts))
	for i := 1; i < len(evts); i++ {
		ids[i] = ulid.MustNew(ulid.Timestamp(now.Add(time.Duration(i)*time.Millisecond)), nil)
		_, err := m.db.ExecContext(ctx,
			`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts)
			 VALUES (?, ?, ?, ?, 'test', ?, ?, ?, '', ?);`,
			accountID.String(), envID.String(), ids[i].String(), now, fmt.Sprintf("evt-%d", i), evts[i].name, json.RawMessage(evts[i].data), now,
		)
		require.NoError(t, err)
	}

	tests := []struct {
		cel  string
		want []int
	}{
		{`!(event.data.a == 1 && event.data.b == 2)`, []int{2, 3, 4, 5, 6}},
		{`!(event.data.a == "x" || event.data.b == 2)`, []int{2, 4, 5, 6}},
		{`event.data.a == event.data.b`, []int{6}},
		{`event.data.a == b"x"`, []int{}},
		{`event.v == null && event.data.a == -1`, []int{5}},
		{`event.v == null && event.data.a in [1, 2]`, []int{1, 2, 6}},
		{`event.v == null && event.data["a.b"] == "x"`, []int{5}},
		{`event.v == null && event.name.startsWith("x")`, []int{1, 2, 6}},
		{`event.v == null && event.data.s.contains("y")`, []int{5}},
		{`event.v == null && size(event.data.list) > 1`, []int{5}},
		{`event.foo == 2`, []int{}},
		{`event.data.a == 1 || event.foo == 2`, []int{1, 2, 6}},
		{`event.data.a == 1`, []int{1, 2, 6}},
		{`event.data.a == true`, []int{4}},
		{`event.data.c == true`, []int{}},
		{`event.data.a != 1`, []int{3, 4, 5}},
		{`event.data.b == null`, []int{4, 5}},
		{`event.name != "x"`, []int{3, 4, 5, 6}},
		{`event.id == 1`, []int{}},
	}

	for _, test := range tests {
		t.Run(test.cel, func(t *testing.T) {
			// GetEvents/ByExpressions order newest first.
			want := []string{}
			for i := len(test.want) - 1; i >= 0; i-- {
				want = append(want, ids[test.want[i]].String())
			}
			eventIDs := func(res []*cqrs.Event) []string {
				out := []string{}
				for _, e := range res {
					out = append(out, e.ID.String())
				}
				return out
			}

			// Page through with a limit of 1 so every page has to skip
			// non-matching candidates.
			opts := &cqrs.WorkspaceEventsOpts{
				Limit: 1, Newest: now.Add(time.Minute), Oldest: now.Add(-time.Minute),
				IncludeInternalEvents: true, CEL: test.cel,
			}
			got := []string{}
			for range evts {
				res, err := m.GetEvents(ctx, accountID, envID, opts)
				require.NoError(t, err)
				if len(res) == 0 {
					break
				}
				got = append(got, eventIDs(res)...)
				opts.Cursor = &res[len(res)-1].ID
			}
			require.Equal(t, want, got)

			count, err := m.GetEventsCount(ctx, accountID, envID, cqrs.WorkspaceEventsOpts{
				Limit: 1, Newest: now.Add(time.Minute), Oldest: now.Add(-time.Minute),
				IncludeInternalEvents: true, CEL: test.cel,
			})
			require.NoError(t, err)
			require.EqualValues(t, len(test.want), count)

			byExpr, err := m.GetEventsByExpressions(ctx, []string{
				fmt.Sprintf(`event.name != "%s"`, uuid.NewString()), test.cel,
			})
			require.NoError(t, err)
			gotByExpr := map[string]bool{}
			for _, id := range eventIDs(byExpr) {
				gotByExpr[id] = true
			}
			wantByExpr := map[string]bool{}
			for _, id := range want {
				wantByExpr[id] = true
			}
			require.Equal(t, wantByExpr, gotByExpr)
		})
	}
}

// TestGetEventsCELStreamsPastNonMatchingCandidates covers a CEL filter the
// SQL can't narrow at all (every event is a candidate) over more events
// than one candidate batch holds: GetEvents has to page past whole batches
// of non-matches and stop mid-batch once a page fills, and GetEventsCount
// and GetEventsByExpressions have to see every candidate.
func TestGetEventsCELStreamsPastNonMatchingCandidates(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	useInsightsCEL(t)
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	const total = 3 * celMinCandidateBatch
	ids := make([]ulid.ULID, total)
	for i := range ids {
		ids[i] = ulid.MustNew(ulid.Timestamp(now.Add(time.Duration(i)*time.Millisecond)), nil)
	}
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts)
		 SELECT ?, ?, (?::VARCHAR[])[i + 1], ?, 'test', printf('evt-%d', i), 'x', json_object('a', i % 100), '', ?
		 FROM range(?) t(i);`,
		accountID.String(), envID.String(), ulidStrings(ids), now, now, total,
	)
	require.NoError(t, err)

	cel := `event.data.a != 1 && event.data.a != 2 && !(event.data.a != 0 || event.name != "x")`
	want := []string{}
	for i := total - 1; i >= 0; i-- {
		if i%100 == 0 {
			want = append(want, ids[i].String())
		}
	}

	opts := &cqrs.WorkspaceEventsOpts{
		Limit: 2, Newest: now.Add(time.Minute), Oldest: now.Add(-time.Minute),
		IncludeInternalEvents: true, CEL: cel,
	}
	got := []string{}
	for range total {
		res, err := m.GetEvents(ctx, accountID, envID, opts)
		require.NoError(t, err)
		for _, e := range res {
			got = append(got, e.ID.String())
		}
		if len(res) < opts.Limit {
			break
		}
		opts.Cursor = &res[len(res)-1].ID
	}
	require.Equal(t, want, got)

	count, err := m.GetEventsCount(ctx, accountID, envID, cqrs.WorkspaceEventsOpts{
		Limit: 2, Newest: now.Add(time.Minute), Oldest: now.Add(-time.Minute),
		IncludeInternalEvents: true, CEL: cel,
	})
	require.NoError(t, err)
	require.EqualValues(t, len(want), count)

	byExpr, err := m.GetEventsByExpressions(ctx, []string{cel})
	require.NoError(t, err)
	require.Len(t, byExpr, len(want))
}

func ulidStrings(ids []ulid.ULID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
