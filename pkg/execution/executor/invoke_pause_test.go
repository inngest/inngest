package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func invokePauseFixture() (state.Pause, event.InternalEvent) {
	runID := ulid.Make()
	stepID := "invoke-step.1"
	correlationID := runID.String() + "." + stepID
	p := state.Pause{
		ID:                  inngest.DeterministicSha1UUID(runID.String() + stepID),
		WorkspaceID:         uuid.New(),
		Identifier:          state.PauseIdentifier{AccountID: uuid.New(), FunctionID: uuid.New(), RunID: runID},
		DataKey:             stepID,
		Opcode:              new(enums.OpcodeInvokeFunction.String()),
		Event:               new(event.FnFinishedName),
		InvokeCorrelationID: &correlationID,
		Expires:             state.Time(time.Now().Add(time.Hour)),
	}
	evt := event.InternalEvent{
		ID: ulid.Make(), AccountID: p.Identifier.AccountID, WorkspaceID: p.WorkspaceID,
		Event: event.Event{Name: event.FnFinishedName, Data: map[string]any{
			consts.InvokeCorrelationId: correlationID, "run_id": ulid.Make().String(), "result": "child result",
		}},
	}
	return p, evt
}

type invokeLookupManager struct {
	pauses.Manager
	lookup  func() (*state.Pause, error)
	byID    func(pauses.Index, uuid.UUID) (*state.Pause, error)
	deletes int
}

func (m *invokeLookupManager) PauseByInvokeCorrelationID(context.Context, uuid.UUID, string) (*state.Pause, error) {
	return m.lookup()
}
func (m *invokeLookupManager) PauseByID(_ context.Context, index pauses.Index, id uuid.UUID) (*state.Pause, error) {
	return m.byID(index, id)
}
func (m *invokeLookupManager) Delete(context.Context, pauses.Index, state.Pause, ...state.DeletePauseOpt) error {
	m.deletes++
	return nil
}

func TestHandleInvokeFinishLookup(t *testing.T) {
	storeErr := errors.New("store unavailable")
	type testCase struct {
		name         string
		byIDErr      error
		indexErr     error
		change       func(*state.Pause, *event.InternalEvent)
		nilPause     bool
		wantErr      error
		wantMismatch bool
		wantByID     bool
		wantIndex    bool
		wantDelete   bool
	}
	tests := []testCase{
		{name: "direct lookup skips index", wantByID: true},
		{name: "missing index is not consulted", indexErr: state.ErrInvokePauseNotFound, wantByID: true},
		{name: "index store error is not consulted", indexErr: storeErr, wantByID: true},
		{name: "legacy random ID", byIDErr: state.ErrPauseNotFound, wantByID: true, wantIndex: true,
			change: func(p *state.Pause, _ *event.InternalEvent) { p.ID = uuid.New() }},
		{name: "expired beyond grace", wantByID: true, wantDelete: true,
			change: func(p *state.Pause, _ *event.InternalEvent) {
				p.Expires = state.Time(time.Now().Add(-2 * consts.PauseExpiredDeletionGracePeriod))
			}},
		{name: "direct store error", byIDErr: storeErr, wantErr: storeErr, wantByID: true},
		{name: "index store error", byIDErr: state.ErrPauseNotFound, indexErr: storeErr, wantErr: storeErr, wantByID: true, wantIndex: true},
		{name: "pause absent", byIDErr: state.ErrPauseNotFound, indexErr: state.ErrInvokePauseNotFound, wantErr: state.ErrInvokePauseNotFound, wantByID: true, wantIndex: true},
		{name: "indexed pause absent", byIDErr: state.ErrPauseNotFound, indexErr: state.ErrPauseNotFound, wantErr: state.ErrInvokePauseNotFound, wantByID: true, wantIndex: true},
		{name: "nil direct pause", nilPause: true, wantMismatch: true, wantByID: true},
		{name: "nil indexed pause", byIDErr: state.ErrPauseNotFound, nilPause: true, wantMismatch: true, wantByID: true, wantIndex: true},
		{name: "no correlation", wantErr: ErrNoCorrelationID, change: func(_ *state.Pause, evt *event.InternalEvent) { delete(evt.Event.Data, consts.InvokeCorrelationId) }},
		{name: "non-finished event", wantErr: ErrNoCorrelationID, change: func(_ *state.Pause, evt *event.InternalEvent) { evt.Event.Name = "other/event" }},
		{name: "wrong direct pause ID", wantMismatch: true, wantByID: true, change: func(p *state.Pause, _ *event.InternalEvent) { p.ID = uuid.New() }},
	}
	for _, malformed := range []string{"not-a-correlation", "invalid.step", ulid.Make().String() + ".", ulid.Make().String()} {
		tests = append(tests, testCase{name: "malformed " + malformed, wantErr: state.ErrInvokePauseNotFound,
			change: func(_ *state.Pause, evt *event.InternalEvent) { evt.Event.Data[consts.InvokeCorrelationId] = malformed }})
	}
	for name, change := range map[string]func(*state.Pause, *event.InternalEvent){
		"account":         func(p *state.Pause, _ *event.InternalEvent) { p.Identifier.AccountID = uuid.New() },
		"workspace":       func(p *state.Pause, _ *event.InternalEvent) { p.WorkspaceID = uuid.New() },
		"run":             func(p *state.Pause, _ *event.InternalEvent) { p.Identifier.RunID = ulid.Make() },
		"step":            func(p *state.Pause, _ *event.InternalEvent) { p.DataKey = "another-step" },
		"correlation":     func(p *state.Pause, _ *event.InternalEvent) { p.InvokeCorrelationID = new("another-correlation") },
		"nil correlation": func(p *state.Pause, _ *event.InternalEvent) { p.InvokeCorrelationID = nil },
		"opcode":          func(p *state.Pause, _ *event.InternalEvent) { p.Opcode = new(enums.OpcodeWaitForEvent.String()) },
		"nil opcode":      func(p *state.Pause, _ *event.InternalEvent) { p.Opcode = nil },
		"event":           func(p *state.Pause, _ *event.InternalEvent) { p.Event = new("another/event") },
		"nil event":       func(p *state.Pause, _ *event.InternalEvent) { p.Event = nil },
	} {
		tests = append(tests,
			testCase{name: "direct mismatch " + name, change: change, wantMismatch: true, wantByID: true},
			testCase{name: "indexed mismatch " + name, byIDErr: state.ErrPauseNotFound, change: change, wantMismatch: true, wantByID: true, wantIndex: true},
		)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, evt := invokePauseFixture()
			expectedID := p.ID
			p.Expires = state.Time(time.Now().Add(-time.Second))
			if tt.change != nil {
				tt.change(&p, &evt)
			}
			returnedPause := &p
			if tt.nilPause {
				returnedPause = nil
			}
			byIDCalls, indexCalls := 0, 0
			pm := &invokeLookupManager{
				lookup: func() (*state.Pause, error) {
					indexCalls++
					return returnedPause, tt.indexErr
				},
				byID: func(index pauses.Index, id uuid.UUID) (*state.Pause, error) {
					byIDCalls++
					require.Equal(t, pauses.Index{WorkspaceID: evt.WorkspaceID, EventName: event.FnFinishedName}, index)
					require.Equal(t, expectedID, id)
					return returnedPause, tt.byIDErr
				},
			}
			e := &executor{pm: pm, log: logger.From(context.Background())}
			err := e.HandleInvokeFinish(context.Background(), evt)
			switch {
			case tt.wantMismatch:
				require.ErrorContains(t, err, "identity mismatch")
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			default:
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantByID, byIDCalls == 1)
			require.Equal(t, tt.wantIndex, indexCalls == 1)
			require.Equal(t, tt.wantDelete, pm.deletes == 1)
		})
	}
}

func TestHandleInvokeFinishResumes(t *testing.T) {
	for _, tt := range []struct {
		name         string
		response     string
		mode         enums.ParallelMode
		coalesceKey  string
		missingIndex bool
		legacyID     bool
	}{
		{name: "result", response: "result"},
		{name: "error", response: "error"},
		{name: "legacy random ID", response: "result", legacyID: true},
		{name: "missing index result", response: "result", missingIndex: true},
		{name: "missing index error", response: "error", missingIndex: true},
		{name: "missing index coalesced", response: "result", mode: enums.ParallelModeWait, coalesceKey: "parallel-group", missingIndex: true},
		{name: "missing index race", response: "result", mode: enums.ParallelModeRace, coalesceKey: "parallel-group", missingIndex: true},
		{name: "coalesced", response: "result", mode: enums.ParallelModeWait, coalesceKey: "parallel-group"},
		{name: "race", response: "result", mode: enums.ParallelModeRace, coalesceKey: "parallel-group"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.response
			ctx := context.Background()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			unsharded := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
			pauseStore := redis_state.NewPauseStore(unsharded)
			mgr, err := redis_state.New(ctx, redis_state.WithShardedClient(redis_state.NewShardedClient(redis_state.ShardedClientOpts{
				UnshardedClient: unsharded, FunctionRunStateClient: rc, BatchClient: rc,
				StateDefaultKey: redis_state.StateDefaultKey, QueueDefaultKey: redis_state.QueueDefaultKey, FnRunIsSharded: redis_state.AlwaysShardOnRun,
			})), redis_state.WithPauseDeleter(pauseStore))
			require.NoError(t, err)
			rs := redis_state.MustRunServiceV2(mgr)
			pm := pauses.NewPauseStoreManager(unsharded)
			p, evt := invokePauseFixture()
			if tt.legacyID {
				p.ID = uuid.New()
			}
			p.ParallelMode = tt.mode
			p.ParallelCoalesceKey = tt.coalesceKey
			md := sv2.Metadata{
				ID:     sv2.ID{RunID: p.Identifier.RunID, FunctionID: p.Identifier.FunctionID, Tenant: sv2.Tenant{AccountID: p.Identifier.AccountID, EnvID: p.WorkspaceID, AppID: uuid.New()}},
				Config: *sv2.InitConfig(&sv2.Config{EventIDs: []ulid.ULID{ulid.Make()}, Idempotency: uuid.NewString()}),
			}
			_, err = rs.Create(ctx, sv2.CreateState{Metadata: md, Events: []json.RawMessage{json.RawMessage(`{"name":"test"}`)}})
			require.NoError(t, err)
			_, err = pauseStore.SavePause(ctx, p)
			require.NoError(t, err)
			indexKey := fmt.Sprintf("{%s}:invoke:%s", redis_state.StateDefaultKey, p.WorkspaceID)
			require.Equal(t, p.ID.String(), r.HGet(indexKey, *p.InvokeCorrelationID))
			if tt.missingIndex {
				r.HDel(indexKey, *p.InvokeCorrelationID)
				_, err = pm.PauseByInvokeCorrelationID(ctx, p.WorkspaceID, *p.InvokeCorrelationID)
				require.ErrorIs(t, err, state.ErrInvokePauseNotFound)
				_, err = pauseStore.SavePause(ctx, p)
				require.ErrorIs(t, err, state.ErrPauseAlreadyExists)
			}
			delete(evt.Event.Data, "result")
			evt.Event.Data[response] = map[string]any{"message": "child response"}
			q := &dedupeQueue{}
			e := &executor{pm: pm, smv2: rs, queue: q, log: logger.From(ctx), tracerProvider: tracing.NewNoopTracerProvider()}
			var wg sync.WaitGroup
			results := make(chan error, 4)
			for range 4 {
				wg.Go(func() { results <- e.HandleInvokeFinish(ctx, evt) })
			}
			wg.Wait()
			close(results)
			successes := 0
			for err := range results {
				if err == nil {
					successes++
				} else {
					require.ErrorIs(t, err, state.ErrInvokePauseNotFound)
				}
			}
			require.Positive(t, successes)
			steps, err := rs.LoadSteps(ctx, md.ID)
			require.NoError(t, err)
			key := "data"
			if response == "error" {
				key = "error"
			}
			require.JSONEq(t, fmt.Sprintf(`{"%s":{"message":"child response"}}`, key), string(steps[p.DataKey]))
			require.Len(t, q.enqueued, 1)
			expectedJobID := fmt.Sprintf("%s-%s-event", md.IdempotencyKey(), p.DataKey)
			if tt.coalesceKey != "" && tt.mode != enums.ParallelModeRace {
				expectedJobID = fmt.Sprintf("%s-%s-discover", md.ID.RunID, tt.coalesceKey)
			}
			require.Equal(t, expectedJobID, *q.enqueued[0].JobID)
			_, err = pm.PauseByID(ctx, pauses.Index{WorkspaceID: p.WorkspaceID, EventName: event.FnFinishedName}, p.ID)
			require.ErrorIs(t, err, state.ErrPauseNotFound)
			require.ErrorIs(t, e.HandleInvokeFinish(ctx, evt), state.ErrInvokePauseNotFound)
			require.Len(t, q.enqueued, 1)
		})
	}
}

type missingInvokeBlockStore struct {
	pauses.BlockStore
	lookups int
}

func (b *missingInvokeBlockStore) PauseByID(context.Context, pauses.Index, uuid.UUID) (*state.Pause, error) {
	b.lookups++
	return nil, state.ErrPauseNotFound
}

func TestHandleInvokeFinishBlockStorageReadError(t *testing.T) {
	storeErr := errors.New("buffer unavailable")
	for _, tt := range []struct {
		name             string
		readErr          error
		wantErr          error
		wantBlockLookups int
	}{
		{name: "read error stays retryable", readErr: storeErr, wantErr: storeErr},
		{name: "missing pause stays missing", readErr: state.ErrPauseNotFound, wantErr: state.ErrInvokePauseNotFound, wantBlockLookups: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, evt := invokePauseFixture()
			buffer := &invokeLookupManager{
				lookup: func() (*state.Pause, error) { return nil, state.ErrInvokePauseNotFound },
				byID: func(index pauses.Index, id uuid.UUID) (*state.Pause, error) {
					require.Equal(t, pauses.Index{WorkspaceID: p.WorkspaceID, EventName: event.FnFinishedName}, index)
					require.Equal(t, p.ID, id)
					return nil, tt.readErr
				},
			}
			blocks := &missingInvokeBlockStore{}
			pm := pauses.NewManager(buffer, blocks, pauses.WithBlockStoreEnabled(func(context.Context, uuid.UUID) bool { return true }))
			e := &executor{pm: pm, log: logger.From(context.Background())}
			err := e.HandleInvokeFinish(context.Background(), evt)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantBlockLookups, blocks.lookups)
		})
	}
}
