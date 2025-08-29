package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

func TestV2Adapter(t *testing.T) {
	ctx := context.Background()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

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
		ctx,
		WithUnshardedClient(unshardedClient),
		WithShardedClient(shardedClient),
	)
	require.NoError(t, err)

	v2svc := MustRunServiceV2(mgr)

	functionID := uuid.New()
	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	testRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	testKey := fmt.Sprintf("test-key-%s", testRunID.String())

	testEvent := map[string]any{
		"name": "test.event",
		"data": map[string]any{
			"user_id": "123",
			"action":  "clicked",
		},
		"id": eventID.String(),
	}

	eventBytes, err := json.Marshal(testEvent)
	require.NoError(t, err)

	t.Run("Create method functionality", func(t *testing.T) {
		stepData1 := map[string]any{"result": "step1_output", "count": 42}
		stepData2 := map[string]any{"result": "step2_output", "status": "completed"}

		v2Input := statev2.CreateState{
			Metadata: statev2.Metadata{
				ID: statev2.ID{
					RunID:      testRunID,
					FunctionID: functionID,
					Tenant: statev2.Tenant{
						AccountID: accountID,
						EnvID:     workspaceID,
						AppID:     appID,
					},
				},
				Config: *statev2.InitConfig(&statev2.Config{
					Context:         map[string]any{"test": "context", "user_id": "123", "trace": "abc123"},
					SpanID:          "test-span-id",
					EventIDs:        []ulid.ULID{eventID},
					Idempotency:     testKey,
					FunctionVersion: 42,
					RequestVersion:  1,
					HasAI:           true,
					ForceStepPlan:   true,
					PriorityFactor:  int64Ptr(100),
					CustomConcurrencyKeys: []statev2.CustomConcurrency{
						{
							Key:   "f:" + functionID.String() + ":user-123",
							Hash:  "hash123",
							Limit: 5,
						},
						{
							Key:   "a:" + accountID.String() + ":account-limit",
							Hash:  "hash456",
							Limit: 10,
						},
					},
				}),
			},
			Events: []json.RawMessage{eventBytes},
			Steps: []state.MemoizedStep{
				{
					ID:   "step-1",
					Data: stepData1,
				},
				{
					ID:   "step-2",
					Data: stepData2,
				},
			},
		}

		createdState, err := v2svc.Create(ctx, v2Input)
		require.NoError(t, err)
		assert.NotZero(t, createdState.Metadata.ID.RunID)
		assert.Equal(t, functionID, createdState.Metadata.ID.FunctionID)
		assert.Equal(t, 2, len(createdState.Steps))
		assert.Nil(t, createdState.Metadata.Stack)

		t.Run("LoadState returns created state", func(t *testing.T) {
			loadedState, err := v2svc.LoadState(ctx, createdState.Metadata.ID)
			require.NoError(t, err)
			assert.Equal(t, createdState.Metadata.ID, loadedState.Metadata.ID)
			assert.Equal(t, len(createdState.Steps), len(loadedState.Steps))
		})

		t.Run("LoadEvents returns events", func(t *testing.T) {
			events, err := v2svc.LoadEvents(ctx, createdState.Metadata.ID)
			require.NoError(t, err)
			assert.Equal(t, 1, len(events))
		})

		t.Run("LoadMetadata returns metadata", func(t *testing.T) {
			metadata, err := v2svc.LoadMetadata(ctx, createdState.Metadata.ID)
			require.NoError(t, err)
			assert.Equal(t, createdState.Metadata.ID, metadata.ID)
			assert.Equal(t, testKey, metadata.Config.Idempotency)
		})

		t.Run("Exists returns true for existing state", func(t *testing.T) {
			exists, err := v2svc.Exists(ctx, createdState.Metadata.ID)
			require.NoError(t, err)
			assert.True(t, exists)
		})

		t.Run("UpdateMetadata works", func(t *testing.T) {
			config := statev2.MutableConfig{
				StartedAt:      time.Now(),
				RequestVersion: 2,
				ForceStepPlan:  true,
			}

			err := v2svc.UpdateMetadata(ctx, createdState.Metadata.ID, config)
			require.NoError(t, err)

			// Verify update by loading metadata
			metadata, err := v2svc.LoadMetadata(ctx, createdState.Metadata.ID)
			require.NoError(t, err)
			assert.Equal(t, int(2), metadata.Config.RequestVersion)
		})

		t.Run("SaveStep works", func(t *testing.T) {
			stepData := json.RawMessage(`{"result": "saved_step_output", "timestamp": "2023-01-01T00:00:00Z"}`)

			hasPending, err := v2svc.SaveStep(ctx, createdState.Metadata.ID, "new-step", stepData)
			require.NoError(t, err)
			assert.False(t, hasPending) // Should be false as we're not setting up pending steps
		})

		t.Run("SavePending works", func(t *testing.T) {
			pendingSteps := []string{"pending-step-1", "pending-step-2"}

			err := v2svc.SavePending(ctx, createdState.Metadata.ID, pendingSteps)
			require.NoError(t, err)
		})

		t.Run("Delete works", func(t *testing.T) {
			deleted, err := v2svc.Delete(ctx, createdState.Metadata.ID)
			require.NoError(t, err)
			assert.True(t, deleted)

			// Verify deletion
			exists, err := v2svc.Exists(ctx, createdState.Metadata.ID)
			require.NoError(t, err)
			assert.False(t, exists)
		})
	})

	t.Run("Error cases", func(t *testing.T) {
		nonExistentID := statev2.ID{
			RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
			FunctionID: functionID,
			Tenant: statev2.Tenant{
				AccountID: accountID,
				EnvID:     workspaceID,
				AppID:     appID,
			},
		}

		t.Run("LoadState with non-existent ID returns error", func(t *testing.T) {
			_, err := v2svc.LoadState(ctx, nonExistentID)
			assert.Error(t, err)
		})

		t.Run("LoadEvents with non-existent ID returns error", func(t *testing.T) {
			_, err := v2svc.LoadEvents(ctx, nonExistentID)
			assert.Error(t, err)
		})

		t.Run("LoadMetadata with non-existent ID returns error", func(t *testing.T) {
			_, err := v2svc.LoadMetadata(ctx, nonExistentID)
			assert.Error(t, err)
		})

		t.Run("Exists returns false for non-existent ID", func(t *testing.T) {
			exists, err := v2svc.Exists(ctx, nonExistentID)
			require.NoError(t, err)
			assert.False(t, exists)
		})

		t.Run("UpdateMetadata with non-existent ID succeeds (no-op)", func(t *testing.T) {
			config := statev2.MutableConfig{
				StartedAt:      time.Now(),
				RequestVersion: 2,
				ForceStepPlan:  true,
			}

			err := v2svc.UpdateMetadata(ctx, nonExistentID, config)
			assert.NoError(t, err) // Should be no-op, not error
		})

		t.Run("SaveStep with non-existent ID succeeds (no-op)", func(t *testing.T) {
			stepData := json.RawMessage(`{"result": "test"}`)

			result, err := v2svc.SaveStep(ctx, nonExistentID, "test-step", stepData)
			assert.NoError(t, err) // Should be no-op, not error
			assert.False(t, result)
		})

		t.Run("SavePending with non-existent ID succeeds (no-op)", func(t *testing.T) {
			pendingSteps := []string{"step1", "step2"}

			err := v2svc.SavePending(ctx, nonExistentID, pendingSteps)
			assert.NoError(t, err) // Should be no-op, not error
		})

		t.Run("Delete with non-existent ID succeeds", func(t *testing.T) {
			_, err := v2svc.Delete(ctx, nonExistentID)
			assert.NoError(t, err)
			// Delete may return true even for non-existent IDs in this implementation
			// The actual behavior depends on the underlying Redis operations
		})

		t.Run("Create with duplicate idempotency key returns ErrIdentifierExists", func(t *testing.T) {
			duplicateRunID := ulid.MustNew(ulid.Now(), rand.Reader)
			duplicateKey := fmt.Sprintf("duplicate-key-%s", duplicateRunID.String())

			v2Input := statev2.CreateState{
				Metadata: statev2.Metadata{
					ID: statev2.ID{
						RunID:      duplicateRunID,
						FunctionID: functionID,
						Tenant: statev2.Tenant{
							AccountID: accountID,
							EnvID:     workspaceID,
							AppID:     appID,
						},
					},
					Config: *statev2.InitConfig(&statev2.Config{
						Context:         map[string]any{"test": "context"},
						SpanID:          "test-span-id",
						EventIDs:        []ulid.ULID{eventID},
						Idempotency:     duplicateKey,
						FunctionVersion: 42,
						RequestVersion:  1,
					}),
				},
				Events: []json.RawMessage{eventBytes},
			}

			// Create first time should succeed
			_, err1 := v2svc.Create(ctx, v2Input)
			require.NoError(t, err1)

			// Create second time with same idempotency key should return error
			_, err2 := v2svc.Create(ctx, v2Input)
			assert.ErrorIs(t, err2, state.ErrIdentifierExists)
		})

		t.Run("SaveStep with duplicate step ID returns ErrDuplicateResponse", func(t *testing.T) {
			freshRunID := ulid.MustNew(ulid.Now(), rand.Reader)
			freshKey := fmt.Sprintf("fresh-key-%s", freshRunID.String())

			freshInput := statev2.CreateState{
				Metadata: statev2.Metadata{
					ID: statev2.ID{
						RunID:      freshRunID,
						FunctionID: functionID,
						Tenant: statev2.Tenant{
							AccountID: accountID,
							EnvID:     workspaceID,
							AppID:     appID,
						},
					},
					Config: *statev2.InitConfig(&statev2.Config{
						Context:         map[string]any{"test": "context"},
						SpanID:          "test-span-id",
						EventIDs:        []ulid.ULID{eventID},
						Idempotency:     freshKey,
						FunctionVersion: 42,
						RequestVersion:  1,
					}),
				},
				Events: []json.RawMessage{eventBytes},
			}

			createdState, err := v2svc.Create(ctx, freshInput)
			require.NoError(t, err)

			stepID := "duplicate-step"
			stepData1 := json.RawMessage(`{"result": "first_save", "timestamp": "2023-01-01T00:00:00Z"}`)
			stepData2 := json.RawMessage(`{"result": "second_save", "timestamp": "2023-01-01T00:01:00Z"}`)

			// Save step first time
			_, err1 := v2svc.SaveStep(ctx, createdState.Metadata.ID, stepID, stepData1)
			assert.NoError(t, err1)

			// Save step second time with different data should return duplicate error
			_, err2 := v2svc.SaveStep(ctx, createdState.Metadata.ID, stepID, stepData2)
			assert.ErrorIs(t, err2, state.ErrDuplicateResponse)
		})

		t.Run("SaveStep with same data succeeds (idempotent)", func(t *testing.T) {
			idempotentRunID := ulid.MustNew(ulid.Now(), rand.Reader)
			idempotentKey := fmt.Sprintf("idempotent-key-%s", idempotentRunID.String())

			idempotentInput := statev2.CreateState{
				Metadata: statev2.Metadata{
					ID: statev2.ID{
						RunID:      idempotentRunID,
						FunctionID: functionID,
						Tenant: statev2.Tenant{
							AccountID: accountID,
							EnvID:     workspaceID,
							AppID:     appID,
						},
					},
					Config: *statev2.InitConfig(&statev2.Config{
						Context:         map[string]any{"test": "context"},
						SpanID:          "test-span-id",
						EventIDs:        []ulid.ULID{eventID},
						Idempotency:     idempotentKey,
						FunctionVersion: 42,
						RequestVersion:  1,
					}),
				},
				Events: []json.RawMessage{eventBytes},
			}

			createdState, err := v2svc.Create(ctx, idempotentInput)
			require.NoError(t, err)

			stepID := "idempotent-step"
			stepData := json.RawMessage(`{"result": "same_data", "timestamp": "2023-01-01T00:00:00Z"}`)

			// Save step first time
			_, err1 := v2svc.SaveStep(ctx, createdState.Metadata.ID, stepID, stepData)
			assert.NoError(t, err1)

			// Save step second time with SAME data - should succeed (idempotent)
			_, err2 := v2svc.SaveStep(ctx, createdState.Metadata.ID, stepID, stepData)
			assert.NoError(t, err2) // Should not error for idempotent responses
		})
	})
}

func TestV2AdapterWithDisabledRetries(t *testing.T) {
	ctx := context.Background()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

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
		ctx,
		WithUnshardedClient(unshardedClient),
		WithShardedClient(shardedClient),
	)
	require.NoError(t, err)

	v2svc := MustRunServiceV2(mgr, WithDisabledRetries())

	functionID := uuid.New()
	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	testRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	testKey := fmt.Sprintf("test-key-no-retry-%s", testRunID.String())

	testEvent := map[string]any{
		"name": "test.event",
		"data": map[string]any{
			"user_id": "123",
			"action":  "clicked",
		},
		"id": eventID.String(),
	}

	eventBytes, err := json.Marshal(testEvent)
	require.NoError(t, err)

	t.Run("Basic functionality works with disabled retries", func(t *testing.T) {
		v2Input := statev2.CreateState{
			Metadata: statev2.Metadata{
				ID: statev2.ID{
					RunID:      testRunID,
					FunctionID: functionID,
					Tenant: statev2.Tenant{
						AccountID: accountID,
						EnvID:     workspaceID,
						AppID:     appID,
					},
				},
				Config: *statev2.InitConfig(&statev2.Config{
					Context:         map[string]any{"test": "context"},
					SpanID:          "test-span-id",
					EventIDs:        []ulid.ULID{eventID},
					Idempotency:     testKey,
					FunctionVersion: 42,
					RequestVersion:  1,
				}),
			},
			Events: []json.RawMessage{eventBytes},
		}

		createdState, err := v2svc.Create(ctx, v2Input)
		require.NoError(t, err)
		assert.NotZero(t, createdState.Metadata.ID.RunID)

		// Test that other operations still work
		exists, err := v2svc.Exists(ctx, createdState.Metadata.ID)
		require.NoError(t, err)
		assert.True(t, exists)

		// Test SaveStep works without retries
		stepData := json.RawMessage(`{"result": "no_retry_test"}`)
		_, err = v2svc.SaveStep(ctx, createdState.Metadata.ID, "no-retry-step", stepData)
		require.NoError(t, err)
	})
}

func int64Ptr(v int64) *int64 {
	return &v
}