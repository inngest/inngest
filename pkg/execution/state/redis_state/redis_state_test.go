package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/testharness"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunMetadata(t *testing.T) {
	byt, _ := json.Marshal(state.Identifier{})

	tests := []struct {
		name          string
		data          map[string]string
		expectedVal   *runMetadata
		expectedError error
	}{
		{
			name: "should return value if data is valid",
			data: map[string]string{
				"status":   "1",
				"version":  "1",
				"debugger": "false",
				"id":       string(byt),
			},
			expectedVal: &runMetadata{
				Status:   enums.RunStatusCompleted,
				Version:  1,
				Debugger: false,
			},
			expectedError: nil,
		},
		{
			name:          "should error with missing identifier",
			data:          map[string]string{},
			expectedError: state.ErrRunNotFound,
		},
		{
			name: "missing ID should return err run not found",
			data: map[string]string{
				"status": "hello",
			},
			expectedError: state.ErrRunNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runMeta, err := newRunMetadata(test.data)
			require.Equal(t, test.expectedError, err)
			require.Equal(t, test.expectedVal, runMeta)
		})
	}
}

func TestIdempotencyCheck(t *testing.T) {
	ctx := context.Background()

	r, rc := initRedis(t)
	defer rc.Close()

	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})

	acctID, wsID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	runState := shardedClient.fnRunState
	ftc, shared := runState.Client(ctx, acctID, runID)
	require.True(t, shared)

	mgr := shardedMgr{s: shardedClient}

	t.Run("with idempotency key defined in identifier", func(t *testing.T) {
		id := state.Identifier{
			AccountID:   acctID,
			WorkspaceID: wsID,
			AppID:       appID,
			WorkflowID:  fnID,
			RunID:       runID,
			Key:         "yolo",
		}
		key := runState.kg.Idempotency(ctx, shared, id)

		t.Run("returns nil if no idempotency key is available", func(t *testing.T) {
			r.FlushAll()

			runID, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.NoError(t, err)
			require.Nil(t, runID)
		})

		t.Run("returns state if idempotency is already there", func(t *testing.T) {
			r.FlushAll()

			created, err := mgr.New(ctx, state.Input{
				Identifier:     id,
				EventBatchData: []map[string]any{},
				Steps:          []state.MemoizedStep{},
				StepInputs:     []state.MemoizedStep{},
				Context: map[string]any{
					"hello": "world",
				},
			})
			require.NoError(t, err)

			runID, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.NoError(t, err)
			require.NotNil(t, runID)

			require.Equal(t, created.Identifier().RunID, *runID)
		})

		t.Run("returns invalid identifier error if previous value is not a ULID", func(t *testing.T) {
			r.FlushAll()
			require.NoError(t, r.Set(key, ""))

			runID, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.Nil(t, runID)
			require.ErrorIs(t, err, state.ErrInvalidIdentifier)
		})
	})

	t.Run("with idempotency key not defined in identifier", func(t *testing.T) {
		id := state.Identifier{
			AccountID:   acctID,
			WorkspaceID: wsID,
			AppID:       appID,
			WorkflowID:  fnID,
			RunID:       runID,
		}
		key := runState.kg.Idempotency(ctx, shared, id)

		t.Run("returns nil if no idempotency key is available", func(t *testing.T) {
			r.FlushAll()

			st, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.NoError(t, err)
			require.Nil(t, st)
		})

		t.Run("returns nil if runID is different", func(t *testing.T) {
			r.FlushAll()

			_, err := mgr.New(ctx, state.Input{
				Identifier:     id,
				EventBatchData: []map[string]any{},
			})
			require.NoError(t, err)

			diffID := id // copy
			diffID.RunID = ulid.MustNew(ulid.Now(), rand.Reader)
			diffKey := runState.kg.Idempotency(ctx, shared, diffID)
			runID, err := mgr.idempotencyCheck(ctx, ftc, diffKey, diffID)
			require.NoError(t, err)
			require.Nil(t, runID)
		})

		t.Run("returns invalid identifier error if previous value is not a ULID", func(t *testing.T) {
			r.FlushAll()
			require.NoError(t, r.Set(key, ""))

			runID, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.Nil(t, runID)
			require.ErrorIs(t, err, state.ErrInvalidIdentifier)
		})
	})
}

func TestStateHarness(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})

	sm, err := New(
		context.Background(),
		WithUnshardedClient(unshardedClient),
		WithShardedClient(shardedClient),
	)
	require.NoError(t, err)

	create := func() (state.Manager, func()) {
		return sm, func() {
			r.FlushAll()
		}
	}

	testharness.CheckState(t, create)
}

func TestScanIter(t *testing.T) {
	ctx := context.Background()
	redis := miniredis.RunT(t)
	r, err := rueidis.NewClient(rueidis.ClientOption{
		// InitAddress: []string{"127.0.0.1:6379"},
		InitAddress:  []string{redis.Addr()},
		Password:     "",
		DisableCache: true,
	})
	require.NoError(t, err)
	defer r.Close()

	entries := 50_000
	key := "test-scan"
	for i := 0; i < entries; i++ {
		cmd := r.B().Hset().Key(key).FieldValue().
			FieldValue(strconv.Itoa(i), strconv.Itoa(i)).
			Build()
		require.NoError(t, r.Do(ctx, cmd).Error())
	}
	fmt.Println("loaded keys")

	// Create a new scanIter, which iterates through all items.
	si := &scanIter{r: r}
	err = si.init(ctx, key, 1000)
	require.NoError(t, err)

	listed := 0

	for n := 0; n < entries*2; n++ {
		if !si.Next(ctx) {
			break
		}
		listed++
	}

	require.Equal(t, context.Canceled, si.Error())
	require.Equal(t, entries, listed)
}

func TestBufIter(t *testing.T) {
	ctx := context.Background()
	redis := miniredis.RunT(t)
	r, err := rueidis.NewClient(rueidis.ClientOption{
		// InitAddress: []string{"127.0.0.1:6379"},
		InitAddress:  []string{redis.Addr()},
		Password:     "",
		DisableCache: true,
	})
	require.NoError(t, err)
	defer r.Close()

	entries := 10_000
	key := "test-bufiter"
	for i := 0; i < entries; i++ {
		cmd := r.B().Hset().Key(key).FieldValue().FieldValue(strconv.Itoa(i), "{}").Build()
		require.NoError(t, r.Do(ctx, cmd).Error())
	}
	fmt.Println("loaded keys")

	// Create a new scanIter, which iterates through all items.
	bi := &bufIter{r: r}
	err = bi.init(ctx, key)
	require.NoError(t, err)

	listed := 0
	for n := 0; n < entries*2; n++ {
		if !bi.Next(ctx) {
			break
		}
		listed++
	}

	require.Equal(t, context.Canceled, bi.Error())
	require.Equal(t, entries, listed)
}

func TestLoadStackStepInputsStepsWithIDs(t *testing.T) {
	ctx := context.Background()

	_, rc := initRedis(t)
	defer rc.Close()

	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})

	sm, err := New(
		ctx,
		WithUnshardedClient(unshardedClient),
		WithShardedClient(shardedClient),
	)
	require.NoError(t, err)

	mgr := sm.(*mgr)

	acctID, wsID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	// Create a state with steps
	id := state.Identifier{
		AccountID:   acctID,
		WorkspaceID: wsID,
		AppID:       appID,
		WorkflowID:  fnID,
		RunID:       runID,
	}

	steps := []state.MemoizedStep{
		{
			ID:   "step-1",
			Data: map[string]any{"input": "step1_input"},
		},
		{
			ID:   "step-2",
			Data: map[string]any{"input": "step2_input"},
		},
		{
			ID:   "step-3",
			Data: map[string]any{"input": "step3_input"},
		},
	}

	stepInputs := []state.MemoizedStep{
		{
			ID:   "step-1",
			Data: map[string]any{"input": "step1_input"},
		},
		{
			ID:   "step-2",
			Data: map[string]any{"input": "step2_input"},
		},
		{
			ID:   "step-3",
			Data: map[string]any{"input": "step3_input"},
		},
	}

	init := state.Input{
		Identifier:     id,
		EventBatchData: []map[string]any{{"test": "event"}},
		Steps:          steps,
		StepInputs:     stepInputs,
	}

	createdState, err := mgr.New(ctx, init)
	require.NoError(t, err)

	// Save some step results so LoadStepsWithIDs has something to return
	// Use different step IDs to avoid duplicate response errors
	_, err = mgr.shardedMgr.SaveResponse(ctx, id, "step-result-1", `{"result": "step1_result"}`)
	require.NoError(t, err)
	_, err = mgr.shardedMgr.SaveResponse(ctx, id, "step-result-3", `{"result": "step3_result"}`)
	require.NoError(t, err)

	t.Run("LoadStack works", func(t *testing.T) {
		stack, err := mgr.shardedMgr.stack(ctx, acctID, runID)
		require.NoError(t, err)
		assert.NotNil(t, stack)
	})

	t.Run("LoadStepInputs returns only step inputs", func(t *testing.T) {
		stepInputsResult, err := mgr.shardedMgr.LoadStepInputs(ctx, acctID, fnID, runID)
		require.NoError(t, err)
		assert.Equal(t, 3, len(stepInputsResult))
		assert.Contains(t, stepInputsResult, "step-1")
		assert.Contains(t, stepInputsResult, "step-2")
		assert.Contains(t, stepInputsResult, "step-3")

		// Verify that step inputs are returned directly (not wrapped)
		for stepID, stepData := range stepInputsResult {
			var data map[string]any
			err := json.Unmarshal(stepData, &data)
			require.NoError(t, err)
			assert.NotNil(t, data, "Step %s should contain data", stepID)
		}
	})

	t.Run("LoadStepsWithIDs returns specific steps", func(t *testing.T) {
		requestedSteps := []string{"step-result-1", "step-result-3"}
		stepsResult, err := mgr.shardedMgr.LoadStepsWithIDs(ctx, acctID, fnID, runID, requestedSteps)
		require.NoError(t, err)
		assert.Equal(t, 2, len(stepsResult))
		assert.Contains(t, stepsResult, "step-result-1")
		assert.Contains(t, stepsResult, "step-result-3")
		assert.NotContains(t, stepsResult, "step-2")
	})

	t.Run("LoadStepsWithIDs with empty slice returns empty map", func(t *testing.T) {
		requestedSteps := []string{}
		stepsResult, err := mgr.shardedMgr.LoadStepsWithIDs(ctx, acctID, fnID, runID, requestedSteps)
		require.NoError(t, err)
		assert.Equal(t, 0, len(stepsResult))
	})

	t.Run("LoadStepsWithIDs with non-existent steps returns partial results", func(t *testing.T) {
		requestedSteps := []string{"step-result-1", "non-existent-step"}
		stepsResult, err := mgr.shardedMgr.LoadStepsWithIDs(ctx, acctID, fnID, runID, requestedSteps)
		require.NoError(t, err)
		assert.Equal(t, 1, len(stepsResult))
		assert.Contains(t, stepsResult, "step-result-1")
		assert.NotContains(t, stepsResult, "non-existent-step")
	})

	// Test non-existent run ID
	nonExistentRunID := ulid.MustNew(ulid.Now(), rand.Reader)

	t.Run("LoadStepInputs with non-existent run returns empty", func(t *testing.T) {
		stepInputsResult, err := mgr.shardedMgr.LoadStepInputs(ctx, acctID, fnID, nonExistentRunID)
		require.NoError(t, err)
		assert.Equal(t, 0, len(stepInputsResult))
	})

	t.Run("LoadStepsWithIDs with non-existent run returns empty", func(t *testing.T) {
		requestedSteps := []string{"step-1", "step-2"}
		stepsResult, err := mgr.shardedMgr.LoadStepsWithIDs(ctx, acctID, fnID, nonExistentRunID, requestedSteps)
		require.NoError(t, err)
		assert.Equal(t, 0, len(stepsResult))
	})

	// Clean up
	_, err = mgr.Delete(ctx, createdState.Identifier())
	require.NoError(t, err)
}

func TestPauseCreatedAt(t *testing.T) {
	// Setup miniredis
	r, rc := initRedis(t)
	defer rc.Close()

	// Create Redis state manager
	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})

	mgr, err := New(
		context.Background(),
		WithUnshardedClient(unshardedClient),
		WithShardedClient(shardedClient),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Create test data
	workspaceID := uuid.New()
	eventName := "test.event"
	pauseID := uuid.New()
	runID := ulid.Make()
	
	// Create a pause with our test data
	pause := state.Pause{
		ID:          pauseID,
		WorkspaceID: workspaceID,
		Identifier: state.PauseIdentifier{
			RunID:      runID,
			FunctionID: uuid.New(),
			AccountID:  uuid.New(),
		},
		Event:   &eventName,
		Expires: state.Time(time.Now().Add(time.Hour)),
	}

	// Save the pause first
	_, err = mgr.SavePause(ctx, pause)
	require.NoError(t, err)

	// Now test PauseCreatedAt - this should work with our ZMSCORE fix
	createdAt, err := mgr.PauseCreatedAt(ctx, workspaceID, eventName, pauseID)
	require.NoError(t, err)
	require.False(t, createdAt.IsZero(), "created at timestamp should not be zero")
	
	// The timestamp should be reasonably recent (within the last minute)
	require.True(t, time.Since(createdAt) < time.Minute, "timestamp should be recent")

	// Test with non-existent pause
	nonExistentPauseID := uuid.New()
	_, err = mgr.PauseCreatedAt(ctx, workspaceID, eventName, nonExistentPauseID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "pause timestamp not found")

	// Clean up
	r.FlushAll()
}

func BenchmarkNew(b *testing.B) {
	r := miniredis.RunT(b)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(b, err)
	defer rc.Close()

	statePrefix := "state"
	unshardedClient := NewUnshardedClient(rc, statePrefix, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        statePrefix,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})

	sm, err := New(
		context.Background(),
		WithUnshardedClient(unshardedClient),
		WithShardedClient(shardedClient),
	)
	require.NoError(b, err)

	id := state.Identifier{
		WorkflowID: uuid.New(),
	}
	evt := event.Event{
		Name: "test-event",
		Data: map[string]any{
			"title": "They don't think it be like it is, but it do",
			"data": map[string]any{
				"float": 3.14132,
			},
		},
		User: map[string]any{
			"external_id": "1",
		},
		Version: "1985-01-01",
	}.Map()
	init := state.Input{
		Identifier:     id,
		EventBatchData: []map[string]any{evt},
	}

	ctx := context.Background()
	for n := 0; n < b.N; n++ {
		init.Identifier.RunID = ulid.MustNew(ulid.Now(), rand.Reader)
		_, err := sm.New(ctx, init)
		require.NoError(b, err)
	}

}
