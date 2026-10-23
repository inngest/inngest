package devserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	cqrsmanager "github.com/inngest/inngest/pkg/cqrs/manager"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestReplayRejectsFunctionsFromAnotherStoredEnvironment(t *testing.T) {
	for _, byUUID := range []bool{false, true} {
		t.Run(fmt.Sprintf("uuid=%t", byUUID), func(t *testing.T) {
			fn, run, evt := replayFixture()
			config, err := json.Marshal(fn.Function)
			require.NoError(t, err)
			other := &cqrs.Function{ID: uuid.New(), EnvID: uuid.New(), AppID: fn.AppID, Slug: "foreign-function", Config: config}
			store := &fakeFunctionStore{app: &cqrs.App{ID: fn.AppID, Name: fn.AppName}, fns: []*cqrs.Function{
				{ID: fn.ID, AppID: fn.AppID, Slug: fn.Slug, Config: config},
				other,
			}}
			var queries atomic.Int32
			data := &replayData{events: []*cqrs.Event{evt}, runs: func(context.Context, cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
				queries.Add(1)
				return []*cqrs.TraceRun{run}, nil
			}}
			scheduler := newReplayScheduler()
			provider := NewReplayProvider(data, NewFunctionProvider(store, nil, nil), scheduler, nil)
			wrongApp := replayOpts(fn.Slug)
			wrongApp.AppID = "another-app"
			_, err = provider.CreateReplay(t.Context(), wrongApp)
			require.ErrorIs(t, err, apiv2.ErrFunctionNotFound)
			secondID := other.Slug
			if byUUID {
				secondID = other.ID.String()
			}
			result, err := provider.CreateReplay(t.Context(), replayOpts(secondID))
			require.ErrorIs(t, err, apiv2.ErrFunctionNotFound)
			require.Equal(t, apiv2.ReplayResult{}, result)
			require.Zero(t, queries.Load())
			require.Empty(t, scheduler.calls)

			for _, envID := range []uuid.UUID{uuid.Nil, consts.DevServerEnvID} {
				store.fns[0].EnvID = envID
				result, err = provider.CreateReplay(t.Context(), replayOpts(fn.Slug))
				require.NoError(t, err)
				call := receiveReplay(t, scheduler.calls)
				require.Equal(t, result.ID, *call.request.ReplayID)
				require.Equal(t, fn.ID, call.request.Function.ID)
			}
		})
	}
}

func TestReplayRejectsPausedFunctionsUntilUnpaused(t *testing.T) {
	for _, source := range []string{"function", "queue override"} {
		for _, status := range []enums.ReplayRunStatus{enums.ReplayRunStatusFailed, enums.ReplayRunStatusAll, enums.ReplayRunStatusSkippedPaused} {
			t.Run(fmt.Sprintf("%s/%s", source, status), func(t *testing.T) {
				fn, run, evt := replayFixture()
				run.Status = enums.RunStatusFailed
				if status == enums.ReplayRunStatusSkippedPaused {
					run.Status = enums.RunStatusSkipped
					reason := enums.SkipReasonFunctionPaused
					run.SkipReason = &reason
				}
				var paused atomic.Bool
				paused.Store(true)
				functions := replayFunctions{get: func(context.Context, string) (inngest.DeployedFunction, error) {
					result := fn
					if source == "function" && paused.Load() {
						result.PausedAt = time.Now()
					}
					return result, nil
				}}
				var queries atomic.Int32
				data := &replayData{
					runs: func(context.Context, cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
						queries.Add(1)
						return []*cqrs.TraceRun{run}, nil
					},
					events: []*cqrs.Event{evt},
				}
				scheduler := newReplayScheduler()
				provider := NewReplayProvider(data, functions, scheduler, func(id uuid.UUID) bool {
					require.Equal(t, fn.ID, id)
					return source == "queue override" && paused.Load()
				})
				opts := replayOpts(fn.Slug)
				opts.Statuses = []enums.ReplayRunStatus{status}
				result, err := provider.CreateReplay(t.Context(), opts)
				require.ErrorIs(t, err, apiv2.ErrFunctionPaused)
				require.Equal(t, apiv2.ReplayResult{}, result)
				require.Zero(t, queries.Load())
				require.Empty(t, scheduler.calls)

				paused.Store(false)
				result, err = provider.CreateReplay(t.Context(), opts)
				require.NoError(t, err)
				call := receiveReplay(t, scheduler.calls)
				require.Equal(t, result.ID, *call.request.ReplayID)
				require.Equal(t, run.RunID, call.request.OriginalRunID.String())
			})
		}
	}
}

func TestReplayPreservesBatchEvents(t *testing.T) {
	fn, run, evt := replayFixture()
	second := *evt
	second.ID = ulid.Make()
	second.EventID = "second-event"
	second.EventData = map[string]any{"position": "second"}
	run.TriggerIDs = append(run.TriggerIDs, second.ID.String())
	batchID := ulid.Make()
	run.BatchID = &batchID
	queries := make(chan cqrs.GetTraceRunOpt, 2)
	data := &replayData{
		runs: func(_ context.Context, query cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
			queries <- query
			return []*cqrs.TraceRun{run}, nil
		},
		// Database result order must not reorder a batch's trigger events.
		events: []*cqrs.Event{&second, evt},
	}
	scheduler := newReplayScheduler()
	functions := replayFunctions{fn: fn}
	provider := NewReplayProvider(data, functions, scheduler, nil)
	opts := replayOpts(fn.Slug)
	result, err := provider.CreateReplay(t.Context(), opts)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, result.ID)
	require.False(t, result.CreatedAt.IsZero())
	call := receiveReplay(t, scheduler.calls)
	req := call.request
	require.Equal(t, fn.ID, req.Function.ID)
	require.Equal(t, fn.AppID, req.AppID)
	require.Equal(t, fn.AppName, req.AppName)
	require.Equal(t, consts.DevServerAccountID, req.AccountID)
	require.Equal(t, consts.DevServerEnvID, req.WorkspaceID)
	require.Equal(t, result.ID, *req.ReplayID)
	require.Equal(t, run.RunID, req.OriginalRunID.String())
	require.Nil(t, req.IdempotencyKey, "replays use the existing rerun scheduling behavior")
	require.NotNil(t, req.BatchID)
	require.NotEqual(t, ulid.Zero, *req.BatchID)
	require.NotEqual(t, batchID, *req.BatchID)
	require.Nil(t, req.FromStep)
	require.Nil(t, req.RunID)
	require.True(t, req.PreventRateLimit)
	require.True(t, req.PreventDebounce)
	require.Len(t, req.Events, 2)
	require.Equal(t, evt.Event(), req.Events[0].GetEvent())
	require.Equal(t, second.Event(), req.Events[1].GetEvent())
	require.Equal(t, evt.ID, req.Events[0].GetInternalID())
	require.Equal(t, second.ID, req.Events[1].GetInternalID())
	query := receiveReplay(t, queries)
	require.Equal(t, []uuid.UUID{fn.ID}, query.Filter.FunctionID)
	require.Equal(t, consts.DevServerAccountID, query.Filter.AccountID)
	require.Equal(t, consts.DevServerEnvID, query.Filter.WorkspaceID)
	require.Equal(t, opts.From, query.Filter.From)
	require.Equal(t, opts.Until, query.Filter.Until)
	require.Equal(t, enums.TraceRunTimeQueuedAt, query.Filter.TimeField)
	require.Equal(t, []enums.RunStatus{enums.RunStatusCompleted}, query.Filter.Status)
	require.False(t, query.IncludeSkipped)

	// Exercise the real scheduling lifecycle: both runs must retain their
	// complete batch membership in the local database.
	db, err := dbsqlite.Open(t.Context(), dbsqlite.Options{Persist: false, ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := cqrsmanager.New(dbsqlite.New(db))
	lifecycle := Lifecycle{Cqrs: store}
	originalRunID, replayRunID := ulid.MustParse(run.RunID), ulid.Make()
	for _, batch := range []struct{ runID, batchID ulid.ULID }{{originalRunID, batchID}, {replayRunID, *req.BatchID}} {
		lifecycle.OnFunctionScheduled(t.Context(), sv2.Metadata{
			ID: sv2.ID{RunID: batch.runID, FunctionID: fn.ID, Tenant: sv2.Tenant{
				AccountID: req.AccountID, EnvID: req.WorkspaceID, AppID: req.AppID,
			}},
			Config: *sv2.InitConfig(&sv2.Config{BatchID: &batch.batchID, EventIDs: []ulid.ULID{evt.ID, second.ID}}),
		}, queue.Item{}, req.Events)
		persisted, err := store.GetEventBatchByRunID(t.Context(), batch.runID)
		require.NoError(t, err)
		require.Equal(t, batch.batchID, persisted.ID)
		require.Equal(t, []ulid.ULID{evt.ID, second.ID}, persisted.EventIDs())
	}
}

func TestReplayStatusesAndPausedSkips(t *testing.T) {
	for _, tc := range []struct {
		name      string
		selectors []enums.ReplayRunStatus
		want      []enums.RunStatus
	}{
		{"all", []enums.ReplayRunStatus{enums.ReplayRunStatusAll}, []enums.RunStatus{enums.RunStatusCompleted, enums.RunStatusFailed, enums.RunStatusCancelled, enums.RunStatusSkipped}},
		{"completed", []enums.ReplayRunStatus{enums.ReplayRunStatusCompleted}, []enums.RunStatus{enums.RunStatusCompleted}},
		{"failed and cancelled", []enums.ReplayRunStatus{enums.ReplayRunStatusFailed, enums.ReplayRunStatusCancelled}, []enums.RunStatus{enums.RunStatusFailed, enums.RunStatusCancelled}},
		{"paused", []enums.ReplayRunStatus{enums.ReplayRunStatusSkippedPaused}, []enums.RunStatus{enums.RunStatusSkipped}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn, template, evt := replayFixture()
			paused, singleton := enums.SkipReasonFunctionPaused, enums.SkipReasonSingleton
			var rows []*cqrs.TraceRun
			for _, status := range []enums.RunStatus{enums.RunStatusCompleted, enums.RunStatusFailed, enums.RunStatusCancelled, enums.RunStatusRunning, enums.RunStatusOverflowed, enums.RunStatusSkipped, enums.RunStatusSkipped, enums.RunStatusSkipped} {
				row := *template
				row.RunID = ulid.Make().String()
				row.Status = status
				rows = append(rows, &row)
			}
			rows[5].SkipReason = &singleton
			rows[7].SkipReason = &paused
			tail := *template
			tail.RunID = ulid.Make().String()
			tail.Status = tc.want[0]
			if tail.Status == enums.RunStatusSkipped {
				tail.SkipReason = &paused
			}
			rows = append(rows, &tail)
			queries := make(chan cqrs.GetTraceRunOpt, 1)
			data := &replayData{events: []*cqrs.Event{evt}, runs: func(_ context.Context, query cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
				queries <- query
				return rows, nil
			}}
			scheduler := newReplayScheduler()
			provider := NewReplayProvider(data, replayFunctions{fn: fn}, scheduler, nil)
			opts := replayOpts(fn.Slug)
			opts.Statuses = tc.selectors
			result, err := provider.CreateReplay(t.Context(), opts)
			require.NoError(t, err)
			for i, status := range append(slices.Clone(tc.want), tail.Status) {
				call := receiveReplay(t, scheduler.calls)
				require.Equal(t, result.ID, *call.request.ReplayID)
				var original *cqrs.TraceRun
				for _, row := range rows {
					if row.RunID == call.request.OriginalRunID.String() {
						original = row
					}
				}
				require.NotNil(t, original)
				require.Equal(t, status, original.Status)
				if status == enums.RunStatusSkipped {
					require.Equal(t, &paused, original.SkipReason)
				}
				if i == len(tc.want) {
					require.Equal(t, tail.RunID, original.RunID)
				}
			}
			query := receiveReplay(t, queries)
			require.Equal(t, tc.want, query.Filter.Status)
			require.Equal(t, slices.Contains(tc.want, enums.RunStatusSkipped), query.IncludeSkipped)
		})
	}
}

func TestReplayPaginatesAnInclusiveSingleTimestamp(t *testing.T) {
	fn, template, evt := replayFixture()
	queries := make(chan cqrs.GetTraceRunOpt, 2)
	rows := make([]*cqrs.TraceRun, replayPageSize+1)
	for i := range rows {
		row := *template
		row.RunID = ulid.Make().String()
		row.Cursor = fmt.Sprintf("cursor-%d", i)
		rows[i] = &row
	}
	data := &replayData{events: []*cqrs.Event{evt}, runs: func(_ context.Context, query cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
		queries <- query
		if query.Cursor == "" {
			return rows[:replayPageSize], nil
		}
		return rows[replayPageSize:], nil
	}}
	scheduler := newReplayScheduler()
	provider := NewReplayProvider(data, replayFunctions{fn: fn}, scheduler, nil)
	opts := replayOpts(fn.Slug)
	opts.From = opts.Until
	result, err := provider.CreateReplay(t.Context(), opts)
	require.NoError(t, err)
	for _, row := range rows {
		call := receiveReplay(t, scheduler.calls)
		require.Equal(t, row.RunID, call.request.OriginalRunID.String())
		require.Equal(t, result.ID, *call.request.ReplayID)
	}
	first, second := receiveReplay(t, queries), receiveReplay(t, queries)
	require.Equal(t, uint(replayPageSize), first.Items)
	require.Empty(t, first.Cursor)
	require.Equal(t, rows[replayPageSize-1].Cursor, second.Cursor)
	require.Equal(t, opts.From, first.Filter.From)
	require.Equal(t, opts.From, first.Filter.Until)
	require.Equal(t, []cqrs.GetTraceRunOrder{{Field: enums.TraceRunTimeQueuedAt, Direction: enums.TraceRunOrderDesc}}, first.Order)
}

func TestReplayDetachesRequestAndBoundsFutureWindow(t *testing.T) {
	fn, run, evt := replayFixture()
	started := make(chan cqrs.GetTraceRunOpt, 1)
	proceed := make(chan struct{})
	data := &replayData{events: []*cqrs.Event{evt}, runs: func(_ context.Context, query cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
		started <- query
		<-proceed
		return []*cqrs.TraceRun{run}, nil
	}}
	scheduler := newReplayScheduler()
	provider := NewReplayProvider(data, replayFunctions{fn: fn}, scheduler, nil)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	opts := replayOpts(fn.Slug)
	opts.Until = time.Now().Add(time.Hour)
	result, err := provider.CreateReplay(ctx, opts)
	require.NoError(t, err)
	query := receiveReplay(t, started)
	cancel()
	close(proceed)
	call := receiveReplay(t, scheduler.calls)
	require.NoError(t, call.contextErr)
	require.Equal(t, result.CreatedAt, query.Filter.Until)
	require.Equal(t, result.ID, *call.request.ReplayID)
}

func TestReplayContinuesAfterMissingEventsAndScheduleFailure(t *testing.T) {
	fn, template, evt := replayFixture()
	missing, failed, success := *template, *template, *template
	missing.RunID, failed.RunID, success.RunID = ulid.Make().String(), ulid.Make().String(), ulid.Make().String()
	missing.TriggerIDs = []string{ulid.Make().String()}
	data := &replayData{events: []*cqrs.Event{evt}, runs: func(context.Context, cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
		return []*cqrs.TraceRun{&missing, &failed, &success}, nil
	}}
	scheduler := newReplayScheduler()
	scheduler.err = errors.New("schedule unavailable")
	provider := NewReplayProvider(data, replayFunctions{fn: fn}, scheduler, nil)
	result, err := provider.CreateReplay(t.Context(), replayOpts(fn.Slug))
	require.NoError(t, err)
	for _, run := range []*cqrs.TraceRun{&failed, &success} {
		call := receiveReplay(t, scheduler.calls)
		require.Equal(t, run.RunID, call.request.OriginalRunID.String())
		require.Equal(t, result.ID, *call.request.ReplayID)
	}
}

type replayFunctions struct {
	fn  inngest.DeployedFunction
	get func(context.Context, string) (inngest.DeployedFunction, error)
}

func (f replayFunctions) GetFunctionByApp(ctx context.Context, appID, id string) (inngest.DeployedFunction, error) {
	if f.get != nil {
		return f.get(ctx, id)
	}
	return f.fn, nil
}

type replayData struct {
	runs   func(context.Context, cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error)
	events []*cqrs.Event
}

func (d *replayData) GetRuns(ctx context.Context, query cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	return d.runs(ctx, query)
}

func (d *replayData) GetEventsByInternalIDs(context.Context, []ulid.ULID) ([]*cqrs.Event, error) {
	return d.events, nil
}

type replayCall struct {
	request    execution.ScheduleRequest
	contextErr error
}

type replayScheduler struct {
	calls chan replayCall
	err   error
}

func newReplayScheduler() *replayScheduler {
	return &replayScheduler{calls: make(chan replayCall, replayPageSize+2)}
}

func (s *replayScheduler) Schedule(ctx context.Context, req execution.ScheduleRequest) (*ulid.ULID, *sv2.Metadata, error) {
	s.calls <- replayCall{request: req, contextErr: ctx.Err()}
	id := ulid.Make()
	return &id, nil, s.err
}

func receiveReplay[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for replay")
		var zero T
		return zero
	}
}

func replayFixture() (inngest.DeployedFunction, *cqrs.TraceRun, *cqrs.Event) {
	fn := inngest.DeployedFunction{
		ID: uuid.New(), AppID: uuid.New(), AppName: "my-app", Slug: "my-app-function",
		AccountID: consts.DevServerAccountID, EnvironmentID: consts.DevServerEnvID,
	}
	fn.Function = inngest.Function{ID: fn.ID, Slug: fn.Slug, Name: "Replayable function"}
	evt := &cqrs.Event{ID: ulid.Make(), EventID: "first-event", EventName: "test/replay", EventData: map[string]any{"position": "first"}}
	run := &cqrs.TraceRun{
		RunID: ulid.Make().String(), FunctionID: fn.ID, AppID: fn.AppID,
		AccountID: fn.AccountID, WorkspaceID: fn.EnvironmentID,
		Status: enums.RunStatusCompleted, TriggerIDs: []string{evt.ID.String()},
	}
	return fn, run, evt
}

func replayOpts(id string) apiv2.CreateReplayOpts {
	return apiv2.CreateReplayOpts{
		Name: "Replay test", AppID: "my-app", FunctionID: id,
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Until: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Statuses: []enums.ReplayRunStatus{enums.ReplayRunStatusCompleted},
	}
}
