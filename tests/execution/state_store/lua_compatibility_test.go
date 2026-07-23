package state_store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/tests/execution/queue/helper"
	"github.com/inngest/inngest/tests/testutil"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// StateStoreLuaCompatibilityTestCase defines a test case for state store Lua compatibility
type StateStoreLuaCompatibilityTestCase struct {
	Name       string                // Test case name
	ValkeyOpts []helper.ValkeyOption // Optional Valkey configuration
}

// TestUpdateMetadataIsFieldEmpty tests that the is_field_empty function in updateMetadata.lua
// works correctly against Valkey.
func TestUpdateMetadataIsFieldEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	testCases := []StateStoreLuaCompatibilityTestCase{
		{
			Name: "Valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			setup := func(t *testing.T) state.Manager {
				container, err := helper.StartValkey(t, tc.ValkeyOpts...)
				require.NoError(t, err)
				t.Cleanup(func() { _ = container.Terminate(ctx) })

				client, err := helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
				require.NoError(t, err)
				t.Cleanup(func() { client.Close() })

				unsharded := redis_state.NewUnshardedClient(client, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
				sharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
					UnshardedClient:        unsharded,
					FunctionRunStateClient: client,
					BatchClient:            client,
					StateDefaultKey:        redis_state.StateDefaultKey,
					QueueDefaultKey:        redis_state.QueueDefaultKey,
					FnRunIsSharded:         redis_state.AlwaysShardOnRun,
				})
				pauseMgr := redis_state.NewPauseStore(unsharded)
				mgr, err := redis_state.New(ctx, redis_state.WithShardedClient(sharded), redis_state.WithPauseDeleter(pauseMgr))
				require.NoError(t, err)
				return mgr
			}

			t.Run("sat empty gets updated", func(t *testing.T) {
				mgr := setup(t)

				runID := ulid.Make()
				id := state.Identifier{
					AccountID:   uuid.New(),
					WorkspaceID: uuid.New(),
					AppID:       uuid.New(),
					WorkflowID:  uuid.New(),
					RunID:       runID,
				}

				_, err := mgr.New(ctx, state.Input{
					Identifier:     id,
					EventBatchData: []map[string]any{{"name": "test", "data": map[string]any{}}},
				})
				require.NoError(t, err)

				startedAt := time.Now()
				err = mgr.UpdateMetadata(ctx, id.AccountID, runID, state.MetadataUpdate{
					RequestVersion: 1,
					StartedAt:      startedAt,
				})
				require.NoError(t, err)

				metadata, err := mgr.Metadata(ctx, id.AccountID, runID)
				require.NoError(t, err)
				require.Equal(t, startedAt.UnixMilli(), metadata.StartedAt.UnixMilli())
			})

			t.Run("sat with value not updated", func(t *testing.T) {
				mgr := setup(t)

				runID := ulid.Make()
				id := state.Identifier{
					AccountID:   uuid.New(),
					WorkspaceID: uuid.New(),
					AppID:       uuid.New(),
					WorkflowID:  uuid.New(),
					RunID:       runID,
				}

				_, err := mgr.New(ctx, state.Input{
					Identifier:     id,
					EventBatchData: []map[string]any{{"name": "test", "data": map[string]any{}}},
				})
				require.NoError(t, err)

				firstStartedAt := time.Now()
				err = mgr.UpdateMetadata(ctx, id.AccountID, runID, state.MetadataUpdate{
					RequestVersion: 1,
					StartedAt:      firstStartedAt,
				})
				require.NoError(t, err)

				// Try to update with a different time - should NOT update
				secondStartedAt := firstStartedAt.Add(time.Hour)
				err = mgr.UpdateMetadata(ctx, id.AccountID, runID, state.MetadataUpdate{
					RequestVersion: 2,
					StartedAt:      secondStartedAt,
				})
				require.NoError(t, err)

				metadata, err := mgr.Metadata(ctx, id.AccountID, runID)
				require.NoError(t, err)
				require.Equal(t, firstStartedAt.UnixMilli(), metadata.StartedAt.UnixMilli(), "sat should NOT be updated when already set")
				require.Equal(t, 2, metadata.RequestVersion, "rv should be updated")
			})
		})
	}
}

func TestStateStoreLuaCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	ctx := context.Background()

	// Setup function that returns a state manager backed by Valkey.
	setupManager := func(t *testing.T) state.Manager {
		container, err := helper.StartValkey(t, helper.WithValkeyImage(testutil.ValkeyDefaultImage))
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = container.Terminate(ctx)
		})

		client, err := helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
		require.NoError(t, err)
		t.Cleanup(func() {
			client.Close()
		})

		// Create unsharded client for state management
		unsharded := redis_state.NewUnshardedClient(client, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
		sharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
			UnshardedClient:        unsharded,
			FunctionRunStateClient: client,
			BatchClient:            client,
			StateDefaultKey:        redis_state.StateDefaultKey,
			QueueDefaultKey:        redis_state.QueueDefaultKey,
			FnRunIsSharded:         redis_state.AlwaysShardOnRun,
		})
		pauseMgr := redis_state.NewPauseStore(unsharded)

		// Create state manager
		mgr, err := redis_state.New(ctx, redis_state.WithShardedClient(sharded), redis_state.WithPauseDeleter(pauseMgr))
		require.NoError(t, err)
		return mgr
	}

	t.Run("metadata cjson compatibility verification", func(t *testing.T) {
		accountID := uuid.New()
		workflowID := uuid.New()
		workspaceID := uuid.New()
		appID := uuid.New()
		runID := ulid.Make()

		valkeyMgr := setupManager(t)

		identifier := state.Identifier{
			AccountID:       accountID,
			WorkspaceID:     workspaceID,
			AppID:           appID,
			WorkflowID:      workflowID,
			WorkflowVersion: 5, // Use 5 specifically since this was the problematic value in the original error
			RunID:           runID,
		}

		batchData := []map[string]any{
			{
				"name": "test/valkey.metadata",
				"data": map[string]any{
					"testField":    "valkey_metadata_test",
					"numericValue": 42,         // Additional numeric data
					"floatValue":   3.14,       // Float that might affect cjson behavior
					"largeNumber":  1234567890, // Large number to test parsing limits
				},
				"id": ulid.Make().String(),
			},
		}

		input := state.Input{
			Identifier:     identifier,
			EventBatchData: batchData,
		}

		// Create state via Lua script (with cjson.decode)
		_, err := valkeyMgr.New(ctx, input)
		require.NoError(t, err, "Failed to create state on Valkey")

		// Get metadata - this exercises newRunMetadata parsing
		metadata, err := valkeyMgr.Metadata(ctx, accountID, runID)
		require.NoError(t, err, "Failed to get metadata from Valkey")

		// Comprehensive metadata validation
		require.NotNil(t, metadata, "Valkey metadata should not be nil")
		require.Equal(t, runID.String(), metadata.Identifier.RunID.String(), "Valkey RunID should match")
		require.Equal(t, identifier.WorkflowVersion, metadata.Identifier.WorkflowVersion, "Valkey WorkflowVersion should be preserved")
		require.Equal(t, accountID, metadata.Identifier.AccountID, "Valkey AccountID should match")
		require.Equal(t, workflowID, metadata.Identifier.WorkflowID, "Valkey WorkflowID should match")

		// Validate status is a valid enum (should be RunStatusScheduled = 5)
		require.Greater(t, int(metadata.Status), 0, "Status should be a positive value")
		require.LessOrEqual(t, int(metadata.Status), 10, "Status should be within reasonable enum range")
		require.Equal(t, 5, int(metadata.Status), "Status should be 5 (RunStatusScheduled) - the original problematic value")

		// Validate version
		require.GreaterOrEqual(t, metadata.Version, 0, "Version should be non-negative")

		// WorkflowVersion validation (this was set to 5 to test the original problematic value)
		require.Equal(t, 5, metadata.Identifier.WorkflowVersion, "WorkflowVersion should be 5 as set in test")

		// RunID / UUID format validation
		require.Len(t, metadata.Identifier.RunID.String(), 26, "RunID should be 26 characters (ULID format)")
		require.Len(t, metadata.Identifier.AccountID.String(), 36, "AccountID should be UUID format (36 chars with hyphens)")
		require.Len(t, metadata.Identifier.WorkflowID.String(), 36, "WorkflowID should be UUID format (36 chars with hyphens)")
	})

	t.Run("update metadata StartedAt field", func(t *testing.T) {
		accountID := uuid.New()
		runID := ulid.Make()

		container, err := helper.StartValkey(t, helper.WithValkeyImage(testutil.ValkeyDefaultImage))
		require.NoError(t, err)
		t.Cleanup(func() { _ = container.Terminate(ctx) })

		client, err := helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
		require.NoError(t, err)
		t.Cleanup(func() { client.Close() })

		unsharded := redis_state.NewUnshardedClient(client, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
		sharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
			UnshardedClient:        unsharded,
			FunctionRunStateClient: client,
			BatchClient:            client,
			StateDefaultKey:        redis_state.StateDefaultKey,
			QueueDefaultKey:        redis_state.QueueDefaultKey,
			FnRunIsSharded:         redis_state.AlwaysShardOnRun,
		})
		pauseMgr := redis_state.NewPauseStore(unsharded)
		mgr, err := redis_state.New(ctx, redis_state.WithShardedClient(sharded), redis_state.WithPauseDeleter(pauseMgr))
		require.NoError(t, err)

		kg := sharded.FunctionRunState().KeyGenerator()
		key := kg.RunMetadata(ctx, true, runID)

		client.Do(ctx, client.B().Hset().Key(key).FieldValue().
			FieldValue("die", "0").FieldValue("rv", "1").FieldValue("sat", "0").Build())

		newStartTime := time.Now()
		err = mgr.UpdateMetadata(ctx, accountID, runID, state.MetadataUpdate{
			StartedAt:      newStartTime,
			RequestVersion: 2,
		})
		require.NoError(t, err)

		satAfter, _ := client.Do(ctx, client.B().Hget().Key(key).Field("sat").Build()).ToString()
		rvAfter, _ := client.Do(ctx, client.B().Hget().Key(key).Field("rv").Build()).ToString()

		require.Equal(t, "2", rvAfter)
		require.NotEmpty(t, satAfter)
		require.NotEqual(t, "0", satAfter, "StartedAt is zero")
		require.NotEqual(t, "0.0", satAfter, "StartedAt is zero float")
	})
}

// TestSavePauseLuaCompatibility tests that the savePause Lua script works correctly
// against Valkey.
//
// The savePause script:
// - KEYS[1]: pauseKey - Main pause data key
// - KEYS[2]: pauseEvtKey - Event index key
// - KEYS[3]: pauseInvokeKey - Invoke correlation index
// - KEYS[4]: pauseSignalKey - Signal correlation index
// - KEYS[5]: keyPauseAddIdx - Sorted set for pause add timestamps
// - KEYS[6]: keyPauseExpIdx - Sorted set for pause expiration timestamps
// - KEYS[7]: keyRunPauses - Set of pauses for this run
// - KEYS[8]: keyPausesIdx - Global pause index
//
// Returns:
//
//	[1..N]: Successfully saved pause; returns # of pauses in AddIdx
//	-1: Pause already exists
func TestSavePauseLuaCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	testCases := []StateStoreLuaCompatibilityTestCase{
		{
			Name: "Valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			setup := func(t *testing.T) rueidis.Client {
				container, err := helper.StartValkey(t, tc.ValkeyOpts...)
				require.NoError(t, err)
				t.Cleanup(func() { _ = container.Terminate(ctx) })

				client, err := helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
				require.NoError(t, err)
				t.Cleanup(func() { client.Close() })

				return client
			}

			runSavePause := func(t *testing.T, rc rueidis.Client, keys []string, args []string) (int64, error) {
				script := redis_state.GetScript("savePause")
				require.NotNil(t, script, "savePause script should exist")

				return script.Exec(ctx, rc, keys, args).AsInt64()
			}

			t.Run("successfully saves new pause", func(t *testing.T) {
				rc := setup(t)

				// All keys need same hash tag for cluster mode
				keys := []string{
					"{sp1}:pause",
					"{sp1}:evt",
					"{sp1}:invoke",
					"{sp1}:signal",
					"{sp1}:addIdx",
					"{sp1}:expIdx",
					"{sp1}:runPauses",
					"{sp1}:pausesIdx",
				}
				pauseData := `{"id":"pause-1","data":"test"}`
				pauseID := "pause-1"
				event := "test/event"
				invokeCorrelationID := ""
				signalCorrelationID := ""
				extendedExpiry := "3600"
				nowUnixSeconds := fmt.Sprintf("%d", time.Now().Unix())
				canReplaceSignal := "0"

				args := []string{pauseData, pauseID, event, invokeCorrelationID, signalCorrelationID, extendedExpiry, nowUnixSeconds, canReplaceSignal}

				result, err := runSavePause(t, rc, keys, args)
				require.NoError(t, err)
				require.Equal(t, int64(1), result, "should return 1 (count of pauses in AddIdx)")

				// Verify pause was saved
				savedPause, err := rc.Do(ctx, rc.B().Get().Key("{sp1}:pause").Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, pauseData, savedPause)

				// Verify event index was populated
				eventPause, err := rc.Do(ctx, rc.B().Hget().Key("{sp1}:evt").Field(pauseID).Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, pauseData, eventPause)

				// Verify global index
				isMember, err := rc.Do(ctx, rc.B().Sismember().Key("{sp1}:pausesIdx").Member(pauseID).Build()).AsBool()
				require.NoError(t, err)
				require.True(t, isMember)
			})

			t.Run("returns -1 when pause already exists", func(t *testing.T) {
				rc := setup(t)

				keys := []string{
					"{sp2}:pause",
					"{sp2}:evt",
					"{sp2}:invoke",
					"{sp2}:signal",
					"{sp2}:addIdx",
					"{sp2}:expIdx",
					"{sp2}:runPauses",
					"{sp2}:pausesIdx",
				}
				pauseData := `{"id":"pause-2","data":"test"}`
				pauseID := "pause-2"
				event := "test/event"
				nowUnixSeconds := fmt.Sprintf("%d", time.Now().Unix())
				args := []string{pauseData, pauseID, event, "", "", "3600", nowUnixSeconds, "0"}

				// First save should succeed
				result1, err := runSavePause(t, rc, keys, args)
				require.NoError(t, err)
				require.Equal(t, int64(1), result1)

				// Second save should return -1
				result2, err := runSavePause(t, rc, keys, args)
				require.NoError(t, err)
				require.Equal(t, int64(-1), result2, "should return -1 when pause already exists")
			})

			t.Run("saves pause with invoke correlation ID", func(t *testing.T) {
				rc := setup(t)

				keys := []string{
					"{sp3}:pause",
					"{sp3}:evt",
					"{sp3}:invoke",
					"{sp3}:signal",
					"{sp3}:addIdx",
					"{sp3}:expIdx",
					"{sp3}:runPauses",
					"{sp3}:pausesIdx",
				}
				pauseData := `{"id":"pause-3","data":"test"}`
				pauseID := "pause-3"
				event := ""
				invokeCorrelationID := "invoke-correlation-123"
				nowUnixSeconds := fmt.Sprintf("%d", time.Now().Unix())
				args := []string{pauseData, pauseID, event, invokeCorrelationID, "", "3600", nowUnixSeconds, "0"}

				result, err := runSavePause(t, rc, keys, args)
				require.NoError(t, err)
				require.Equal(t, int64(1), result)

				// Verify invoke correlation was set
				invokeVal, err := rc.Do(ctx, rc.B().Hget().Key("{sp3}:invoke").Field(invokeCorrelationID).Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, pauseID, invokeVal)
			})

			t.Run("saves pause with signal correlation ID", func(t *testing.T) {
				rc := setup(t)

				keys := []string{
					"{sp4}:pause",
					"{sp4}:evt",
					"{sp4}:invoke",
					"{sp4}:signal",
					"{sp4}:addIdx",
					"{sp4}:expIdx",
					"{sp4}:runPauses",
					"{sp4}:pausesIdx",
				}
				pauseData := `{"id":"pause-4","data":"test"}`
				pauseID := "pause-4"
				signalCorrelationID := "signal-correlation-456"
				nowUnixSeconds := fmt.Sprintf("%d", time.Now().Unix())
				args := []string{pauseData, pauseID, "", "", signalCorrelationID, "3600", nowUnixSeconds, "0"}

				result, err := runSavePause(t, rc, keys, args)
				require.NoError(t, err)
				require.Equal(t, int64(1), result)

				// Verify signal correlation was set
				signalVal, err := rc.Do(ctx, rc.B().Hget().Key("{sp4}:signal").Field(signalCorrelationID).Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, pauseID, signalVal)
			})
		})
	}
}

// TestDeletePauseLuaCompatibility tests that the deletePause Lua script works correctly
// against Valkey.
//
// The deletePause script:
// - KEYS[1]: pauseKey - Main pause data key
// - KEYS[2]: pauseEventKey - Event index key
// - KEYS[3]: pauseInvokeKey - Invoke correlation index
// - KEYS[4]: pauseSignalKey - Signal correlation index
// - KEYS[5]: keyPauseAddIdx - Sorted set for pause add timestamps
// - KEYS[6]: keyPauseExpIdx - Sorted set for pause expiration timestamps
// - KEYS[7]: keyRunPauses - Set of pauses for this run
// - KEYS[8]: keyPausesIdx - Global pause index
// - KEYS[9]: keyPausesBlockIdx - Block index key
//
// Returns:
//
//	0: Successfully deleted
//	1: Pause not in buffer (race condition)
func TestDeletePauseLuaCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	testCases := []StateStoreLuaCompatibilityTestCase{
		{
			Name: "Valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			setup := func(t *testing.T) rueidis.Client {
				container, err := helper.StartValkey(t, tc.ValkeyOpts...)
				require.NoError(t, err)
				t.Cleanup(func() { _ = container.Terminate(ctx) })

				client, err := helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
				require.NoError(t, err)
				t.Cleanup(func() { client.Close() })

				return client
			}

			runDeletePause := func(t *testing.T, rc rueidis.Client, keys []string, args []string) int64 {
				script := redis_state.GetScript("deletePause")
				require.NotNil(t, script, "deletePause script should exist")

				val, err := script.Exec(ctx, rc, keys, args).AsInt64()
				require.NoError(t, err)
				return val
			}

			t.Run("successfully deletes existing pause", func(t *testing.T) {
				rc := setup(t)

				pauseID := "pause-del-1"
				pauseData := `{"id":"pause-del-1","data":"test"}`

				// Pre-populate pause data
				err := rc.Do(ctx, rc.B().Set().Key("{dp1}:pause").Value(pauseData).Build()).Error()
				require.NoError(t, err)
				_, err = rc.Do(ctx, rc.B().Hset().Key("{dp1}:evt").FieldValue().FieldValue(pauseID, pauseData).Build()).AsInt64()
				require.NoError(t, err)
				_, err = rc.Do(ctx, rc.B().Sadd().Key("{dp1}:pausesIdx").Member(pauseID).Build()).AsInt64()
				require.NoError(t, err)
				_, err = rc.Do(ctx, rc.B().Sadd().Key("{dp1}:runPauses").Member(pauseID).Build()).AsInt64()
				require.NoError(t, err)
				_, err = rc.Do(ctx, rc.B().Zadd().Key("{dp1}:addIdx").ScoreMember().ScoreMember(float64(time.Now().Unix()), pauseID).Build()).AsInt64()
				require.NoError(t, err)
				_, err = rc.Do(ctx, rc.B().Zadd().Key("{dp1}:expIdx").ScoreMember().ScoreMember(float64(time.Now().Add(time.Hour).Unix()), pauseID).Build()).AsInt64()
				require.NoError(t, err)

				keys := []string{
					"{dp1}:pause",
					"{dp1}:evt",
					"{dp1}:invoke",
					"{dp1}:signal",
					"{dp1}:addIdx",
					"{dp1}:expIdx",
					"{dp1}:runPauses",
					"{dp1}:pausesIdx",
					"{dp1}:blockIdx",
				}
				args := []string{pauseID, "", "", ""} // pauseID, invokeCorrelationId, signalCorrelationId, blockIdxValue

				result := runDeletePause(t, rc, keys, args)
				require.Equal(t, int64(0), result, "should return 0 when successfully deleted")

				// Verify pause was deleted
				exists, err := rc.Do(ctx, rc.B().Exists().Key("{dp1}:pause").Build()).AsInt64()
				require.NoError(t, err)
				require.Equal(t, int64(0), exists, "pause key should be deleted")

				// Verify removed from global index
				isMember, err := rc.Do(ctx, rc.B().Sismember().Key("{dp1}:pausesIdx").Member(pauseID).Build()).AsBool()
				require.NoError(t, err)
				require.False(t, isMember, "should be removed from global index")
			})

			t.Run("deletes pause with invoke correlation", func(t *testing.T) {
				rc := setup(t)

				pauseID := "pause-del-2"
				pauseData := `{"id":"pause-del-2","data":"test"}`
				invokeCorrelationID := "invoke-del-123"

				// Pre-populate
				err := rc.Do(ctx, rc.B().Set().Key("{dp2}:pause").Value(pauseData).Build()).Error()
				require.NoError(t, err)
				_, err = rc.Do(ctx, rc.B().Hset().Key("{dp2}:invoke").FieldValue().FieldValue(invokeCorrelationID, pauseID).Build()).AsInt64()
				require.NoError(t, err)

				keys := []string{
					"{dp2}:pause",
					"{dp2}:evt",
					"{dp2}:invoke",
					"{dp2}:signal",
					"{dp2}:addIdx",
					"{dp2}:expIdx",
					"{dp2}:runPauses",
					"{dp2}:pausesIdx",
					"{dp2}:blockIdx",
				}
				args := []string{pauseID, invokeCorrelationID, "", ""}

				result := runDeletePause(t, rc, keys, args)
				require.Equal(t, int64(0), result)

				// Verify invoke correlation was deleted
				exists, err := rc.Do(ctx, rc.B().Hexists().Key("{dp2}:invoke").Field(invokeCorrelationID).Build()).AsBool()
				require.NoError(t, err)
				require.False(t, exists, "invoke correlation should be deleted")
			})

			t.Run("deletes pause with signal correlation only if it matches", func(t *testing.T) {
				rc := setup(t)

				pauseID := "pause-del-3"
				pauseData := `{"id":"pause-del-3","data":"test"}`
				signalCorrelationID := "signal-del-456"

				// Pre-populate with matching signal
				err := rc.Do(ctx, rc.B().Set().Key("{dp3}:pause").Value(pauseData).Build()).Error()
				require.NoError(t, err)
				_, err = rc.Do(ctx, rc.B().Hset().Key("{dp3}:signal").FieldValue().FieldValue(signalCorrelationID, pauseID).Build()).AsInt64()
				require.NoError(t, err)

				keys := []string{
					"{dp3}:pause",
					"{dp3}:evt",
					"{dp3}:invoke",
					"{dp3}:signal",
					"{dp3}:addIdx",
					"{dp3}:expIdx",
					"{dp3}:runPauses",
					"{dp3}:pausesIdx",
					"{dp3}:blockIdx",
				}
				args := []string{pauseID, "", signalCorrelationID, ""}

				result := runDeletePause(t, rc, keys, args)
				require.Equal(t, int64(0), result)

				// Verify signal correlation was deleted
				exists, err := rc.Do(ctx, rc.B().Hexists().Key("{dp3}:signal").Field(signalCorrelationID).Build()).AsBool()
				require.NoError(t, err)
				require.False(t, exists, "signal correlation should be deleted")
			})

			t.Run("does not delete signal correlation if it belongs to different pause", func(t *testing.T) {
				rc := setup(t)

				pauseID := "pause-del-4"
				otherPauseID := "other-pause"
				pauseData := `{"id":"pause-del-4","data":"test"}`
				signalCorrelationID := "signal-del-789"

				// Pre-populate with signal pointing to a DIFFERENT pause
				err := rc.Do(ctx, rc.B().Set().Key("{dp4}:pause").Value(pauseData).Build()).Error()
				require.NoError(t, err)
				_, err = rc.Do(ctx, rc.B().Hset().Key("{dp4}:signal").FieldValue().FieldValue(signalCorrelationID, otherPauseID).Build()).AsInt64()
				require.NoError(t, err)

				keys := []string{
					"{dp4}:pause",
					"{dp4}:evt",
					"{dp4}:invoke",
					"{dp4}:signal",
					"{dp4}:addIdx",
					"{dp4}:expIdx",
					"{dp4}:runPauses",
					"{dp4}:pausesIdx",
					"{dp4}:blockIdx",
				}
				args := []string{pauseID, "", signalCorrelationID, ""}

				result := runDeletePause(t, rc, keys, args)
				require.Equal(t, int64(0), result)

				// Verify signal correlation was NOT deleted (belongs to different pause)
				signalVal, err := rc.Do(ctx, rc.B().Hget().Key("{dp4}:signal").Field(signalCorrelationID).Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, otherPauseID, signalVal, "signal correlation should NOT be deleted when it belongs to different pause")
			})

			t.Run("returns 1 when deleting non-existent pause with blockIdxValue", func(t *testing.T) {
				rc := setup(t)

				pauseID := "pause-del-5"

				keys := []string{
					"{dp5}:pause",
					"{dp5}:evt",
					"{dp5}:invoke",
					"{dp5}:signal",
					"{dp5}:addIdx",
					"{dp5}:expIdx",
					"{dp5}:runPauses",
					"{dp5}:pausesIdx",
					"{dp5}:blockIdx",
				}
				// Non-empty blockIdxValue triggers block deletion logic
				args := []string{pauseID, "", "", "block-value"}

				result := runDeletePause(t, rc, keys, args)
				require.Equal(t, int64(1), result, "should return 1 when pause not in buffer (race condition)")
			})
		})
	}
}
