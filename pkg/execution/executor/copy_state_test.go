package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	redis_state "github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestCopyRunState(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})

	pauseStore := redis_state.NewPauseStore(unshardedClient)

	sm, err := redis_state.New(
		ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseStore),
	)
	require.NoError(t, err)

	smv2 := redis_state.MustRunServiceV2(sm)

	acctID := uuid.New()
	wsID := uuid.New()
	appID := uuid.New()
	fnID := uuid.New()

	// ----------------------------------------------------------------
	// Step 1: Create a source run with completed steps via SaveResponse
	// ----------------------------------------------------------------
	sourceRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	sourceID := state.Identifier{
		AccountID:   acctID,
		WorkspaceID: wsID,
		AppID:       appID,
		WorkflowID:  fnID,
		RunID:       sourceRunID,
		Key:         sourceRunID.String(),
	}

	_, err = sm.New(ctx, state.Input{
		Identifier:     sourceID,
		EventBatchData: []map[string]any{{"name": "test/source.event", "data": map[string]any{}}},
	})
	require.NoError(t, err)

	// Simulate completed steps by saving responses (this is what happens
	// during normal execution when steps complete).
	sourceSteps := map[string]string{
		"step-a": `{"data":"result-a"}`,
		"step-b": `{"data":"result-b"}`,
	}
	for stepID, output := range sourceSteps {
		_, err = sm.SaveResponse(ctx, sourceID, stepID, output)
		require.NoError(t, err)
	}

	// Verify source run state looks right before we copy it.
	t.Run("source run has steps and stack", func(t *testing.T) {
		stack, err := smv2.LoadStack(ctx, sv2.ID{
			RunID:      sourceRunID,
			FunctionID: fnID,
			Tenant:     sv2.Tenant{AccountID: acctID, EnvID: wsID, AppID: appID},
		})
		require.NoError(t, err)
		require.Len(t, stack, 2, "source run should have 2 entries in stack")
		t.Logf("source stack: %v", stack)

		steps, err := smv2.LoadSteps(ctx, sv2.ID{
			RunID:      sourceRunID,
			FunctionID: fnID,
			Tenant:     sv2.Tenant{AccountID: acctID, EnvID: wsID, AppID: appID},
		})
		require.NoError(t, err)
		require.Len(t, steps, 2, "source run should have 2 step entries")
		t.Logf("source steps keys: %v", mapKeys(steps))
	})

	// ----------------------------------------------------------------
	// Step 2: Call copyRunState and inspect the result
	// ----------------------------------------------------------------
	t.Run("copyRunState populates steps and embeds original events", func(t *testing.T) {
		deferredEvt := `{"name":"inngest/deferred.start","data":{"fnSlug":"my-fn","runId":"abc"}}`
		newState := sv2.CreateState{
			Steps:  []state.MemoizedStep{},
			Events: []json.RawMessage{json.RawMessage(deferredEvt)},
		}

		req := execution.ScheduleRequest{
			AccountID:     acctID,
			WorkspaceID:   wsID,
			AppID:         appID,
			CopyStateFrom: &sourceRunID,
		}

		err := copyRunState(ctx, smv2, req, &newState)
		require.NoError(t, err)

		// Verify steps were copied.
		require.NotEmpty(t, newState.Steps, "copyRunState should populate Steps")
		stepsByID := map[string]state.MemoizedStep{}
		for _, s := range newState.Steps {
			stepsByID[s.ID] = s
		}
		require.Contains(t, stepsByID, "step-a")
		require.Contains(t, stepsByID, "step-b")

		// Verify original events are embedded in the deferred.start event's data.
		var updatedEvt map[string]any
		err = json.Unmarshal(newState.Events[0], &updatedEvt)
		require.NoError(t, err)

		data := updatedEvt["data"].(map[string]any)
		require.Equal(t, "my-fn", data["fnSlug"], "original data fields preserved")
		require.Equal(t, "abc", data["runId"], "original data fields preserved")

		// data.event should be the source run's first event
		origEvt, ok := data["event"].(map[string]any)
		require.True(t, ok, "data.event should be present")
		require.Equal(t, "test/source.event", origEvt["name"])
		t.Logf("embedded event: %v", origEvt)

		// data.events should be an array of all source events
		origEvents, ok := data["events"].([]any)
		require.True(t, ok, "data.events should be present")
		require.Len(t, origEvents, 1)
		first := origEvents[0].(map[string]any)
		require.Equal(t, "test/source.event", first["name"])
	})

	// ----------------------------------------------------------------
	// Step 3: Create a NEW run with the copied steps and verify they
	//         survive the full Create → LoadMetadata → LoadSteps round-trip
	// ----------------------------------------------------------------
	t.Run("new run has memoized steps after Create", func(t *testing.T) {
		deferredEvt := `{"name":"inngest/deferred.start","data":{"fnSlug":"my-fn","runId":"abc"}}`
		newState := sv2.CreateState{
			Steps:  []state.MemoizedStep{},
			Events: []json.RawMessage{json.RawMessage(deferredEvt)},
		}

		req := execution.ScheduleRequest{
			AccountID:     acctID,
			WorkspaceID:   wsID,
			AppID:         appID,
			CopyStateFrom: &sourceRunID,
		}

		err := copyRunState(ctx, smv2, req, &newState)
		require.NoError(t, err)
		require.NotEmpty(t, newState.Steps)

		newRunID := ulid.MustNew(ulid.Now(), rand.Reader)
		newFnID := uuid.New() // could be the same or different function

		newState.Metadata = sv2.Metadata{
			ID: sv2.ID{
				RunID:      newRunID,
				FunctionID: newFnID,
				Tenant:     sv2.Tenant{AccountID: acctID, EnvID: wsID, AppID: appID},
			},
			Config: *sv2.InitConfig(&sv2.Config{
				FunctionVersion: 1,
				EventIDs:        []ulid.ULID{ulid.MustNew(ulid.Now(), rand.Reader)},
			}),
		}

		created, err := smv2.Create(ctx, newState)
		require.NoError(t, err)

		// Load the new run's metadata and verify the stack.
		newID := sv2.ID{
			RunID:      newRunID,
			FunctionID: newFnID,
			Tenant:     sv2.Tenant{AccountID: acctID, EnvID: wsID, AppID: appID},
		}

		md, err := smv2.LoadMetadata(ctx, newID)
		require.NoError(t, err)
		require.NotEmpty(t, md.Stack, "new run metadata should have a non-empty stack")
		t.Logf("new run stack: %v", md.Stack)

		// Load the new run's steps.
		steps, err := smv2.LoadSteps(ctx, newID)
		require.NoError(t, err)
		require.NotEmpty(t, steps, "new run should have steps loaded")
		t.Logf("new run steps keys: %v", mapKeys(steps))

		// Verify the step data round-tripped correctly.
		for _, stepID := range []string{"step-a", "step-b"} {
			raw, ok := steps[stepID]
			require.True(t, ok, "step %s should exist in new run", stepID)

			var data map[string]any
			err := json.Unmarshal(raw, &data)
			require.NoError(t, err)
			t.Logf("  step %s = %v", stepID, data)
		}

		// Verify stored events contain the embedded original events.
		storedEvents, err := smv2.LoadEvents(ctx, newID)
		require.NoError(t, err)
		require.Len(t, storedEvents, 1)

		var storedEvt map[string]any
		err = json.Unmarshal(storedEvents[0], &storedEvt)
		require.NoError(t, err)
		require.Equal(t, "inngest/deferred.start", storedEvt["name"])

		evtData := storedEvt["data"].(map[string]any)
		origEvt := evtData["event"].(map[string]any)
		require.Equal(t, "test/source.event", origEvt["name"],
			"stored event should contain embedded original event")
		origEvents := evtData["events"].([]any)
		require.Len(t, origEvents, 1)

		_ = created
	})
}

// TestCopyRunState_IdempotencyRace simulates the scenario where the same event
// triggers BOTH the deferred handler (with CopyStateFrom) and the normal trigger
// path (without CopyStateFrom). If they use the same idempotency key, the first
// call to Create wins. This test proves that if the normal path creates the run
// first, the copied steps are lost.
func TestCopyRunState_IdempotencyRace(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})

	pauseStore := redis_state.NewPauseStore(unshardedClient)

	sm, err := redis_state.New(
		ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseStore),
	)
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	acctID := uuid.New()
	wsID := uuid.New()
	appID := uuid.New()
	fnID := uuid.New()
	newFnID := uuid.New()

	// Create source run with steps
	sourceRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	sourceID := state.Identifier{
		AccountID: acctID, WorkspaceID: wsID, AppID: appID,
		WorkflowID: fnID, RunID: sourceRunID, Key: sourceRunID.String(),
	}
	_, err = sm.New(ctx, state.Input{
		Identifier:     sourceID,
		EventBatchData: []map[string]any{{"name": "test", "data": map[string]any{}}},
	})
	require.NoError(t, err)
	_, err = sm.SaveResponse(ctx, sourceID, "step-a", `{"data":"result-a"}`)
	require.NoError(t, err)

	// Simulate: normal trigger path creates the run FIRST (no CopyStateFrom)
	newRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	normalState := sv2.CreateState{
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID: newRunID, FunctionID: newFnID,
				Tenant: sv2.Tenant{AccountID: acctID, EnvID: wsID, AppID: appID},
			},
			Config: *sv2.InitConfig(&sv2.Config{
				FunctionVersion: 1,
				Idempotency:     "same-key",
				EventIDs:        []ulid.ULID{ulid.MustNew(ulid.Now(), rand.Reader)},
			}),
		},
		Events: []json.RawMessage{json.RawMessage(`{"name":"inngest/deferred.start","data":{}}`)},
		Steps:  []state.MemoizedStep{}, // NO steps - this is the normal path
	}
	_, err = smv2.Create(ctx, normalState)
	require.NoError(t, err)

	// Now simulate: deferred path tries to create the same run (with CopyStateFrom)
	deferredState := sv2.CreateState{Steps: []state.MemoizedStep{}}
	req := execution.ScheduleRequest{
		AccountID: acctID, WorkspaceID: wsID, AppID: appID,
		CopyStateFrom: &sourceRunID,
	}
	err = copyRunState(ctx, smv2, req, &deferredState)
	require.NoError(t, err)
	require.NotEmpty(t, deferredState.Steps, "copyRunState should have found steps")

	deferredState.Metadata = sv2.Metadata{
		ID: sv2.ID{
			RunID: newRunID, FunctionID: newFnID,
			Tenant: sv2.Tenant{AccountID: acctID, EnvID: wsID, AppID: appID},
		},
		Config: *sv2.InitConfig(&sv2.Config{
			FunctionVersion: 1,
			Idempotency:     "same-key", // same key!
			EventIDs:        []ulid.ULID{ulid.MustNew(ulid.Now(), rand.Reader)},
		}),
	}
	deferredState.Events = []json.RawMessage{json.RawMessage(`{"name":"inngest/deferred.start","data":{}}`)}
	_, err = smv2.Create(ctx, deferredState)
	// This will return ErrIdentifierExists because the normal path already created the state
	require.ErrorIs(t, err, state.ErrIdentifierExists, "second Create with same idempotency key should conflict")

	// Now verify: the actual run has NO steps (the normal path won the race)
	newID := sv2.ID{
		RunID: newRunID, FunctionID: newFnID,
		Tenant: sv2.Tenant{AccountID: acctID, EnvID: wsID, AppID: appID},
	}
	md, err := smv2.LoadMetadata(ctx, newID)
	require.NoError(t, err)
	t.Logf("stack after race: %v (empty = normal path won)", md.Stack)
	require.Empty(t, md.Stack, "normal path wins → no copied steps (this is the bug)")
}

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
