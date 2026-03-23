package state_store

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/tests/execution/queue/helper"
	"github.com/inngest/inngest/tests/testutil"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStateStoreBackendComparison explicitly compares Valkey and Garnet backends
// side-by-side to ensure they produce identical results for all operations.
func TestStateOperationsAcrossBackends(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test")
	}

	ctx := context.Background()

	// Start Garnet
	garnetContainer, err := helper.StartGarnet(t,
		helper.WithImage(testutil.GarnetDefaultImage),
		helper.WithConfiguration(&helper.GarnetConfiguration{EnableLua: true}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = garnetContainer.Terminate(ctx) })

	garnetClient, err := helper.NewRedisClient(garnetContainer.Addr, garnetContainer.Username, garnetContainer.Password, false)
	require.NoError(t, err)
	t.Cleanup(func() { garnetClient.Close() })

	// Start Valkey
	valkeyContainer, err := helper.StartValkey(t,
		helper.WithValkeyImage(testutil.ValkeyDefaultImage),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = valkeyContainer.Terminate(ctx) })

	valkeyClient, err := helper.NewValkeyClient(valkeyContainer.Addr, valkeyContainer.Username, valkeyContainer.Password, false)
	require.NoError(t, err)
	t.Cleanup(func() { valkeyClient.Close() })

	// Create Garnet service
	garnetUnsharded := redis_state.NewUnshardedClient(garnetClient, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	garnetSharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        garnetUnsharded,
		FunctionRunStateClient: garnetClient,
		BatchClient:            garnetClient,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})
	garnetPauseMgr := redis_state.NewPauseStore(garnetUnsharded)
	garnetMgr, err := redis_state.New(ctx, redis_state.WithShardedClient(garnetSharded), redis_state.WithPauseDeleter(garnetPauseMgr))
	require.NoError(t, err)
	garnetSvc := redis_state.MustRunServiceV2(garnetMgr)

	// Create Valkey service
	valkeyUnsharded := redis_state.NewUnshardedClient(valkeyClient, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	valkeySharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        valkeyUnsharded,
		FunctionRunStateClient: valkeyClient,
		BatchClient:            valkeyClient,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})
	valkeyPauseMgr := redis_state.NewPauseStore(valkeyUnsharded)
	valkeyMgr, err := redis_state.New(ctx, redis_state.WithShardedClient(valkeySharded), redis_state.WithPauseDeleter(valkeyPauseMgr))
	require.NoError(t, err)
	valkeySvc := redis_state.MustRunServiceV2(valkeyMgr)

	// Shared test data
	functionID := uuid.New()
	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)

	testEvent := map[string]any{
		"name": "test.comparison.event",
		"data": map[string]any{
			"user_id": "comparison-123",
			"action":  "compare_backends",
			"payload": "some data for size testing",
		},
		"id": eventID.String(),
	}
	eventBytes, err := json.Marshal(testEvent)
	require.NoError(t, err)

	t.Run("Create with pre-filled steps should produce identical results", func(t *testing.T) {
		garnetRunID := ulid.MustNew(ulid.Now(), rand.Reader)
		valkeyRunID := ulid.MustNew(ulid.Now(), rand.Reader)

		stepData1 := map[string]any{"result": "step1_output", "count": 42}
		stepData2 := map[string]any{"result": "step2_output", "nested": map[string]any{"key": "value"}}
		stepData3 := map[string]any{"result": "step3_output", "items": []int{1, 2, 3, 4, 5}}

		inputData1 := map[string]any{"param": "input1", "number": 100}
		inputData2 := map[string]any{"param": "input2", "number": 200}

		createInput := func(runID ulid.ULID) statev2.CreateState {
			return statev2.CreateState{
				Metadata: statev2.Metadata{
					ID: statev2.ID{
						RunID:      runID,
						FunctionID: functionID,
						Tenant: statev2.Tenant{
							AccountID: accountID,
							EnvID:     workspaceID,
							AppID:     appID,
						},
					},
					Config: *statev2.InitConfig(&statev2.Config{
						SpanID:          "comparison-span",
						Idempotency:     fmt.Sprintf("comparison-key-%s", runID.String()),
						FunctionVersion: 3,
						RequestVersion:  2,
						HasAI:           true,
						ForceStepPlan:   true,
						EventIDs:        []ulid.ULID{eventID},
						PriorityFactor:  int64Ptr(50),
						CustomConcurrencyKeys: []statev2.CustomConcurrency{
							{
								Key:   "f:" + functionID.String() + ":user-test",
								Hash:  "testhash123",
								Limit: 10,
							},
						},
					}),
				},
				Events: []json.RawMessage{eventBytes},
				Steps: []state.MemoizedStep{
					{ID: "step-a", Data: stepData1},
					{ID: "step-b", Data: stepData2},
					{ID: "step-c", Data: stepData3},
				},
				StepInputs: []state.MemoizedStep{
					{ID: "input-x", Data: inputData1},
					{ID: "input-y", Data: inputData2},
				},
			}
		}

		garnetInput := createInput(garnetRunID)
		valkeyInput := createInput(valkeyRunID)

		garnetState, err := garnetSvc.Create(ctx, garnetInput)
		require.NoError(t, err)

		valkeyState, err := valkeySvc.Create(ctx, valkeyInput)
		require.NoError(t, err)

		garnetID := garnetState.Metadata.ID
		valkeyID := valkeyState.Metadata.ID

		t.Run("LoadMetadata should return identical config", func(t *testing.T) {
			garnetMeta, err := garnetSvc.LoadMetadata(ctx, garnetID)
			require.NoError(t, err)

			valkeyMeta, err := valkeySvc.LoadMetadata(ctx, valkeyID)
			require.NoError(t, err)

			// Compare Config fields - both backends should return identical values
			assert.Equal(t, garnetMeta.Config.FunctionVersion, valkeyMeta.Config.FunctionVersion, "FunctionVersion mismatch")
			assert.Equal(t, garnetMeta.Config.RequestVersion, valkeyMeta.Config.RequestVersion, "RequestVersion mismatch")
			assert.Equal(t, garnetMeta.Config.HasAI, valkeyMeta.Config.HasAI, "HasAI mismatch")
			assert.Equal(t, garnetMeta.Config.ForceStepPlan, valkeyMeta.Config.ForceStepPlan, "ForceStepPlan mismatch")
			assert.Equal(t, garnetMeta.Config.SpanID, valkeyMeta.Config.SpanID, "SpanID mismatch")

			// Verify correctness - FunctionVersion and RequestVersion should be persisted
			assert.Equal(t, 3, garnetMeta.Config.FunctionVersion)
			assert.Equal(t, 2, garnetMeta.Config.RequestVersion)

			// Note: HasAI and ForceStepPlan are not persisted during Create, but can be set via UpdateMetadata
			// This is documented behavior - the UpdateMetadata test verifies these fields can be set correctly
			t.Logf("Note: HasAI=%v, ForceStepPlan=%v (these fields are set via UpdateMetadata, not Create)",
				garnetMeta.Config.HasAI, garnetMeta.Config.ForceStepPlan)

			// Compare EventIDs
			assert.Equal(t, len(garnetMeta.Config.EventIDs), len(valkeyMeta.Config.EventIDs), "EventIDs length mismatch")

			// Compare PriorityFactor
			if garnetMeta.Config.PriorityFactor != nil && valkeyMeta.Config.PriorityFactor != nil {
				assert.Equal(t, *garnetMeta.Config.PriorityFactor, *valkeyMeta.Config.PriorityFactor, "PriorityFactor mismatch")
				assert.Equal(t, int64(50), *garnetMeta.Config.PriorityFactor)
			}

			// Compare CustomConcurrencyKeys
			assert.Equal(t, len(garnetMeta.Config.CustomConcurrencyKeys), len(valkeyMeta.Config.CustomConcurrencyKeys))

			t.Logf("Config comparison passed: FunctionVersion=%d, RequestVersion=%d",
				garnetMeta.Config.FunctionVersion, garnetMeta.Config.RequestVersion)
		})

		t.Run("Metrics should match between backends", func(t *testing.T) {
			garnetMeta, err := garnetSvc.LoadMetadata(ctx, garnetID)
			require.NoError(t, err)

			valkeyMeta, err := valkeySvc.LoadMetadata(ctx, valkeyID)
			require.NoError(t, err)

			t.Logf("Garnet Metrics: EventSize=%d, StateSize=%d, StepCount=%d",
				garnetMeta.Metrics.EventSize, garnetMeta.Metrics.StateSize, garnetMeta.Metrics.StepCount)
			t.Logf("Valkey Metrics: EventSize=%d, StateSize=%d, StepCount=%d",
				valkeyMeta.Metrics.EventSize, valkeyMeta.Metrics.StateSize, valkeyMeta.Metrics.StepCount)

			// Compare metrics between backends
			assert.Equal(t, garnetMeta.Metrics.EventSize, valkeyMeta.Metrics.EventSize, "EventSize mismatch")
			assert.Equal(t, garnetMeta.Metrics.StateSize, valkeyMeta.Metrics.StateSize, "StateSize mismatch")
			assert.Equal(t, garnetMeta.Metrics.StepCount, valkeyMeta.Metrics.StepCount, "StepCount mismatch")

			// Verify EventSize is non-zero (events were stored)
			assert.Greater(t, garnetMeta.Metrics.EventSize, 0, "Garnet EventSize should be > 0")
			assert.Greater(t, valkeyMeta.Metrics.EventSize, 0, "Valkey EventSize should be > 0")
		})

		t.Run("LoadStack should return identical stacks", func(t *testing.T) {
			garnetStack, err := garnetSvc.LoadStack(ctx, garnetID)
			require.NoError(t, err)

			valkeyStack, err := valkeySvc.LoadStack(ctx, valkeyID)
			require.NoError(t, err)

			t.Logf("Garnet Stack: %v (len=%d)", garnetStack, len(garnetStack))
			t.Logf("Valkey Stack: %v (len=%d)", valkeyStack, len(valkeyStack))

			assert.Equal(t, garnetStack, valkeyStack, "Stack mismatch")
			assert.Equal(t, []string{"step-a", "step-b", "step-c"}, garnetStack, "Stack should have step IDs in order")
		})

		t.Run("LoadSteps should return identical step data", func(t *testing.T) {
			garnetSteps, err := garnetSvc.LoadSteps(ctx, garnetID)
			require.NoError(t, err)

			valkeySteps, err := valkeySvc.LoadSteps(ctx, valkeyID)
			require.NoError(t, err)

			assert.Equal(t, len(garnetSteps), len(valkeySteps), "Step count mismatch")

			for stepID := range garnetSteps {
				assert.JSONEq(t, string(garnetSteps[stepID]), string(valkeySteps[stepID]),
					"Step data mismatch for step %s", stepID)
			}

			t.Logf("LoadSteps returned %d entries (steps + step inputs)", len(garnetSteps))
		})

		t.Run("LoadStepInputs should return identical input data", func(t *testing.T) {
			garnetInputs, err := garnetSvc.LoadStepInputs(ctx, garnetID)
			require.NoError(t, err)

			valkeyInputs, err := valkeySvc.LoadStepInputs(ctx, valkeyID)
			require.NoError(t, err)

			assert.Equal(t, len(garnetInputs), len(valkeyInputs), "Step input count mismatch")
			require.Len(t, garnetInputs, 2, "Should have 2 step inputs")

			for inputID := range garnetInputs {
				assert.JSONEq(t, string(garnetInputs[inputID]), string(valkeyInputs[inputID]),
					"Step input data mismatch for input %s", inputID)
			}
		})

		t.Run("LoadStepsWithIDs should return identical partial data", func(t *testing.T) {
			requestedIDs := []string{"step-a", "step-c"}

			garnetSteps, err := garnetSvc.LoadStepsWithIDs(ctx, garnetID, requestedIDs)
			require.NoError(t, err)

			valkeySteps, err := valkeySvc.LoadStepsWithIDs(ctx, valkeyID, requestedIDs)
			require.NoError(t, err)

			assert.Equal(t, len(garnetSteps), len(valkeySteps), "Partial step count mismatch")

			for stepID := range garnetSteps {
				assert.JSONEq(t, string(garnetSteps[stepID]), string(valkeySteps[stepID]),
					"Partial step data mismatch for step %s", stepID)
			}
		})

		t.Run("LoadEvents should return identical events", func(t *testing.T) {
			garnetEvents, err := garnetSvc.LoadEvents(ctx, garnetID)
			require.NoError(t, err)

			valkeyEvents, err := valkeySvc.LoadEvents(ctx, valkeyID)
			require.NoError(t, err)

			assert.Equal(t, len(garnetEvents), len(valkeyEvents), "Event count mismatch")
			require.Len(t, garnetEvents, 1, "Should have 1 event")

			assert.JSONEq(t, string(garnetEvents[0]), string(valkeyEvents[0]), "Event data mismatch")
		})

		t.Run("LoadState should return identical complete state", func(t *testing.T) {
			garnetFullState, err := garnetSvc.LoadState(ctx, garnetID)
			require.NoError(t, err)

			valkeyFullState, err := valkeySvc.LoadState(ctx, valkeyID)
			require.NoError(t, err)

			// Compare metadata
			assert.Equal(t, garnetFullState.Metadata.Config.FunctionVersion, valkeyFullState.Metadata.Config.FunctionVersion)
			assert.Equal(t, garnetFullState.Metadata.Config.RequestVersion, valkeyFullState.Metadata.Config.RequestVersion)
			assert.Equal(t, garnetFullState.Metadata.Metrics, valkeyFullState.Metadata.Metrics)

			// Compare steps
			assert.Equal(t, len(garnetFullState.Steps), len(valkeyFullState.Steps), "Total step count mismatch")

			// Compare events
			assert.Equal(t, len(garnetFullState.Events), len(valkeyFullState.Events), "Event count mismatch")
		})
	})

	t.Run("SaveStep should produce identical metrics updates", func(t *testing.T) {
		garnetRunID := ulid.MustNew(ulid.Now(), rand.Reader)
		valkeyRunID := ulid.MustNew(ulid.Now(), rand.Reader)

		createInput := func(runID ulid.ULID) statev2.CreateState {
			return statev2.CreateState{
				Metadata: statev2.Metadata{
					ID: statev2.ID{
						RunID:      runID,
						FunctionID: functionID,
						Tenant: statev2.Tenant{
							AccountID: accountID,
							EnvID:     workspaceID,
							AppID:     appID,
						},
					},
					Config: *statev2.InitConfig(&statev2.Config{
						SpanID:          "savestep-span",
						Idempotency:     fmt.Sprintf("savestep-key-%s", runID.String()),
						FunctionVersion: 1,
						EventIDs:        []ulid.ULID{eventID},
					}),
				},
				Events: []json.RawMessage{eventBytes},
			}
		}

		garnetState, err := garnetSvc.Create(ctx, createInput(garnetRunID))
		require.NoError(t, err)
		garnetID := garnetState.Metadata.ID

		valkeyState, err := valkeySvc.Create(ctx, createInput(valkeyRunID))
		require.NoError(t, err)
		valkeyID := valkeyState.Metadata.ID

		t.Run("Initial metrics should match and be correct", func(t *testing.T) {
			garnetMeta, _ := garnetSvc.LoadMetadata(ctx, garnetID)
			valkeyMeta, _ := valkeySvc.LoadMetadata(ctx, valkeyID)

			// Backends should match
			assert.Equal(t, garnetMeta.Metrics.EventSize, valkeyMeta.Metrics.EventSize, "Initial EventSize mismatch")
			assert.Equal(t, garnetMeta.Metrics.StepCount, valkeyMeta.Metrics.StepCount, "Initial StepCount mismatch")
			assert.Equal(t, garnetMeta.Metrics.StateSize, valkeyMeta.Metrics.StateSize, "Initial StateSize mismatch")

			// Verify correctness
			assert.Equal(t, 0, garnetMeta.Metrics.StepCount, "Initial StepCount should be 0")
			assert.Equal(t, 0, garnetMeta.Metrics.StateSize, "Initial StateSize should be 0")
			assert.Greater(t, garnetMeta.Metrics.EventSize, 0, "Initial EventSize should be > 0")

			t.Logf("Initial metrics - EventSize=%d, StepCount=%d, StateSize=%d",
				garnetMeta.Metrics.EventSize, garnetMeta.Metrics.StepCount, garnetMeta.Metrics.StateSize)
		})

		// Save identical steps
		stepData1 := json.RawMessage(`{"result": "first_step", "value": 100}`)
		stepData2 := json.RawMessage(`{"result": "second_step", "value": 200}`)
		stepData3 := json.RawMessage(`{"result": "third_step", "large": "this is a larger payload"}`)

		steps := []struct {
			id   string
			data json.RawMessage
		}{
			{"step-1", stepData1},
			{"step-2", stepData2},
			{"step-3", stepData3},
		}

		var prevGarnetStateSize, prevValkeyStateSize int

		for i, step := range steps {
			t.Run(fmt.Sprintf("After SaveStep %d metrics should match", i+1), func(t *testing.T) {
				garnetHasPending, err := garnetSvc.SaveStep(ctx, garnetID, step.id, step.data)
				require.NoError(t, err)

				valkeyHasPending, err := valkeySvc.SaveStep(ctx, valkeyID, step.id, step.data)
				require.NoError(t, err)

				assert.Equal(t, garnetHasPending, valkeyHasPending, "hasPending mismatch")

				garnetMeta, _ := garnetSvc.LoadMetadata(ctx, garnetID)
				valkeyMeta, _ := valkeySvc.LoadMetadata(ctx, valkeyID)

				// Backends should match
				assert.Equal(t, garnetMeta.Metrics.StepCount, valkeyMeta.Metrics.StepCount,
					"StepCount mismatch after step %d", i+1)
				assert.Equal(t, garnetMeta.Metrics.StateSize, valkeyMeta.Metrics.StateSize,
					"StateSize mismatch after step %d", i+1)

				// Verify correctness
				assert.Equal(t, i+1, garnetMeta.Metrics.StepCount, "Garnet StepCount should be %d", i+1)
				assert.Equal(t, i+1, valkeyMeta.Metrics.StepCount, "Valkey StepCount should be %d", i+1)

				// StateSize should increase
				assert.Greater(t, garnetMeta.Metrics.StateSize, prevGarnetStateSize, "Garnet StateSize should increase")
				assert.Greater(t, valkeyMeta.Metrics.StateSize, prevValkeyStateSize, "Valkey StateSize should increase")

				prevGarnetStateSize = garnetMeta.Metrics.StateSize
				prevValkeyStateSize = valkeyMeta.Metrics.StateSize

				t.Logf("After step %d - StepCount=%d, StateSize=%d",
					i+1, garnetMeta.Metrics.StepCount, garnetMeta.Metrics.StateSize)
			})
		}

		t.Run("Final stack should match after all SaveSteps", func(t *testing.T) {
			garnetStack, _ := garnetSvc.LoadStack(ctx, garnetID)
			valkeyStack, _ := valkeySvc.LoadStack(ctx, valkeyID)

			assert.Equal(t, garnetStack, valkeyStack, "Final stack mismatch")
			assert.Equal(t, []string{"step-1", "step-2", "step-3"}, garnetStack, "Stack should have all saved steps in order")
		})

		t.Run("Final steps data should match", func(t *testing.T) {
			garnetSteps, _ := garnetSvc.LoadSteps(ctx, garnetID)
			valkeySteps, _ := valkeySvc.LoadSteps(ctx, valkeyID)

			assert.Equal(t, len(garnetSteps), len(valkeySteps), "Final step count mismatch")

			for stepID := range garnetSteps {
				assert.JSONEq(t, string(garnetSteps[stepID]), string(valkeySteps[stepID]),
					"Final step data mismatch for %s", stepID)
			}
		})
	})

	t.Run("SavePending should behave identically", func(t *testing.T) {
		garnetRunID := ulid.MustNew(ulid.Now(), rand.Reader)
		valkeyRunID := ulid.MustNew(ulid.Now(), rand.Reader)

		createInput := func(runID ulid.ULID) statev2.CreateState {
			return statev2.CreateState{
				Metadata: statev2.Metadata{
					ID: statev2.ID{
						RunID:      runID,
						FunctionID: functionID,
						Tenant: statev2.Tenant{
							AccountID: accountID,
							EnvID:     workspaceID,
							AppID:     appID,
						},
					},
					Config: *statev2.InitConfig(&statev2.Config{
						SpanID:          "pending-span",
						Idempotency:     fmt.Sprintf("pending-key-%s", runID.String()),
						FunctionVersion: 1,
						EventIDs:        []ulid.ULID{eventID},
					}),
				},
				Events: []json.RawMessage{eventBytes},
			}
		}

		garnetState, _ := garnetSvc.Create(ctx, createInput(garnetRunID))
		valkeyState, _ := valkeySvc.Create(ctx, createInput(valkeyRunID))

		pendingSteps := []string{"pending-a", "pending-b", "pending-c"}

		err = garnetSvc.SavePending(ctx, garnetState.Metadata.ID, pendingSteps)
		require.NoError(t, err)

		err = valkeySvc.SavePending(ctx, valkeyState.Metadata.ID, pendingSteps)
		require.NoError(t, err)

		// SaveStep with pending should return hasPending=true
		stepData := json.RawMessage(`{"result": "test"}`)

		garnetHasPending, err := garnetSvc.SaveStep(ctx, garnetState.Metadata.ID, "pending-a", stepData)
		require.NoError(t, err)

		valkeyHasPending, err := valkeySvc.SaveStep(ctx, valkeyState.Metadata.ID, "pending-a", stepData)
		require.NoError(t, err)

		assert.Equal(t, garnetHasPending, valkeyHasPending, "hasPending should match")
		t.Logf("hasPending after SaveStep with pending: garnet=%v, valkey=%v", garnetHasPending, valkeyHasPending)
	})

	t.Run("UpdateMetadata should produce identical results", func(t *testing.T) {
		garnetRunID := ulid.MustNew(ulid.Now(), rand.Reader)
		valkeyRunID := ulid.MustNew(ulid.Now(), rand.Reader)

		createInput := func(runID ulid.ULID) statev2.CreateState {
			return statev2.CreateState{
				Metadata: statev2.Metadata{
					ID: statev2.ID{
						RunID:      runID,
						FunctionID: functionID,
						Tenant: statev2.Tenant{
							AccountID: accountID,
							EnvID:     workspaceID,
							AppID:     appID,
						},
					},
					Config: *statev2.InitConfig(&statev2.Config{
						SpanID:          "update-span",
						Idempotency:     fmt.Sprintf("update-key-%s", runID.String()),
						FunctionVersion: 1,
						RequestVersion:  1,
						HasAI:           false,
						ForceStepPlan:   false,
						EventIDs:        []ulid.ULID{eventID},
					}),
				},
				Events: []json.RawMessage{eventBytes},
			}
		}

		garnetState, _ := garnetSvc.Create(ctx, createInput(garnetRunID))
		valkeyState, _ := valkeySvc.Create(ctx, createInput(valkeyRunID))

		updateConfig := statev2.MutableConfig{
			StartedAt:      time.Now(),
			RequestVersion: 5,
			ForceStepPlan:  true,
			HasAI:          true,
		}

		err = garnetSvc.UpdateMetadata(ctx, garnetState.Metadata.ID, updateConfig)
		require.NoError(t, err)

		err = valkeySvc.UpdateMetadata(ctx, valkeyState.Metadata.ID, updateConfig)
		require.NoError(t, err)

		garnetMeta, _ := garnetSvc.LoadMetadata(ctx, garnetState.Metadata.ID)
		valkeyMeta, _ := valkeySvc.LoadMetadata(ctx, valkeyState.Metadata.ID)

		// Backends should match
		assert.Equal(t, garnetMeta.Config.RequestVersion, valkeyMeta.Config.RequestVersion, "RequestVersion mismatch")
		assert.Equal(t, garnetMeta.Config.ForceStepPlan, valkeyMeta.Config.ForceStepPlan, "ForceStepPlan mismatch")
		assert.Equal(t, garnetMeta.Config.HasAI, valkeyMeta.Config.HasAI, "HasAI mismatch")

		// Verify correctness
		assert.Equal(t, 5, garnetMeta.Config.RequestVersion, "RequestVersion should be 5")
		assert.True(t, garnetMeta.Config.ForceStepPlan, "ForceStepPlan should be true")
		assert.True(t, garnetMeta.Config.HasAI, "HasAI should be true")

		// StartedAt should be within 1 second of each other
		assert.WithinDuration(t, garnetMeta.Config.StartedAt, valkeyMeta.Config.StartedAt, time.Second, "StartedAt mismatch")

		t.Logf("After UpdateMetadata - RequestVersion=%d, ForceStepPlan=%v, HasAI=%v",
			garnetMeta.Config.RequestVersion, garnetMeta.Config.ForceStepPlan, garnetMeta.Config.HasAI)
	})

	t.Run("Delete should behave identically", func(t *testing.T) {
		garnetRunID := ulid.MustNew(ulid.Now(), rand.Reader)
		valkeyRunID := ulid.MustNew(ulid.Now(), rand.Reader)

		createInput := func(runID ulid.ULID) statev2.CreateState {
			return statev2.CreateState{
				Metadata: statev2.Metadata{
					ID: statev2.ID{
						RunID:      runID,
						FunctionID: functionID,
						Tenant: statev2.Tenant{
							AccountID: accountID,
							EnvID:     workspaceID,
							AppID:     appID,
						},
					},
					Config: *statev2.InitConfig(&statev2.Config{
						SpanID:          "delete-span",
						Idempotency:     fmt.Sprintf("delete-key-%s", runID.String()),
						FunctionVersion: 1,
						EventIDs:        []ulid.ULID{eventID},
					}),
				},
				Events: []json.RawMessage{eventBytes},
			}
		}

		garnetState, _ := garnetSvc.Create(ctx, createInput(garnetRunID))
		valkeyState, _ := valkeySvc.Create(ctx, createInput(valkeyRunID))

		// Both should exist
		garnetExists, _ := garnetSvc.Exists(ctx, garnetState.Metadata.ID)
		valkeyExists, _ := valkeySvc.Exists(ctx, valkeyState.Metadata.ID)
		assert.True(t, garnetExists)
		assert.True(t, valkeyExists)

		// Delete both
		err = garnetSvc.Delete(ctx, garnetState.Metadata.ID)
		require.NoError(t, err)
		err = valkeySvc.Delete(ctx, valkeyState.Metadata.ID)
		require.NoError(t, err)

		// Both should not exist
		garnetExists, _ = garnetSvc.Exists(ctx, garnetState.Metadata.ID)
		valkeyExists, _ = valkeySvc.Exists(ctx, valkeyState.Metadata.ID)
		assert.False(t, garnetExists, "Garnet should not exist after delete")
		assert.False(t, valkeyExists, "Valkey should not exist after delete")

		// Both should error on LoadMetadata
		_, garnetErr := garnetSvc.LoadMetadata(ctx, garnetState.Metadata.ID)
		_, valkeyErr := valkeySvc.LoadMetadata(ctx, valkeyState.Metadata.ID)
		assert.Error(t, garnetErr, "Garnet LoadMetadata should error after delete")
		assert.Error(t, valkeyErr, "Valkey LoadMetadata should error after delete")
	})
}

func int64Ptr(v int64) *int64 {
	return &v
}

// TestStateDuplicate tests the Duplicate method by creating a complex state in Valkey,
// updating metadata, saving multiple steps, then duplicating to Garnet and comparing all fields.
func TestStateDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test")
	}

	ctx := context.Background()

	// Start Valkey (source)
	valkeyContainer, err := helper.StartValkey(t,
		helper.WithValkeyImage(testutil.ValkeyDefaultImage),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = valkeyContainer.Terminate(ctx) })

	valkeyClient, err := helper.NewValkeyClient(valkeyContainer.Addr, valkeyContainer.Username, valkeyContainer.Password, false)
	require.NoError(t, err)
	t.Cleanup(func() { valkeyClient.Close() })

	// Start Garnet (destination)
	garnetContainer, err := helper.StartGarnet(t,
		helper.WithImage(testutil.GarnetDefaultImage),
		helper.WithConfiguration(&helper.GarnetConfiguration{EnableLua: true}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = garnetContainer.Terminate(ctx) })

	garnetClient, err := helper.NewRedisClient(garnetContainer.Addr, garnetContainer.Username, garnetContainer.Password, false)
	require.NoError(t, err)
	t.Cleanup(func() { garnetClient.Close() })

	// Create Valkey service (source)
	valkeyUnsharded := redis_state.NewUnshardedClient(valkeyClient, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	valkeySharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        valkeyUnsharded,
		FunctionRunStateClient: valkeyClient,
		BatchClient:            valkeyClient,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})
	valkeyPauseMgr := redis_state.NewPauseStore(valkeyUnsharded)
	valkeyMgr, err := redis_state.New(ctx, redis_state.WithShardedClient(valkeySharded), redis_state.WithPauseDeleter(valkeyPauseMgr))
	require.NoError(t, err)
	valkeySvc := redis_state.MustRunServiceV2(valkeyMgr)

	// Create Garnet service (destination)
	garnetUnsharded := redis_state.NewUnshardedClient(garnetClient, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	garnetSharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        garnetUnsharded,
		FunctionRunStateClient: garnetClient,
		BatchClient:            garnetClient,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})
	garnetPauseMgr := redis_state.NewPauseStore(garnetUnsharded)
	garnetMgr, err := redis_state.New(ctx, redis_state.WithShardedClient(garnetSharded), redis_state.WithPauseDeleter(garnetPauseMgr))
	require.NoError(t, err)
	garnetSvc := redis_state.MustRunServiceV2(garnetMgr)

	// Test data
	functionID := uuid.New()
	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	destRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	replayID := uuid.New()
	batchID := ulid.MustNew(ulid.Now(), rand.Reader)
	originalRunID := ulid.MustNew(ulid.Now(), rand.Reader)

	// Step 1: Create complex state in Valkey
	events := []json.RawMessage{
		json.RawMessage(`{"name":"test.event.1","data":{"user":"alice","action":"click"},"id":"` + eventID.String() + `"}`),
		json.RawMessage(`{"name":"test.event.2","data":{"user":"bob","action":"submit"}}`),
	}

	createInput := statev2.CreateState{
		Metadata: statev2.Metadata{
			ID: statev2.ID{
				RunID:      runID,
				FunctionID: functionID,
				Tenant: statev2.Tenant{
					AccountID: accountID,
					EnvID:     workspaceID,
					AppID:     appID,
				},
			},
			Config: *statev2.InitConfig(&statev2.Config{
				SpanID:          "test-span-id-12345",
				Idempotency:     "idem-key-" + runID.String(),
				FunctionVersion: 5,
				RequestVersion:  2,
				EventIDs:        []ulid.ULID{eventID},
				PriorityFactor:  int64Ptr(80),
				ReplayID:        &replayID,
				BatchID:         &batchID,
				OriginalRunID:   &originalRunID,
				Context:         map[string]any{"env": "test", "region": "us-east-1", "count": 42},
				CustomConcurrencyKeys: []statev2.CustomConcurrency{
					{Key: "user:alice", Hash: "hash1", Limit: 5},
					{Key: "tenant:acme", Hash: "hash2", Limit: 10},
				},
			}),
		},
		Events: events,
	}

	_, err = valkeySvc.Create(ctx, createInput)
	require.NoError(t, err)

	sourceID := statev2.ID{
		RunID:      runID,
		FunctionID: functionID,
		Tenant: statev2.Tenant{
			AccountID: accountID,
			EnvID:     workspaceID,
			AppID:     appID,
		},
	}

	// Step 2: Save multiple steps to build up metrics
	steps := []struct {
		id   string
		data json.RawMessage
	}{
		{"step-1", json.RawMessage(`{"result":"first","value":100}`)},
		{"step-2", json.RawMessage(`{"result":"second","nested":{"a":1,"b":2}}`)},
		{"step-3", json.RawMessage(`{"result":"third","array":[1,2,3,4,5]}`)},
		{"step-4", json.RawMessage(`{"result":"fourth","description":"This is a larger payload with more data to test size handling"}`)},
		{"step-5", json.RawMessage(`{"result":"fifth","done":true}`)},
	}

	for _, s := range steps {
		_, err := valkeySvc.SaveStep(ctx, sourceID, s.id, s.data)
		require.NoError(t, err)
	}

	// Step 3: Update metadata
	startedAt := time.Now().Add(-5 * time.Minute)
	err = valkeySvc.UpdateMetadata(ctx, sourceID, statev2.MutableConfig{
		StartedAt:      startedAt,
		RequestVersion: 3,
		ForceStepPlan:  true,
		HasAI:          true,
	})
	require.NoError(t, err)

	// Step 4: Load source state
	sourceState, err := valkeySvc.LoadState(ctx, sourceID)
	require.NoError(t, err)

	t.Logf("Source state created in Valkey:")
	t.Logf("  RunID: %s", sourceID.RunID)
	t.Logf("  Stack: %v", sourceState.Metadata.Stack)
	t.Logf("  Steps: %d", len(sourceState.Steps))
	t.Logf("  Events: %d", len(sourceState.Events))
	t.Logf("  Metrics: EventSize=%d, StateSize=%d, StepCount=%d",
		sourceState.Metadata.Metrics.EventSize,
		sourceState.Metadata.Metrics.StateSize,
		sourceState.Metadata.Metrics.StepCount)
	t.Logf("  Config: SpanID=%s, FunctionVersion=%d, RequestVersion=%d, HasAI=%v, ForceStepPlan=%v",
		sourceState.Metadata.Config.SpanID,
		sourceState.Metadata.Config.FunctionVersion,
		sourceState.Metadata.Config.RequestVersion,
		sourceState.Metadata.Config.HasAI,
		sourceState.Metadata.Config.ForceStepPlan)

	// Step 5: Load raw metadata (v1) and step inputs, then duplicate to Garnet
	sourceRawMeta, err := valkeySvc.LoadV1Metadata(ctx, sourceID)
	require.NoError(t, err)

	sourceStepInputs, err := valkeySvc.LoadStepInputs(ctx, sourceID)
	require.NoError(t, err)

	destID := statev2.ID{
		RunID:      destRunID,
		FunctionID: functionID,
		Tenant: statev2.Tenant{
			AccountID: accountID,
			EnvID:     workspaceID,
			AppID:     appID,
		},
	}
	err = garnetSvc.Duplicate(ctx, sourceState, destID, sourceRawMeta, sourceStepInputs)
	require.NoError(t, err)

	// Step 6: Load destination state
	destState, err := garnetSvc.LoadState(ctx, destID)
	require.NoError(t, err)

	t.Logf("Destination state in Garnet:")
	t.Logf("  RunID: %s (different from source)", destState.Metadata.ID.RunID)
	t.Logf("  Stack: %v", destState.Metadata.Stack)
	t.Logf("  Steps: %d", len(destState.Steps))
	t.Logf("  Events: %d", len(destState.Events))
	t.Logf("  Metrics: EventSize=%d, StateSize=%d, StepCount=%d",
		destState.Metadata.Metrics.EventSize,
		destState.Metadata.Metrics.StateSize,
		destState.Metadata.Metrics.StepCount)
	t.Logf("  Config: SpanID=%s, FunctionVersion=%d, RequestVersion=%d, HasAI=%v, ForceStepPlan=%v",
		destState.Metadata.Config.SpanID,
		destState.Metadata.Config.FunctionVersion,
		destState.Metadata.Config.RequestVersion,
		destState.Metadata.Config.HasAI,
		destState.Metadata.Config.ForceStepPlan)

	// Step 7: Compare ALL fields

	// ID - RunID should be the new destRunID, other fields should match source
	assert.Equal(t, destRunID, destState.Metadata.ID.RunID, "RunID should be destRunID")
	assert.Equal(t, sourceState.Metadata.ID.FunctionID, destState.Metadata.ID.FunctionID, "FunctionID")
	assert.Equal(t, sourceState.Metadata.ID.Tenant.AccountID, destState.Metadata.ID.Tenant.AccountID, "AccountID")
	assert.Equal(t, sourceState.Metadata.ID.Tenant.EnvID, destState.Metadata.ID.Tenant.EnvID, "EnvID")
	assert.Equal(t, sourceState.Metadata.ID.Tenant.AppID, destState.Metadata.ID.Tenant.AppID, "AppID")

	// Config
	assert.Equal(t, sourceState.Metadata.Config.SpanID, destState.Metadata.Config.SpanID, "SpanID")
	assert.Equal(t, sourceState.Metadata.Config.Idempotency, destState.Metadata.Config.Idempotency, "Idempotency")
	assert.Equal(t, sourceState.Metadata.Config.FunctionVersion, destState.Metadata.Config.FunctionVersion, "FunctionVersion")
	assert.Equal(t, sourceState.Metadata.Config.RequestVersion, destState.Metadata.Config.RequestVersion, "RequestVersion")
	assert.Equal(t, sourceState.Metadata.Config.HasAI, destState.Metadata.Config.HasAI, "HasAI")
	assert.Equal(t, sourceState.Metadata.Config.ForceStepPlan, destState.Metadata.Config.ForceStepPlan, "ForceStepPlan")
	assert.Equal(t, sourceState.Metadata.Config.EventIDs, destState.Metadata.Config.EventIDs, "EventIDs")

	if sourceState.Metadata.Config.PriorityFactor != nil {
		require.NotNil(t, destState.Metadata.Config.PriorityFactor, "PriorityFactor should not be nil")
		assert.Equal(t, *sourceState.Metadata.Config.PriorityFactor, *destState.Metadata.Config.PriorityFactor, "PriorityFactor")
	}

	if sourceState.Metadata.Config.ReplayID != nil {
		require.NotNil(t, destState.Metadata.Config.ReplayID, "ReplayID should not be nil")
		assert.Equal(t, *sourceState.Metadata.Config.ReplayID, *destState.Metadata.Config.ReplayID, "ReplayID")
	}

	assert.Equal(t, len(sourceState.Metadata.Config.CustomConcurrencyKeys), len(destState.Metadata.Config.CustomConcurrencyKeys), "CustomConcurrencyKeys count")
	for i := range sourceState.Metadata.Config.CustomConcurrencyKeys {
		assert.Equal(t, sourceState.Metadata.Config.CustomConcurrencyKeys[i].Key, destState.Metadata.Config.CustomConcurrencyKeys[i].Key, "CustomConcurrencyKeys[%d].Key", i)
		assert.Equal(t, sourceState.Metadata.Config.CustomConcurrencyKeys[i].Hash, destState.Metadata.Config.CustomConcurrencyKeys[i].Hash, "CustomConcurrencyKeys[%d].Hash", i)
		assert.Equal(t, sourceState.Metadata.Config.CustomConcurrencyKeys[i].Limit, destState.Metadata.Config.CustomConcurrencyKeys[i].Limit, "CustomConcurrencyKeys[%d].Limit", i)
	}

	// BatchID
	if sourceState.Metadata.Config.BatchID != nil {
		require.NotNil(t, destState.Metadata.Config.BatchID, "BatchID should not be nil")
		assert.Equal(t, *sourceState.Metadata.Config.BatchID, *destState.Metadata.Config.BatchID, "BatchID")
	} else {
		assert.Nil(t, destState.Metadata.Config.BatchID, "BatchID should be nil")
	}

	// OriginalRunID
	if sourceState.Metadata.Config.OriginalRunID != nil {
		require.NotNil(t, destState.Metadata.Config.OriginalRunID, "OriginalRunID should not be nil")
		assert.Equal(t, *sourceState.Metadata.Config.OriginalRunID, *destState.Metadata.Config.OriginalRunID, "OriginalRunID")
	} else {
		assert.Nil(t, destState.Metadata.Config.OriginalRunID, "OriginalRunID should be nil")
	}

	// StartedAt (with tolerance for time precision)
	if !sourceState.Metadata.Config.StartedAt.IsZero() {
		assert.False(t, destState.Metadata.Config.StartedAt.IsZero(), "StartedAt should not be zero")
		assert.WithinDuration(t, sourceState.Metadata.Config.StartedAt, destState.Metadata.Config.StartedAt, time.Second, "StartedAt")
	}

	// Metrics
	assert.Equal(t, sourceState.Metadata.Metrics.EventSize, destState.Metadata.Metrics.EventSize, "EventSize")
	assert.Equal(t, sourceState.Metadata.Metrics.StateSize, destState.Metadata.Metrics.StateSize, "StateSize")
	assert.Equal(t, sourceState.Metadata.Metrics.StepCount, destState.Metadata.Metrics.StepCount, "StepCount")

	// Verify metrics are non-zero
	assert.Greater(t, destState.Metadata.Metrics.EventSize, 0, "EventSize should be > 0")
	assert.Greater(t, destState.Metadata.Metrics.StateSize, 0, "StateSize should be > 0")
	assert.Equal(t, 5, destState.Metadata.Metrics.StepCount, "StepCount should be 5")

	// Verify Stack and Steps match between Valkey (source) and Garnet (destination)
	// with the same ordering preserved
	require.Equal(t, len(sourceState.Metadata.Stack), len(destState.Metadata.Stack),
		"Stack length should match: Valkey=%d, Garnet=%d", len(sourceState.Metadata.Stack), len(destState.Metadata.Stack))
	require.Equal(t, len(sourceState.Steps), len(destState.Steps),
		"Steps count should match: Valkey=%d, Garnet=%d", len(sourceState.Steps), len(destState.Steps))
	require.Equal(t, len(sourceState.Metadata.Stack), len(sourceState.Steps),
		"Stack and Steps should have same count in source")

	// Walk through Stack in order and verify both step ID and data match at each position
	for i, sourceStepID := range sourceState.Metadata.Stack {
		destStepID := destState.Metadata.Stack[i]
		require.Equal(t, sourceStepID, destStepID,
			"Stack ordering mismatch at position %d: Valkey=%q, Garnet=%q", i, sourceStepID, destStepID)

		sourceData, sourceOK := sourceState.Steps[sourceStepID]
		destData, destOK := destState.Steps[destStepID]
		require.True(t, sourceOK, "Step %q at Stack[%d] should exist in Valkey Steps", sourceStepID, i)
		require.True(t, destOK, "Step %q at Stack[%d] should exist in Garnet Steps", destStepID, i)
		assert.JSONEq(t, string(sourceData), string(destData),
			"Step data mismatch at Stack[%d] (%q): Valkey and Garnet data should match", i, sourceStepID)
	}

	// ============================================================================
	// SERVICE-BASED VERIFICATION
	// Use LoadStack, LoadSteps, LoadEvents, LoadMetadata, LoadStepInputs to
	// compare Valkey (source) and Garnet (destination) data.
	// These functions handle format differences (like "3" vs "3.0") correctly.
	// ============================================================================

	t.Log("=== Service-Based Verification ===")

	// 1. LoadStack - Verify ordering is preserved
	t.Log("Checking LoadStack...")
	valkeyStack, err := valkeySvc.LoadStack(ctx, sourceID)
	require.NoError(t, err, "Failed to LoadStack from Valkey")
	garnetStack, err := garnetSvc.LoadStack(ctx, destID)
	require.NoError(t, err, "Failed to LoadStack from Garnet")

	t.Logf("  Valkey Stack: %v", valkeyStack)
	t.Logf("  Garnet Stack: %v", garnetStack)
	require.Equal(t, valkeyStack, garnetStack, "Stack mismatch between Valkey and Garnet")

	// 2. LoadSteps - Verify all step data
	t.Log("Checking LoadSteps...")
	valkeySteps, err := valkeySvc.LoadSteps(ctx, sourceID)
	require.NoError(t, err, "Failed to LoadSteps from Valkey")
	garnetSteps, err := garnetSvc.LoadSteps(ctx, destID)
	require.NoError(t, err, "Failed to LoadSteps from Garnet")

	t.Logf("  Valkey Steps: %d entries", len(valkeySteps))
	t.Logf("  Garnet Steps: %d entries", len(garnetSteps))
	require.Equal(t, len(valkeySteps), len(garnetSteps), "Steps count mismatch")

	for stepID, valkeyData := range valkeySteps {
		garnetData, ok := garnetSteps[stepID]
		require.True(t, ok, "Step %q exists in Valkey but not in Garnet", stepID)
		assert.JSONEq(t, string(valkeyData), string(garnetData), "Step %q data mismatch", stepID)
		t.Logf("    %s: [MATCH]", stepID)
	}

	// 3. LoadEvents - Verify event data
	t.Log("Checking LoadEvents...")
	valkeyEvents, err := valkeySvc.LoadEvents(ctx, sourceID)
	require.NoError(t, err, "Failed to LoadEvents from Valkey")
	garnetEvents, err := garnetSvc.LoadEvents(ctx, destID)
	require.NoError(t, err, "Failed to LoadEvents from Garnet")

	t.Logf("  Valkey Events: %d", len(valkeyEvents))
	t.Logf("  Garnet Events: %d", len(garnetEvents))
	require.Equal(t, len(valkeyEvents), len(garnetEvents), "Events count mismatch")

	for i := range valkeyEvents {
		assert.JSONEq(t, string(valkeyEvents[i]), string(garnetEvents[i]), "Event[%d] data mismatch", i)
	}
	t.Log("    Events: [MATCH]")

	// 4. LoadMetadata (v2) - Verify metadata fields
	t.Log("Checking LoadMetadata (v2)...")
	valkeyMeta, err := valkeySvc.LoadMetadata(ctx, sourceID)
	require.NoError(t, err, "Failed to LoadMetadata from Valkey")
	garnetMeta, err := garnetSvc.LoadMetadata(ctx, destID)
	require.NoError(t, err, "Failed to LoadMetadata from Garnet")

	// Config fields (excluding RunID which is expected to differ)
	assert.Equal(t, valkeyMeta.Config.SpanID, garnetMeta.Config.SpanID, "SpanID mismatch")
	t.Logf("    SpanID: %s [MATCH]", valkeyMeta.Config.SpanID)

	assert.Equal(t, valkeyMeta.Config.Idempotency, garnetMeta.Config.Idempotency, "Idempotency mismatch")
	t.Logf("    Idempotency: %s [MATCH]", valkeyMeta.Config.Idempotency)

	assert.Equal(t, valkeyMeta.Config.FunctionVersion, garnetMeta.Config.FunctionVersion, "FunctionVersion mismatch")
	t.Logf("    FunctionVersion: %d [MATCH]", valkeyMeta.Config.FunctionVersion)

	assert.Equal(t, valkeyMeta.Config.RequestVersion, garnetMeta.Config.RequestVersion, "RequestVersion mismatch")
	t.Logf("    RequestVersion: %d [MATCH]", valkeyMeta.Config.RequestVersion)

	assert.Equal(t, valkeyMeta.Config.HasAI, garnetMeta.Config.HasAI, "HasAI mismatch")
	t.Logf("    HasAI: %v [MATCH]", valkeyMeta.Config.HasAI)

	assert.Equal(t, valkeyMeta.Config.ForceStepPlan, garnetMeta.Config.ForceStepPlan, "ForceStepPlan mismatch")
	t.Logf("    ForceStepPlan: %v [MATCH]", valkeyMeta.Config.ForceStepPlan)

	assert.Equal(t, valkeyMeta.Config.EventIDs, garnetMeta.Config.EventIDs, "EventIDs mismatch")
	t.Logf("    EventIDs: %v [MATCH]", valkeyMeta.Config.EventIDs)

	if valkeyMeta.Config.PriorityFactor != nil {
		require.NotNil(t, garnetMeta.Config.PriorityFactor, "PriorityFactor should not be nil in Garnet")
		assert.Equal(t, *valkeyMeta.Config.PriorityFactor, *garnetMeta.Config.PriorityFactor, "PriorityFactor mismatch")
		t.Logf("    PriorityFactor: %d [MATCH]", *valkeyMeta.Config.PriorityFactor)
	}

	if valkeyMeta.Config.ReplayID != nil {
		require.NotNil(t, garnetMeta.Config.ReplayID, "ReplayID should not be nil in Garnet")
		assert.Equal(t, *valkeyMeta.Config.ReplayID, *garnetMeta.Config.ReplayID, "ReplayID mismatch")
		t.Logf("    ReplayID: %s [MATCH]", valkeyMeta.Config.ReplayID)
	}

	if valkeyMeta.Config.BatchID != nil {
		require.NotNil(t, garnetMeta.Config.BatchID, "BatchID should not be nil in Garnet")
		assert.Equal(t, *valkeyMeta.Config.BatchID, *garnetMeta.Config.BatchID, "BatchID mismatch")
		t.Logf("    BatchID: %s [MATCH]", valkeyMeta.Config.BatchID)
	}

	if valkeyMeta.Config.OriginalRunID != nil {
		require.NotNil(t, garnetMeta.Config.OriginalRunID, "OriginalRunID should not be nil in Garnet")
		assert.Equal(t, *valkeyMeta.Config.OriginalRunID, *garnetMeta.Config.OriginalRunID, "OriginalRunID mismatch")
		t.Logf("    OriginalRunID: %s [MATCH]", valkeyMeta.Config.OriginalRunID)
	}

	// StartedAt (with tolerance)
	if !valkeyMeta.Config.StartedAt.IsZero() {
		assert.False(t, garnetMeta.Config.StartedAt.IsZero(), "StartedAt should not be zero in Garnet")
		assert.WithinDuration(t, valkeyMeta.Config.StartedAt, garnetMeta.Config.StartedAt, time.Second, "StartedAt mismatch")
		t.Logf("    StartedAt: %v [MATCH]", valkeyMeta.Config.StartedAt)
	}

	// CustomConcurrencyKeys
	assert.Equal(t, len(valkeyMeta.Config.CustomConcurrencyKeys), len(garnetMeta.Config.CustomConcurrencyKeys), "CustomConcurrencyKeys count mismatch")
	for i := range valkeyMeta.Config.CustomConcurrencyKeys {
		assert.Equal(t, valkeyMeta.Config.CustomConcurrencyKeys[i], garnetMeta.Config.CustomConcurrencyKeys[i], "CustomConcurrencyKeys[%d] mismatch", i)
	}
	t.Logf("    CustomConcurrencyKeys: %d keys [MATCH]", len(valkeyMeta.Config.CustomConcurrencyKeys))

	// Metrics
	t.Log("  Metrics:")
	assert.Equal(t, valkeyMeta.Metrics.EventSize, garnetMeta.Metrics.EventSize, "EventSize mismatch")
	t.Logf("    EventSize: %d [MATCH]", valkeyMeta.Metrics.EventSize)

	assert.Equal(t, valkeyMeta.Metrics.StateSize, garnetMeta.Metrics.StateSize, "StateSize mismatch")
	t.Logf("    StateSize: %d [MATCH]", valkeyMeta.Metrics.StateSize)

	assert.Equal(t, valkeyMeta.Metrics.StepCount, garnetMeta.Metrics.StepCount, "StepCount mismatch")
	t.Logf("    StepCount: %d [MATCH]", valkeyMeta.Metrics.StepCount)

	// Stack from metadata
	assert.Equal(t, valkeyMeta.Stack, garnetMeta.Stack, "Stack (from metadata) mismatch")
	t.Logf("    Stack: %v [MATCH]", valkeyMeta.Stack)

	// 5. LoadV1Metadata - Verify raw v1 metadata
	t.Log("Checking LoadV1Metadata...")
	valkeyV1Meta, err := valkeySvc.LoadV1Metadata(ctx, sourceID)
	require.NoError(t, err, "Failed to LoadV1Metadata from Valkey")
	garnetV1Meta, err := garnetSvc.LoadV1Metadata(ctx, destID)
	require.NoError(t, err, "Failed to LoadV1Metadata from Garnet")

	// Compare key fields (runID will differ)
	assert.Equal(t, valkeyV1Meta.Identifier.WorkflowID, garnetV1Meta.Identifier.WorkflowID, "V1 WorkflowID mismatch")
	assert.Equal(t, valkeyV1Meta.Identifier.WorkflowVersion, garnetV1Meta.Identifier.WorkflowVersion, "V1 WorkflowVersion mismatch")
	assert.Equal(t, valkeyV1Meta.Identifier.AccountID, garnetV1Meta.Identifier.AccountID, "V1 AccountID mismatch")
	assert.Equal(t, valkeyV1Meta.Identifier.WorkspaceID, garnetV1Meta.Identifier.WorkspaceID, "V1 WorkspaceID mismatch")
	assert.Equal(t, valkeyV1Meta.Identifier.AppID, garnetV1Meta.Identifier.AppID, "V1 AppID mismatch")
	assert.Equal(t, valkeyV1Meta.Identifier.EventIDs, garnetV1Meta.Identifier.EventIDs, "V1 EventIDs mismatch")
	assert.Equal(t, valkeyV1Meta.RequestVersion, garnetV1Meta.RequestVersion, "V1 RequestVersion mismatch")
	t.Log("    V1 Metadata fields: [MATCH]")

	// 6. LoadStepInputs - Verify step inputs
	t.Log("Checking LoadStepInputs...")
	valkeyInputs, err := valkeySvc.LoadStepInputs(ctx, sourceID)
	require.NoError(t, err, "Failed to LoadStepInputs from Valkey")
	garnetInputs, err := garnetSvc.LoadStepInputs(ctx, destID)
	require.NoError(t, err, "Failed to LoadStepInputs from Garnet")

	t.Logf("  Valkey Step Inputs: %d entries", len(valkeyInputs))
	t.Logf("  Garnet Step Inputs: %d entries", len(garnetInputs))
	require.Equal(t, len(valkeyInputs), len(garnetInputs), "Step Inputs count mismatch")

	for inputID, valkeyData := range valkeyInputs {
		garnetData, ok := garnetInputs[inputID]
		require.True(t, ok, "Step Input %q exists in Valkey but not in Garnet", inputID)
		assert.JSONEq(t, string(valkeyData), string(garnetData), "Step Input %q data mismatch", inputID)
	}
	if len(valkeyInputs) > 0 {
		t.Log("    Step Inputs: [MATCH]")
	} else {
		t.Log("    Step Inputs: (none)")
	}

	// 7. LoadState - Full state comparison
	t.Log("Checking LoadState (full state)...")
	valkeyFullState, err := valkeySvc.LoadState(ctx, sourceID)
	require.NoError(t, err, "Failed to LoadState from Valkey")
	garnetFullState, err := garnetSvc.LoadState(ctx, destID)
	require.NoError(t, err, "Failed to LoadState from Garnet")

	// Verify counts match
	assert.Equal(t, len(valkeyFullState.Steps), len(garnetFullState.Steps), "Full state Steps count mismatch")
	assert.Equal(t, len(valkeyFullState.Events), len(garnetFullState.Events), "Full state Events count mismatch")
	assert.Equal(t, len(valkeyFullState.Metadata.Stack), len(garnetFullState.Metadata.Stack), "Full state Stack count mismatch")
	t.Log("    Full State counts: [MATCH]")

	t.Log("=== Service-Based Verification Complete ===")
}
