package duckdbquery

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// seedRunRow inserts one lifecycle row into inngest.runs, mirroring the
// columns pkg/execution/dualwrite/listener.go's runCommonFields plus a
// specific hook sets.
func seedRunRow(t *testing.T, ctx context.Context, m *Manager, accountID, envID, appID, functionID uuid.UUID, runID ulid.ULID, queuedAt time.Time, status *enums.StepStatus, startedAt, endedAt *time.Time, output *string) {
	t.Helper()
	cols := []string{"account_id", "env_id", "run_id", "queued_at", "scheduled_at", "app_id", "function_id", "inputs"}
	args := []any{accountID.String(), envID.String(), runID.String(), queuedAt, queuedAt, appID.String(), functionID.String(), "[]"}
	if status != nil {
		cols = append(cols, "status")
		args = append(args, status.String())
	}
	if startedAt != nil {
		cols = append(cols, "started_at")
		args = append(args, *startedAt)
	}
	if endedAt != nil {
		cols = append(cols, "ended_at")
		args = append(args, *endedAt)
	}
	if output != nil {
		cols = append(cols, "output")
		args = append(args, *output)
	}
	placeholders := make([]string, len(cols))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	q := "INSERT INTO inngest.runs (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ");"
	_, err := m.db.ExecContext(ctx, q, args...)
	require.NoError(t, err)
}

func TestGetTraceRunReturnsLatestRowForRun(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	queuedAt := time.Now().UTC().Add(-time.Minute)
	startedAt := queuedAt.Add(time.Second)
	endedAt := startedAt.Add(time.Second)

	queued := enums.StepStatusQueued
	running := enums.StepStatusRunning
	completed := enums.StepStatusCompleted
	out := `{"data":"done"}`

	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, runID, queuedAt, &queued, nil, nil, nil)
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, runID, queuedAt, &running, &startedAt, nil, nil)
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, runID, queuedAt, &completed, &startedAt, &endedAt, &out)

	run, err := m.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: runID})
	require.NoError(t, err)
	require.Equal(t, enums.RunStatusCompleted, run.Status)
	require.WithinDuration(t, endedAt, run.EndedAt, time.Millisecond)
	// TraceRun.Output is the raw column content, unwrapped-envelope-free —
	// no data/error unwrap happens at the run level (only GetSpanOutput does
	// that, for per-span output).
	require.JSONEq(t, out, string(run.Output))
}

func TestGetTraceRunParsesEventIDsArray(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	evt1 := ulid.MustNew(ulid.Now(), rand.Reader)
	evt2 := ulid.MustNew(ulid.Now(), rand.Reader)
	queuedAt := time.Now().UTC()

	_, err := m.db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, scheduled_at, app_id, function_id, inputs, status, event_ids)
		 VALUES (?, ?, ?, ?, ?, ?, ?, '[]', ?, ?);`,
		accountID.String(), envID.String(), runID.String(), queuedAt, queuedAt, appID.String(), functionID.String(),
		enums.StepStatusQueued.String(), []string{evt1.String(), evt2.String()},
	)
	require.NoError(t, err)

	run, err := m.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: runID})
	require.NoError(t, err)
	require.Equal(t, []string{evt1.String(), evt2.String()}, run.TriggerIDs)
}

func TestGetTraceRunHasNilTriggerIDsForCronRuns(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	queuedAt := time.Now().UTC()
	queued := enums.StepStatusQueued

	// A cron-triggered run never sets event_ids — no triggering event exists.
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, runID, queuedAt, &queued, nil, nil, nil)

	run, err := m.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: runID})
	require.NoError(t, err)
	require.Nil(t, run.TriggerIDs)
}

func TestGetTraceRunsByTriggerIDReturnsMatchingRuns(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	trigger := ulid.MustNew(ulid.Now(), rand.Reader)
	other := ulid.MustNew(ulid.Now(), rand.Reader)
	now := time.Now().UTC()

	matchRun := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, scheduled_at, app_id, function_id, inputs, status, event_ids)
		 VALUES (?, ?, ?, ?, ?, ?, ?, '[]', ?, ?);`,
		accountID.String(), envID.String(), matchRun.String(), now, now, appID.String(), functionID.String(),
		enums.StepStatusQueued.String(), []string{trigger.String()},
	)
	require.NoError(t, err)

	// A run triggered by a different event must not match.
	otherRun := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	_, err = m.db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, scheduled_at, app_id, function_id, inputs, status, event_ids)
		 VALUES (?, ?, ?, ?, ?, ?, ?, '[]', ?, ?);`,
		accountID.String(), envID.String(), otherRun.String(), now, now, appID.String(), functionID.String(),
		enums.StepStatusQueued.String(), []string{other.String()},
	)
	require.NoError(t, err)

	// A cron-only run (nil event_ids) must not match either.
	cronQueued := enums.StepStatusQueued
	cronRun := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, cronRun, now, &cronQueued, nil, nil, nil)

	runs, err := m.GetTraceRunsByTriggerID(ctx, trigger)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, matchRun.String(), runs[0].RunID)
	require.Equal(t, []string{trigger.String()}, runs[0].TriggerIDs)
}

// TestGetTraceRunsByTriggerIDCollapsesToLatestRow proves a run with
// multiple lifecycle rows (inngest.runs is append-only) surfaces once, at
// its latest status — matching GetTraceRuns' own collapse behavior.
func TestGetTraceRunsByTriggerIDCollapsesToLatestRow(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	trigger := ulid.MustNew(ulid.Now(), rand.Reader)
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	queuedAt := time.Now().UTC().Add(-time.Minute)
	startedAt := queuedAt.Add(time.Second)
	endedAt := startedAt.Add(time.Second)

	for _, row := range []struct {
		status             enums.StepStatus
		startedAt, endedAt *time.Time
	}{
		{enums.StepStatusQueued, nil, nil},
		{enums.StepStatusRunning, &startedAt, nil},
		{enums.StepStatusCompleted, &startedAt, &endedAt},
	} {
		cols := []string{"account_id", "env_id", "run_id", "queued_at", "scheduled_at", "app_id", "function_id", "inputs", "status", "event_ids"}
		args := []any{accountID.String(), envID.String(), runID.String(), queuedAt, queuedAt, appID.String(), functionID.String(), "[]", row.status.String(), []string{trigger.String()}}
		if row.startedAt != nil {
			cols = append(cols, "started_at")
			args = append(args, *row.startedAt)
		}
		if row.endedAt != nil {
			cols = append(cols, "ended_at")
			args = append(args, *row.endedAt)
		}
		placeholders := make([]string, len(cols))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		q := "INSERT INTO inngest.runs (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ");"
		_, err := m.db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
	}

	runs, err := m.GetTraceRunsByTriggerID(ctx, trigger)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, enums.RunStatusCompleted, runs[0].Status)
	require.WithinDuration(t, endedAt, runs[0].EndedAt, time.Millisecond)
}

// TestGetTraceRunsByTriggerIDNoMatchReturnsEmptySlice proves a trigger ID
// with no matching runs returns an empty, non-nil slice — the GQL resolver
// (EventV2.Runs) ranges over the result directly without a nil check.
func TestGetTraceRunsByTriggerIDNoMatchReturnsEmptySlice(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	runs, err := m.GetTraceRunsByTriggerID(ctx, ulid.MustNew(ulid.Now(), rand.Reader))
	require.NoError(t, err)
	require.NotNil(t, runs)
	require.Empty(t, runs)
}

func TestGetTraceRunsFiltersByStatusAndTimeRange(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()

	completed := enums.StepStatusCompleted
	failed := enums.StepStatusFailed

	run1 := ulid.MustNew(ulid.Timestamp(now.Add(-time.Hour)), rand.Reader)
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, run1, now.Add(-time.Hour), &completed, nil, nil, nil)

	run2 := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, run2, now, &failed, nil, nil, nil)

	runs, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField: enums.TraceRunTimeQueuedAt,
			Status:    []enums.RunStatus{enums.RunStatusCompleted},
			From:      now.Add(-2 * time.Hour), Until: now.Add(time.Hour),
		},
		Items: 40,
	})
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, run1.String(), runs[0].RunID)
}

// TestGetTraceRunsFiltersByAppName proves resolveAppAndFunctionFilters
// resolves filter.AppName to a real AppID via the embedded manager's
// GetAppByName (inngest.runs has no app name column to filter on directly)
// and that the resolved AppID actually narrows the result set.
func TestGetTraceRunsFiltersByAppName(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	accountID, envID := uuid.New(), uuid.New()
	matchAppID, otherAppID, functionID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()

	fake := &fakeManager{apps: map[string]*cqrs.App{
		"my-app": {ID: matchAppID, Name: "my-app"},
	}}
	m := Wrap(fake, db).(*Manager)

	completed := enums.StepStatusCompleted
	matchRun := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	seedRunRow(t, ctx, m, accountID, envID, matchAppID, functionID, matchRun, now, &completed, nil, nil, nil)
	otherRun := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	seedRunRow(t, ctx, m, accountID, envID, otherAppID, functionID, otherRun, now, &completed, nil, nil, nil)

	runs, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField: enums.TraceRunTimeQueuedAt,
			AppName:   []string{"my-app"},
			From:      now.Add(-time.Hour), Until: now.Add(time.Hour),
		},
		Items: 40,
	})
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, matchRun.String(), runs[0].RunID)
}

// TestGetTraceRunsAppNameWithNoMatchReturnsEmpty proves a filter.AppName
// that resolves to no real app returns zero runs, not every run — a naive
// merge into an empty AppID list would otherwise silently behave as "no
// filter" instead of "match nothing".
func TestGetTraceRunsAppNameWithNoMatchReturnsEmpty(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()

	fake := &fakeManager{apps: map[string]*cqrs.App{}}
	m := Wrap(fake, db).(*Manager)

	completed := enums.StepStatusCompleted
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, ulid.MustNew(ulid.Timestamp(now), rand.Reader), now, &completed, nil, nil, nil)

	runs, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField: enums.TraceRunTimeQueuedAt,
			AppName:   []string{"no-such-app"},
			From:      now.Add(-time.Hour), Until: now.Add(time.Hour),
		},
		Items: 40,
	})
	require.NoError(t, err)
	require.Empty(t, runs)
}

// TestGetTraceRunsFiltersByFunctionSlug proves filter.FunctionSlug resolves
// via the embedded manager's GetFunctions and narrows the result set, the
// same way AppName does.
func TestGetTraceRunsFiltersByFunctionSlug(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	accountID, envID, appID := uuid.New(), uuid.New(), uuid.New()
	matchFnID, otherFnID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	fake := &fakeManager{functions: []*cqrs.Function{
		{ID: matchFnID, Slug: "my-fn"},
		{ID: otherFnID, Slug: "other-fn"},
	}}
	m := Wrap(fake, db).(*Manager)

	completed := enums.StepStatusCompleted
	matchRun := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	seedRunRow(t, ctx, m, accountID, envID, appID, matchFnID, matchRun, now, &completed, nil, nil, nil)
	otherRun := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	seedRunRow(t, ctx, m, accountID, envID, appID, otherFnID, otherRun, now, &completed, nil, nil, nil)

	runs, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField:    enums.TraceRunTimeQueuedAt,
			FunctionSlug: []string{"my-fn"},
			From:         now.Add(-time.Hour), Until: now.Add(time.Hour),
		},
		Items: 40,
	})
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, matchRun.String(), runs[0].RunID)
}

// TestGetTraceRunsFunctionSlugWithNoMatchReturnsEmpty mirrors
// TestGetTraceRunsAppNameWithNoMatchReturnsEmpty for FunctionSlug.
func TestGetTraceRunsFunctionSlugWithNoMatchReturnsEmpty(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()

	fake := &fakeManager{functions: []*cqrs.Function{{ID: uuid.New(), Slug: "other-fn"}}}
	m := Wrap(fake, db).(*Manager)

	completed := enums.StepStatusCompleted
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, ulid.MustNew(ulid.Timestamp(now), rand.Reader), now, &completed, nil, nil, nil)

	runs, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField:    enums.TraceRunTimeQueuedAt,
			FunctionSlug: []string{"no-such-fn"},
			From:         now.Add(-time.Hour), Until: now.Add(time.Hour),
		},
		Items: 40,
	})
	require.NoError(t, err)
	require.Empty(t, runs)
}

// TestGetTraceRunsPropagatesAppNameResolutionError proves a real (non-
// "not found") error from the embedded manager surfaces to the caller
// rather than being swallowed the same way an unresolvable name is.
func TestGetTraceRunsPropagatesAppNameResolutionError(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	fake := &fakeManager{appErr: errors.New("boom")}
	m := Wrap(fake, db).(*Manager)

	_, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: uuid.New(), WorkspaceID: uuid.New(),
			TimeField: enums.TraceRunTimeQueuedAt,
			AppName:   []string{"my-app"},
			From:      time.Now().Add(-time.Hour), Until: time.Now().Add(time.Hour),
		},
		Items: 40,
	})
	require.ErrorContains(t, err, "boom")
}

// TestGetTraceRunsFiltersByEventID proves filter.EventID matches a run
// whose event_ids array contains ANY of the given IDs — event_ids is a real
// VARCHAR[] column (see the earlier VARCHAR[] migration work), so this is a
// direct list_contains match, no name/slug resolution involved.
func TestGetTraceRunsFiltersByEventID(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	evt1, evt2, unrelatedEvt := ulid.MustNew(ulid.Now(), rand.Reader), ulid.MustNew(ulid.Now(), rand.Reader), ulid.MustNew(ulid.Now(), rand.Reader)

	matchRun := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, scheduled_at, app_id, function_id, inputs, status, event_ids)
		 VALUES (?, ?, ?, ?, ?, ?, ?, '[]', ?, ?);`,
		accountID.String(), envID.String(), matchRun.String(), now, now, appID.String(), functionID.String(),
		enums.StepStatusCompleted.String(), []string{evt1.String()},
	)
	require.NoError(t, err)

	otherRun := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	_, err = m.db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, scheduled_at, app_id, function_id, inputs, status, event_ids)
		 VALUES (?, ?, ?, ?, ?, ?, ?, '[]', ?, ?);`,
		accountID.String(), envID.String(), otherRun.String(), now, now, appID.String(), functionID.String(),
		enums.StepStatusCompleted.String(), []string{unrelatedEvt.String()},
	)
	require.NoError(t, err)

	runs, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField: enums.TraceRunTimeQueuedAt,
			EventID:   []ulid.ULID{evt1, evt2},
			From:      now.Add(-time.Hour), Until: now.Add(time.Hour),
		},
		Items: 40,
	})
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, matchRun.String(), runs[0].RunID)
}

func TestGetTraceRunsAppliesOutputCELFilter(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	completed := enums.StepStatusCompleted

	// inngest.runs.output stores the raw function result directly (see
	// OnFunctionFinished's resp.GetTraceFunctionOutput()) — unlike
	// run_trace_spans.output, this is not wrapped in a {"data":...}/
	// {"error":...} envelope, and run.MatchOutputExpressions evaluates CEL
	// directly against it (data := map[string]any{"output": <this value>}).
	matchOut := `{"ok":true}`
	run1 := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, run1, now, &completed, nil, nil, &matchOut)

	noMatchOut := `{"ok":false}`
	run2 := ulid.MustNew(ulid.Timestamp(now.Add(time.Millisecond)), rand.Reader)
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, run2, now, &completed, nil, nil, &noMatchOut)

	runs, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField: enums.TraceRunTimeQueuedAt,
			From:      now.Add(-time.Hour), Until: now.Add(time.Hour),
			CEL: "output.ok == true",
		},
		Items: 40,
	})
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, run1.String(), runs[0].RunID)
}

// TestGetTraceRunsPaginatesWithCursor is a regression test for a real bug
// found after implementation: GetTraceRuns never decoded or applied
// opt.Cursor at all (every "next page" request just re-ran the same
// unfiltered query), and the response cursor it set (trun.Cursor =
// trun.RunID) wasn't even the encoded cqrs.TracePageCursor shape the next
// request's opt.Cursor needs to decode. This walks three pages of five
// runs (page size 2) via the real cursor round-trip and asserts every run
// is seen exactly once, in the expected (queued_at DESC) order.
func TestGetTraceRunsPaginatesWithCursor(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	completed := enums.StepStatusCompleted

	runIDs := make([]ulid.ULID, 5)
	for i := range runIDs {
		// Well-separated (whole seconds apart) so TIMESTAMP_MS truncation
		// never ties two runs' queued_at.
		ts := now.Add(time.Duration(i) * time.Second)
		runIDs[i] = ulid.MustNew(ulid.Timestamp(ts), rand.Reader)
		seedRunRow(t, ctx, m, accountID, envID, appID, functionID, runIDs[i], ts, &completed, nil, nil, nil)
	}
	// Expected order is queued_at DESC (the default), i.e. reverse insertion order.
	want := []string{runIDs[4].String(), runIDs[3].String(), runIDs[2].String(), runIDs[1].String(), runIDs[0].String()}

	filter := cqrs.GetTraceRunFilter{
		AccountID: accountID, WorkspaceID: envID,
		TimeField: enums.TraceRunTimeQueuedAt,
		From:      now.Add(-time.Hour), Until: now.Add(time.Hour),
	}

	var got []string
	cursor := ""
	for page := 0; page < 10; page++ { // bounded loop; a real bug here would otherwise hang forever
		runs, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{Filter: filter, Items: 2, Cursor: cursor})
		require.NoError(t, err)
		if len(runs) == 0 {
			break
		}
		for _, r := range runs {
			got = append(got, r.RunID)
		}
		cursor = runs[len(runs)-1].Cursor
		require.NotEmpty(t, cursor, "GetTraceRuns must set a real cursor on each returned run")
	}

	require.Equal(t, want, got, "pagination must visit every run exactly once, in order, across pages")
}

func TestGetTraceRunsCountCollapsesToOnePerRun(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	queued, running, completed := enums.StepStatusQueued, enums.StepStatusRunning, enums.StepStatusCompleted

	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, runID, now, &queued, nil, nil, nil)
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, runID, now, &running, nil, nil, nil)
	seedRunRow(t, ctx, m, accountID, envID, appID, functionID, runID, now, &completed, nil, nil, nil)

	count, err := m.GetTraceRunsCount(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField: enums.TraceRunTimeQueuedAt,
			From:      now.Add(-time.Hour), Until: now.Add(time.Hour),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, count, "3 lifecycle rows for one run must count as 1 run, not 3")
}

// TestGetTraceRunsCountRespectsAppNameFilter proves GetTraceRunsCount shares
// GetTraceRuns' resolveAppAndFunctionFilters call, including the
// no-match-means-zero behavior — not just the CTE/postWhere building it
// otherwise shares.
func TestGetTraceRunsCountRespectsAppNameFilter(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	accountID, envID, matchAppID, otherAppID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()

	fake := &fakeManager{apps: map[string]*cqrs.App{"my-app": {ID: matchAppID, Name: "my-app"}}}
	m := Wrap(fake, db).(*Manager)

	completed := enums.StepStatusCompleted
	seedRunRow(t, ctx, m, accountID, envID, matchAppID, functionID, ulid.MustNew(ulid.Timestamp(now), rand.Reader), now, &completed, nil, nil, nil)
	seedRunRow(t, ctx, m, accountID, envID, otherAppID, functionID, ulid.MustNew(ulid.Timestamp(now), rand.Reader), now, &completed, nil, nil, nil)

	count, err := m.GetTraceRunsCount(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField: enums.TraceRunTimeQueuedAt,
			AppName:   []string{"my-app"},
			From:      now.Add(-time.Hour), Until: now.Add(time.Hour),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, count)

	count, err = m.GetTraceRunsCount(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: accountID, WorkspaceID: envID,
			TimeField: enums.TraceRunTimeQueuedAt,
			AppName:   []string{"no-such-app"},
			From:      now.Add(-time.Hour), Until: now.Add(time.Hour),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
