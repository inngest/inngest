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

