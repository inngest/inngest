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

// TestStateOperationsAcrossBackends exercises the RunService v2 surface end to
// end against Valkey to guard the invariants the cross-backend migration
// primitives rely on (Create/LoadMetadata/SaveStep/SavePending/UpdateMetadata/Delete).
func TestStateOperationsAcrossBackends(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test")
	}

	ctx := context.Background()

	valkeyContainer, err := helper.StartValkey(t,
		helper.WithValkeyImage(testutil.ValkeyDefaultImage),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = valkeyContainer.Terminate(ctx) })

	valkeyClient, err := helper.NewValkeyClient(valkeyContainer.Addr, valkeyContainer.Username, valkeyContainer.Password, false)
	require.NoError(t, err)
	t.Cleanup(func() { valkeyClient.Close() })

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
	svc := redis_state.MustRunServiceV2(valkeyMgr)

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

	t.Run("Create with pre-filled steps persists config, steps, inputs, events", func(t *testing.T) {
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		stepData1 := map[string]any{"result": "step1_output", "count": 42}
		stepData2 := map[string]any{"result": "step2_output", "nested": map[string]any{"key": "value"}}
		stepData3 := map[string]any{"result": "step3_output", "items": []int{1, 2, 3, 4, 5}}

		inputData1 := map[string]any{"param": "input1", "number": 100}
		inputData2 := map[string]any{"param": "input2", "number": 200}

		input := statev2.CreateState{
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

		created, err := svc.Create(ctx, input)
		require.NoError(t, err)
		id := created.Metadata.ID

		t.Run("LoadMetadata returns persisted config", func(t *testing.T) {
			md, err := svc.LoadMetadata(ctx, id)
			require.NoError(t, err)

			assert.Equal(t, 3, md.Config.FunctionVersion)
			assert.Equal(t, 2, md.Config.RequestVersion)
			assert.Equal(t, "comparison-span", md.Config.SpanID)
			// Note: HasAI and ForceStepPlan are not persisted during Create,
			// but can be set via UpdateMetadata (covered below).

			assert.Equal(t, 1, len(md.Config.EventIDs))
			require.NotNil(t, md.Config.PriorityFactor)
			assert.Equal(t, int64(50), *md.Config.PriorityFactor)
			assert.Equal(t, 1, len(md.Config.CustomConcurrencyKeys))
		})

		t.Run("Metrics are populated after Create", func(t *testing.T) {
			md, err := svc.LoadMetadata(ctx, id)
			require.NoError(t, err)

			t.Logf("Metrics: EventSize=%d, StateSize=%d, StepCount=%d",
				md.Metrics.EventSize, md.Metrics.StateSize, md.Metrics.StepCount)

			assert.Greater(t, md.Metrics.EventSize, 0, "EventSize should be > 0")
		})

		t.Run("LoadStack returns memoized step IDs in order", func(t *testing.T) {
			stack, err := svc.LoadStack(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, []string{"step-a", "step-b", "step-c"}, stack)
		})

		t.Run("LoadSteps returns memoized step data", func(t *testing.T) {
			steps, err := svc.LoadSteps(ctx, id)
			require.NoError(t, err)
			// Implementation returns steps and step inputs combined.
			assert.GreaterOrEqual(t, len(steps), 3)
		})

		t.Run("LoadStepInputs returns memoized inputs", func(t *testing.T) {
			inputs, err := svc.LoadStepInputs(ctx, id)
			require.NoError(t, err)
			require.Len(t, inputs, 2, "Should have 2 step inputs")
		})

		t.Run("LoadStepsWithIDs returns only requested steps", func(t *testing.T) {
			requested := []string{"step-a", "step-c"}
			steps, err := svc.LoadStepsWithIDs(ctx, id, requested)
			require.NoError(t, err)
			assert.Len(t, steps, 2)
		})

		t.Run("LoadEvents returns the trigger events", func(t *testing.T) {
			events, err := svc.LoadEvents(ctx, id)
			require.NoError(t, err)
			require.Len(t, events, 1, "Should have 1 event")
			assert.JSONEq(t, string(eventBytes), string(events[0]))
		})

		t.Run("LoadState returns the complete state", func(t *testing.T) {
			full, err := svc.LoadState(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, 3, full.Metadata.Config.FunctionVersion)
			assert.Equal(t, 2, full.Metadata.Config.RequestVersion)
			assert.GreaterOrEqual(t, len(full.Steps), 3)
			assert.Len(t, full.Events, 1)
		})
	})

	t.Run("SaveStep updates metrics", func(t *testing.T) {
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		input := statev2.CreateState{
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

		created, err := svc.Create(ctx, input)
		require.NoError(t, err)
		id := created.Metadata.ID

		t.Run("Initial metrics are correct", func(t *testing.T) {
			md, err := svc.LoadMetadata(ctx, id)
			require.NoError(t, err)

			assert.Equal(t, 0, md.Metrics.StepCount, "Initial StepCount should be 0")
			// StateSize is initialized to len(events) + len(steps) + len(stepInputs)
			// at Create; with no pre-memoized steps/inputs this equals EventSize.
			assert.Equal(t, md.Metrics.EventSize, md.Metrics.StateSize, "Initial StateSize should equal EventSize")
			assert.Greater(t, md.Metrics.EventSize, 0, "Initial EventSize should be > 0")

			t.Logf("Initial metrics - EventSize=%d, StepCount=%d, StateSize=%d",
				md.Metrics.EventSize, md.Metrics.StepCount, md.Metrics.StateSize)
		})

		steps := []struct {
			id   string
			data json.RawMessage
		}{
			{"step-1", json.RawMessage(`{"result": "first_step", "value": 100}`)},
			{"step-2", json.RawMessage(`{"result": "second_step", "value": 200}`)},
			{"step-3", json.RawMessage(`{"result": "third_step", "large": "this is a larger payload"}`)},
		}

		var prevStateSize int
		for i, step := range steps {
			t.Run(fmt.Sprintf("After SaveStep %d metrics update", i+1), func(t *testing.T) {
				_, err := svc.SaveStep(ctx, id, step.id, step.data)
				require.NoError(t, err)

				md, err := svc.LoadMetadata(ctx, id)
				require.NoError(t, err)

				assert.Equal(t, i+1, md.Metrics.StepCount, "StepCount should be %d", i+1)
				assert.Greater(t, md.Metrics.StateSize, prevStateSize, "StateSize should increase")
				prevStateSize = md.Metrics.StateSize

				t.Logf("After step %d - StepCount=%d, StateSize=%d",
					i+1, md.Metrics.StepCount, md.Metrics.StateSize)
			})
		}

		t.Run("Final stack contains all saved steps in order", func(t *testing.T) {
			stack, err := svc.LoadStack(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, []string{"step-1", "step-2", "step-3"}, stack)
		})

		t.Run("Final steps data is retrievable", func(t *testing.T) {
			loaded, err := svc.LoadSteps(ctx, id)
			require.NoError(t, err)
			assert.Len(t, loaded, len(steps))
		})
	})

	t.Run("SavePending marks pending steps and SaveStep reports hasPending", func(t *testing.T) {
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		input := statev2.CreateState{
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

		created, err := svc.Create(ctx, input)
		require.NoError(t, err)
		id := created.Metadata.ID

		err = svc.SavePending(ctx, id, []string{"pending-a", "pending-b", "pending-c"})
		require.NoError(t, err)

		hasPending, err := svc.SaveStep(ctx, id, "pending-a", json.RawMessage(`{"result": "test"}`))
		require.NoError(t, err)
		assert.True(t, hasPending, "hasPending should be true when other pending steps remain")
	})

	t.Run("UpdateMetadata persists mutable config", func(t *testing.T) {
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		input := statev2.CreateState{
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
					EventIDs:        []ulid.ULID{eventID},
				}),
			},
			Events: []json.RawMessage{eventBytes},
		}

		created, err := svc.Create(ctx, input)
		require.NoError(t, err)
		id := created.Metadata.ID

		update := statev2.MutableConfig{
			StartedAt:      time.Now(),
			RequestVersion: 5,
			ForceStepPlan:  true,
			HasAI:          true,
		}
		err = svc.UpdateMetadata(ctx, id, update)
		require.NoError(t, err)

		md, err := svc.LoadMetadata(ctx, id)
		require.NoError(t, err)

		assert.Equal(t, 5, md.Config.RequestVersion, "RequestVersion should be 5")
		assert.True(t, md.Config.ForceStepPlan, "ForceStepPlan should be true")
		assert.True(t, md.Config.HasAI, "HasAI should be true")
		assert.WithinDuration(t, update.StartedAt, md.Config.StartedAt, time.Second, "StartedAt should be preserved")
	})

	t.Run("Delete removes the run", func(t *testing.T) {
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		input := statev2.CreateState{
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

		created, err := svc.Create(ctx, input)
		require.NoError(t, err)
		id := created.Metadata.ID

		exists, err := svc.Exists(ctx, id)
		require.NoError(t, err)
		assert.True(t, exists)

		err = svc.Delete(ctx, id)
		require.NoError(t, err)

		exists, err = svc.Exists(ctx, id)
		require.NoError(t, err)
		assert.False(t, exists, "run should not exist after delete")

		_, err = svc.LoadMetadata(ctx, id)
		assert.Error(t, err, "LoadMetadata should error after delete")
	})
}

func int64Ptr(v int64) *int64 {
	return &v
}
