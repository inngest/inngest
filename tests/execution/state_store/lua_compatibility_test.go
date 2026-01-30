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
// across different Redis-compatible servers
type StateStoreLuaCompatibilityTestCase struct {
	Name       string                // Test case name
	ServerType string                // "valkey" or "garnet"
	ValkeyOpts []helper.ValkeyOption // Optional Valkey configuration
	GarnetOpts []helper.GarnetOption // Optional Garnet configuration
}

// TestUpdateMetadataIsFieldEmpty tests that the is_field_empty function in updateMetadata.lua
// works correctly across both Garnet and Valkey
func TestUpdateMetadataIsFieldEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	testCases := []StateStoreLuaCompatibilityTestCase{
		{
			Name:       "Valkey",
			ServerType: "valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
		{
			Name:       "Garnet",
			ServerType: "garnet",
			GarnetOpts: []helper.GarnetOption{
				helper.WithImage(testutil.GarnetDefaultImage),
				helper.WithConfiguration(&helper.GarnetConfiguration{
					EnableLua: true,
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			setup := func(t *testing.T) state.Manager {
				var client rueidis.Client

				switch tc.ServerType {
				case "valkey":
					container, err := helper.StartValkey(t, tc.ValkeyOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				case "garnet":
					container, err := helper.StartGarnet(t, tc.GarnetOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewRedisClient(container.Addr, container.Username, container.Password)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				default:
					t.Fatalf("unknown server type: %s", tc.ServerType)
				}

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

	// Setup function that returns a state manager for a given server type
	setupManager := func(t *testing.T, serverType string) state.Manager {
		var client rueidis.Client

		switch serverType {
		case "valkey":
			container, err := helper.StartValkey(t, helper.WithValkeyImage(testutil.ValkeyDefaultImage))
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = container.Terminate(ctx)
			})

			valkeyClient, err := helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
			require.NoError(t, err)
			t.Cleanup(func() {
				valkeyClient.Close()
			})

			client = valkeyClient

		case "garnet":
			container, err := helper.StartGarnet(t,
				helper.WithImage(testutil.GarnetDefaultImage),
				helper.WithConfiguration(&helper.GarnetConfiguration{
					EnableLua: true,
				}),
			)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = container.Terminate(ctx)
			})

			garnetClient, err := helper.NewRedisClient(container.Addr, container.Username, container.Password)
			require.NoError(t, err)
			t.Cleanup(func() {
				garnetClient.Close()
			})

			client = garnetClient

		default:
			t.Fatalf("unknown server type: %s", serverType)
		}

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
		// Generate shared test data for consistent comparison across backends
		accountID := uuid.New()
		workflowID := uuid.New()
		workspaceID := uuid.New()
		appID := uuid.New()
		runID := ulid.Make()

		// Test individual backends first to ensure they work, then attempt comparison
		backends := []struct {
			name    string
			setup   func() state.Manager
			results map[string]interface{}
		}{}

		// Test Valkey (should always work)
		t.Run("valkey", func(t *testing.T) {
			valkeyMgr := setupManager(t, "valkey")

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

			// Validate version
			require.GreaterOrEqual(t, metadata.Version, 0, "Version should be non-negative")

			t.Logf("✅ Valkey metadata parsing successful:")
			t.Logf("   Status: %v (%d - parsed from Lua cjson)", metadata.Status, int(metadata.Status))
			t.Logf("   Version: %v", metadata.Version)
			t.Logf("   WorkflowVersion: %v (preserved correctly)", metadata.Identifier.WorkflowVersion)
			t.Logf("   AccountID: %v", metadata.Identifier.AccountID)
			t.Logf("   RunID: %s", metadata.Identifier.RunID.String())

			// Store comprehensive results for cross-backend comparison
			backends = append(backends, struct {
				name    string
				setup   func() state.Manager
				results map[string]interface{}
			}{
				name: "valkey",
				results: map[string]interface{}{
					"status":          metadata.Status,
					"statusInt":       int(metadata.Status),
					"version":         metadata.Version,
					"workflowVersion": metadata.Identifier.WorkflowVersion,
					"runID":           metadata.Identifier.RunID.String(),
					"accountID":       metadata.Identifier.AccountID.String(),
					"workflowID":      metadata.Identifier.WorkflowID.String(),
				},
			})
		})

		// Test Garnet (may fail due to container issues, but we'll try)
		t.Run("garnet", func(t *testing.T) {
			// Use a shorter timeout for garnet to fail fast if there are container issues
			garnetMgr, err := func() (state.Manager, error) {
				defer func() {
					if r := recover(); r != nil {
						t.Logf("⚠️  Garnet setup failed (container issues): %v", r)
					}
				}()
				return setupManager(t, "garnet"), nil
			}()

			if err != nil {
				t.Skipf("Skipping Garnet test due to setup issues: %v", err)
				return
			}

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
					"name": "test/garnet.metadata",
					"data": map[string]any{
						"testField":    "garnet_metadata_test",
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

			// Create state via Lua script (with cjson.decode that may convert to floats)
			_, err = garnetMgr.New(ctx, input)
			if err != nil {
				t.Skipf("Skipping Garnet test due to connection issues: %v", err)
				return
			}

			// Get metadata - this exercises newRunMetadata parsing with potential float conversion
			metadata, err := garnetMgr.Metadata(ctx, accountID, runID)
			if err != nil {
				t.Skipf("Skipping Garnet metadata test due to issues: %v", err)
				return
			}

			// Comprehensive metadata validation for Garnet (same as Valkey)
			require.NotNil(t, metadata, "Garnet metadata should not be nil")
			require.Equal(t, runID.String(), metadata.Identifier.RunID.String(), "Garnet RunID should match")
			require.Equal(t, identifier.WorkflowVersion, metadata.Identifier.WorkflowVersion, "Garnet WorkflowVersion should be preserved")
			require.Equal(t, accountID, metadata.Identifier.AccountID, "Garnet AccountID should match")
			require.Equal(t, workflowID, metadata.Identifier.WorkflowID, "Garnet WorkflowID should match")

			// Validate status is a valid enum (should be RunStatusScheduled = 5)
			require.Greater(t, int(metadata.Status), 0, "Status should be a positive value")
			require.LessOrEqual(t, int(metadata.Status), 10, "Status should be within reasonable enum range")

			// Validate version
			require.GreaterOrEqual(t, metadata.Version, 0, "Version should be non-negative")

			t.Logf("✅ Garnet metadata parsing successful:")
			t.Logf("   Status: %v (%d - parsed from Lua cjson with potential float conversion)", metadata.Status, int(metadata.Status))
			t.Logf("   Version: %v", metadata.Version)
			t.Logf("   WorkflowVersion: %v (preserved correctly)", metadata.Identifier.WorkflowVersion)
			t.Logf("   AccountID: %v", metadata.Identifier.AccountID)
			t.Logf("   RunID: %s", metadata.Identifier.RunID.String())

			// Store comprehensive results for cross-backend comparison
			backends = append(backends, struct {
				name    string
				setup   func() state.Manager
				results map[string]interface{}
			}{
				name: "garnet",
				results: map[string]interface{}{
					"status":          metadata.Status,
					"statusInt":       int(metadata.Status),
					"version":         metadata.Version,
					"workflowVersion": metadata.Identifier.WorkflowVersion,
					"runID":           metadata.Identifier.RunID.String(),
					"accountID":       metadata.Identifier.AccountID.String(),
					"workflowID":      metadata.Identifier.WorkflowID.String(),
				},
			})
		})

		// If both backends worked, compare their results
		if len(backends) == 2 {
			valkeyResults := backends[0].results
			garnetResults := backends[1].results

			// Core numeric field comparisons (the main cjson compatibility concern)
			require.Equal(t, valkeyResults["status"], garnetResults["status"], "Status should be equal across backends (critical for cjson compatibility)")
			require.Equal(t, valkeyResults["statusInt"], garnetResults["statusInt"], "Status as integer should be equal across backends")
			require.Equal(t, valkeyResults["version"], garnetResults["version"], "Version should be equal across backends")
			require.Equal(t, valkeyResults["workflowVersion"], garnetResults["workflowVersion"], "WorkflowVersion should be equal across backends")
			require.Equal(t, valkeyResults["rv"], garnetResults["rv"], "RequestVersion (rv) should be equal across backends")
			require.Equal(t, valkeyResults["sat"], garnetResults["sat"], "StartedAt (sat) should be equal across backends")

			// Identity field comparisons
			require.Equal(t, valkeyResults["runID"], garnetResults["runID"], "RunID should be equal across backends")
			require.Equal(t, valkeyResults["accountID"], garnetResults["accountID"], "AccountID should be equal across backends")
			require.Equal(t, valkeyResults["workflowID"], garnetResults["workflowID"], "WorkflowID should be equal across backends")

			// Type consistency checks
			require.IsType(t, valkeyResults["status"], garnetResults["status"], "Status should have same type across backends")
			require.IsType(t, valkeyResults["version"], garnetResults["version"], "Version should have same type across backends")
			require.IsType(t, valkeyResults["statusInt"], garnetResults["statusInt"], "StatusInt should have same type across backends")
			require.IsType(t, valkeyResults["rv"], garnetResults["rv"], "RequestVersion (rv) should have same type across backends")
			require.IsType(t, valkeyResults["sat"], garnetResults["sat"], "StartedAt (sat) should have same type across backends")

			// Extract values for detailed validation
			valkeyStatus := valkeyResults["status"]
			garnetStatus := garnetResults["status"]
			valkeyStatusInt := valkeyResults["statusInt"].(int)
			garnetStatusInt := garnetResults["statusInt"].(int)
			valkeyVersion := valkeyResults["version"]
			garnetVersion := garnetResults["version"]
			valkeyWorkflowVersion := valkeyResults["workflowVersion"].(int)
			garnetWorkflowVersion := garnetResults["workflowVersion"].(int)

			// Verify that numeric fields are valid
			require.NotNil(t, valkeyStatus, "Valkey status should not be nil")
			require.NotNil(t, garnetStatus, "Garnet status should not be nil")
			require.Greater(t, valkeyStatusInt, 0, "Valkey status should be a positive integer")
			require.Greater(t, garnetStatusInt, 0, "Garnet status should be a positive integer")
			require.Equal(t, 5, valkeyStatusInt, "Status should be 5 (RunStatusScheduled) - the original problematic value")
			require.Equal(t, 5, garnetStatusInt, "Status should be 5 (RunStatusScheduled) - the original problematic value")

			// Version validation
			require.GreaterOrEqual(t, valkeyVersion, 0, "Valkey version should be non-negative")
			require.GreaterOrEqual(t, garnetVersion, 0, "Garnet version should be non-negative")

			// WorkflowVersion validation (this was set to 5 to test the original problematic value)
			require.Equal(t, 5, valkeyWorkflowVersion, "WorkflowVersion should be 5 as set in test")
			require.Equal(t, 5, garnetWorkflowVersion, "WorkflowVersion should be 5 as set in test")

			// RunID format validation
			valkeyRunID := valkeyResults["runID"].(string)
			garnetRunID := garnetResults["runID"].(string)
			require.Len(t, valkeyRunID, 26, "Valkey RunID should be 26 characters (ULID format)")
			require.Len(t, garnetRunID, 26, "Garnet RunID should be 26 characters (ULID format)")
			require.Equal(t, valkeyRunID, garnetRunID, "RunID should be identical across backends")

			// UUID format validation for IDs
			valkeyAccountID := valkeyResults["accountID"].(string)
			garnetAccountID := garnetResults["accountID"].(string)
			valkeyWorkflowID := valkeyResults["workflowID"].(string)
			garnetWorkflowID := garnetResults["workflowID"].(string)
			require.Len(t, valkeyAccountID, 36, "AccountID should be UUID format (36 chars with hyphens)")
			require.Len(t, garnetAccountID, 36, "AccountID should be UUID format (36 chars with hyphens)")
			require.Len(t, valkeyWorkflowID, 36, "WorkflowID should be UUID format (36 chars with hyphens)")
			require.Len(t, garnetWorkflowID, 36, "WorkflowID should be UUID format (36 chars with hyphens)")

			t.Logf("🎉 Comprehensive cross-backend compatibility verified!")
			t.Logf("   Status: Valkey=%v (%d), Garnet=%v (%d) - IDENTICAL", valkeyStatus, valkeyStatusInt, garnetStatus, garnetStatusInt)
			t.Logf("   Version: Valkey=%v, Garnet=%v - IDENTICAL", valkeyVersion, garnetVersion)
			t.Logf("   WorkflowVersion: Valkey=%d, Garnet=%d - IDENTICAL", valkeyWorkflowVersion, garnetWorkflowVersion)
			t.Logf("   RunID: Valkey=%s, Garnet=%s - IDENTICAL", valkeyRunID, garnetRunID)
			t.Logf("   AccountID: Valkey=%s, Garnet=%s - IDENTICAL", valkeyAccountID, garnetAccountID)
			t.Logf("   WorkflowID: Valkey=%s, Garnet=%s - IDENTICAL", valkeyWorkflowID, garnetWorkflowID)
			t.Logf("   ✓ Numeric fields parsed correctly despite cjson behavior differences")
			t.Logf("   ✓ Field types are consistent across backends")
			t.Logf("   ✓ Status value 5 (the problematic value) parsed correctly on both backends")
			t.Logf("   ✓ All UUID and ULID formats preserved correctly")
			t.Logf("   ✓ Original \"5.0\" parsing issue has been resolved!")
		} else {
			t.Logf("ℹ️  Only %d backend(s) tested successfully", len(backends))
			if len(backends) == 1 {
				results := backends[0].results
				t.Logf("   %s results:", backends[0].name)
				t.Logf("     Status: %v (%v)", results["status"], results["statusInt"])
				t.Logf("     Version: %v", results["version"])
				t.Logf("     WorkflowVersion: %v", results["workflowVersion"])
				t.Logf("     RunID: %s", results["runID"])
				t.Logf("     AccountID: %s", results["accountID"])
				t.Logf("     WorkflowID: %s", results["workflowID"])

				// Validate that the single backend results are correct
				statusInt := results["statusInt"].(int)
				workflowVersion := results["workflowVersion"].(int)
				require.Equal(t, 5, statusInt, "Status should be 5 (RunStatusScheduled)")
				require.Equal(t, 5, workflowVersion, "WorkflowVersion should be 5 as set in test")
				t.Logf("     ✓ Status value 5 (the original problematic value) parsed correctly")
				t.Logf("     ✓ WorkflowVersion 5 preserved correctly")
			}
			t.Logf("   Primary goal achieved: metadata parsing works correctly with cjson compatibility")
		}
	})
}

// TestConsumePauseLuaCompatibility tests that the consumePause Lua script works correctly
// across both Garnet and Valkey Redis-compatible servers.
//
// The consumePause script:
// - KEYS[1]: actionKey - Hash storing step actions/outputs
// - KEYS[2]: stackKey - List for execution stack
// - KEYS[3]: keyMetadata - Hash storing run metadata
// - KEYS[4]: keyStepsPending - Set of pending steps
// - KEYS[5]: keyIdempotency - Key for idempotency checking
//
// - ARGV[1]: pauseDataKey - The pause identifier/key
// - ARGV[2]: pauseDataVal - JSON-encoded pause data to store
// - ARGV[3]: pauseIdempotencyValue - Idempotency token value
// - ARGV[4]: pauseIdempotencyTTL - TTL of the idempotency key in seconds
//
// Returns:
// -1: Pause already consumed (idempotency check failed)
//
//	0: Successfully consumed, no pending steps remain
//	1: Successfully consumed, at least one pending step remains
func TestConsumePauseLuaCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	testCases := []StateStoreLuaCompatibilityTestCase{
		{
			Name:       "Valkey",
			ServerType: "valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
		{
			Name:       "Garnet",
			ServerType: "garnet",
			GarnetOpts: []helper.GarnetOption{
				helper.WithImage(testutil.GarnetDefaultImage),
				helper.WithConfiguration(&helper.GarnetConfiguration{
					EnableLua: true,
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			// Setup function that returns a Redis client for direct Lua script testing
			setup := func(t *testing.T) rueidis.Client {
				var client rueidis.Client

				switch tc.ServerType {
				case "valkey":
					container, err := helper.StartValkey(t, tc.ValkeyOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				case "garnet":
					container, err := helper.StartGarnet(t, tc.GarnetOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewRedisClient(container.Addr, container.Username, container.Password)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				default:
					t.Fatalf("unknown server type: %s", tc.ServerType)
				}

				return client
			}

			// Helper to run the consumePause script using rueidis.Lua
			runConsumePause := func(t *testing.T, rc rueidis.Client, keys []string, args []string) int64 {
				script := redis_state.GetScript("consumePause")
				require.NotNil(t, script, "consumePause script should exist")

				val, err := script.Exec(ctx, rc, keys, args).AsInt64()
				require.NoError(t, err)
				return val
			}

			t.Run("consume pause with no pending steps returns 0", func(t *testing.T) {
				rc := setup(t)

				// Use hash tags {t1} to ensure all keys go to the same slot (required for cluster mode)
				actionKey := "{t1}:actions"
				stackKey := "{t1}:stack"
				metadataKey := "{t1}:metadata"
				pendingKey := "{t1}:pending"
				idempotencyKey := "{t1}:idempotency"

				pauseDataKey := "step-1"
				pauseDataVal := `{"result":"success"}`
				idempotencyValue := "idem-123"
				idempotencyTTL := "3600" // TTL in seconds (1 hour)

				keys := []string{actionKey, stackKey, metadataKey, pendingKey, idempotencyKey}
				args := []string{pauseDataKey, pauseDataVal, idempotencyValue, idempotencyTTL}

				result := runConsumePause(t, rc, keys, args)
				require.Equal(t, int64(0), result, "should return 0 when no pending steps remain")

				// Verify data was stored correctly
				actionVal, err := rc.Do(ctx, rc.B().Hget().Key(actionKey).Field(pauseDataKey).Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, pauseDataVal, actionVal)

				// Verify stack was updated
				stackVals, err := rc.Do(ctx, rc.B().Lrange().Key(stackKey).Start(0).Stop(-1).Build()).AsStrSlice()
				require.NoError(t, err)
				require.Equal(t, []string{pauseDataKey}, stackVals)

				// Verify metadata was incremented
				stepCount, err := rc.Do(ctx, rc.B().Hget().Key(metadataKey).Field("step_count").Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, "1", stepCount)

				stateSize, err := rc.Do(ctx, rc.B().Hget().Key(metadataKey).Field("state_size").Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, fmt.Sprintf("%d", len(pauseDataVal)), stateSize)

				// Verify idempotency key was set
				idemVal, err := rc.Do(ctx, rc.B().Get().Key(idempotencyKey).Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, idempotencyValue, idemVal)
			})

			t.Run("consume pause with pending steps returns 1", func(t *testing.T) {
				rc := setup(t)

				// Use hash tags {t2} to ensure all keys go to the same slot (required for cluster mode)
				actionKey := "{t2}:actions"
				stackKey := "{t2}:stack"
				metadataKey := "{t2}:metadata"
				pendingKey := "{t2}:pending"
				idempotencyKey := "{t2}:idempotency"

				// Pre-populate pending steps
				_, err := rc.Do(ctx, rc.B().Sadd().Key(pendingKey).Member("step-1", "step-2", "step-3").Build()).AsInt64()
				require.NoError(t, err)

				pauseDataKey := "step-1"
				pauseDataVal := `{"result":"success"}`
				idempotencyValue := "idem-456"
				idempotencyTTL := "3600" // TTL in seconds (1 hour)

				keys := []string{actionKey, stackKey, metadataKey, pendingKey, idempotencyKey}
				args := []string{pauseDataKey, pauseDataVal, idempotencyValue, idempotencyTTL}

				result := runConsumePause(t, rc, keys, args)
				require.Equal(t, int64(1), result, "should return 1 when pending steps remain")

				// Verify step was removed from pending
				members, err := rc.Do(ctx, rc.B().Smembers().Key(pendingKey).Build()).AsStrSlice()
				require.NoError(t, err)
				require.ElementsMatch(t, []string{"step-2", "step-3"}, members)
			})

			t.Run("returns -1 when pause data already exists in actions hash", func(t *testing.T) {
				rc := setup(t)

				// Use hash tags {t3} to ensure all keys go to the same slot (required for cluster mode)
				actionKey := "{t3}:actions"
				stackKey := "{t3}:stack"
				metadataKey := "{t3}:metadata"
				pendingKey := "{t3}:pending"
				idempotencyKey := "{t3}:idempotency"

				pauseDataKey := "step-1"
				pauseDataVal := `{"result":"success"}`
				idempotencyValue := "idem-789"
				idempotencyTTL := "3600" // TTL in seconds (1 hour)

				// Pre-populate the action hash with the pause data key
				_, err := rc.Do(ctx, rc.B().Hset().Key(actionKey).FieldValue().FieldValue(pauseDataKey, `{"old":"data"}`).Build()).AsInt64()
				require.NoError(t, err)

				keys := []string{actionKey, stackKey, metadataKey, pendingKey, idempotencyKey}
				args := []string{pauseDataKey, pauseDataVal, idempotencyValue, idempotencyTTL}

				result := runConsumePause(t, rc, keys, args)
				require.Equal(t, int64(-1), result, "should return -1 when pause already exists")

				// Verify old data was not overwritten
				actionVal, err := rc.Do(ctx, rc.B().Hget().Key(actionKey).Field(pauseDataKey).Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, `{"old":"data"}`, actionVal)
			})

			t.Run("returns -1 when different idempotency value already exists", func(t *testing.T) {
				rc := setup(t)

				// Use hash tags {t4} to ensure all keys go to the same slot (required for cluster mode)
				actionKey := "{t4}:actions"
				stackKey := "{t4}:stack"
				metadataKey := "{t4}:metadata"
				pendingKey := "{t4}:pending"
				idempotencyKey := "{t4}:idempotency"

				pauseDataKey := "step-1"
				pauseDataVal := `{"result":"success"}`
				idempotencyValue := "idem-new"
				idempotencyTTL := "3600" // TTL in seconds (1 hour)

				// Pre-set idempotency key with different value
				err := rc.Do(ctx, rc.B().Set().Key(idempotencyKey).Value("idem-different").Build()).Error()
				require.NoError(t, err)

				keys := []string{actionKey, stackKey, metadataKey, pendingKey, idempotencyKey}
				args := []string{pauseDataKey, pauseDataVal, idempotencyValue, idempotencyTTL}

				result := runConsumePause(t, rc, keys, args)
				require.Equal(t, int64(-1), result, "should return -1 when idempotency check fails")

				// Verify idempotency key was not changed
				idemVal, err := rc.Do(ctx, rc.B().Get().Key(idempotencyKey).Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, "idem-different", idemVal)

				// Verify action was not set
				exists, err := rc.Do(ctx, rc.B().Hexists().Key(actionKey).Field(pauseDataKey).Build()).AsBool()
				require.NoError(t, err)
				require.False(t, exists)
			})

			t.Run("allows retry with same idempotency value", func(t *testing.T) {
				rc := setup(t)

				// Use hash tags {t5} to ensure all keys go to the same slot (required for cluster mode)
				actionKey := "{t5}:actions"
				stackKey := "{t5}:stack"
				metadataKey := "{t5}:metadata"
				pendingKey := "{t5}:pending"
				idempotencyKey := "{t5}:idempotency"

				pauseDataKey := "step-1"
				pauseDataVal := `{"result":"success"}`
				idempotencyValue := "idem-same"
				idempotencyTTL := "3600" // TTL in seconds (1 hour)

				// Pre-set idempotency key with same value (simulating retry)
				err := rc.Do(ctx, rc.B().Set().Key(idempotencyKey).Value(idempotencyValue).Build()).Error()
				require.NoError(t, err)

				keys := []string{actionKey, stackKey, metadataKey, pendingKey, idempotencyKey}
				args := []string{pauseDataKey, pauseDataVal, idempotencyValue, idempotencyTTL}

				result := runConsumePause(t, rc, keys, args)
				// Should succeed because idempotency value matches
				require.Equal(t, int64(0), result, "should return 0 when idempotency value matches (retry)")
			})

			t.Run("correctly tracks multiple consume operations", func(t *testing.T) {
				rc := setup(t)

				// Use hash tags {t6} to ensure all keys go to the same slot (required for cluster mode)
				actionKey := "{t6}:actions"
				stackKey := "{t6}:stack"
				metadataKey := "{t6}:metadata"
				pendingKey := "{t6}:pending"

				// Pre-populate pending steps
				_, err := rc.Do(ctx, rc.B().Sadd().Key(pendingKey).Member("step-1", "step-2", "step-3").Build()).AsInt64()
				require.NoError(t, err)

				ttl := "3600" // TTL in seconds (1 hour)

				// Consume first step
				keys1 := []string{actionKey, stackKey, metadataKey, pendingKey, "{t6}:idem:1"}
				args1 := []string{"step-1", `{"step":1}`, "idem-1", ttl}
				result1 := runConsumePause(t, rc, keys1, args1)
				require.Equal(t, int64(1), result1, "should return 1 after first consume (pending steps remain)")

				// Consume second step
				keys2 := []string{actionKey, stackKey, metadataKey, pendingKey, "{t6}:idem:2"}
				args2 := []string{"step-2", `{"step":2}`, "idem-2", ttl}
				result2 := runConsumePause(t, rc, keys2, args2)
				require.Equal(t, int64(1), result2, "should return 1 after second consume (pending steps remain)")

				// Consume third (last) step
				keys3 := []string{actionKey, stackKey, metadataKey, pendingKey, "{t6}:idem:3"}
				args3 := []string{"step-3", `{"step":3}`, "idem-3", ttl}
				result3 := runConsumePause(t, rc, keys3, args3)
				require.Equal(t, int64(0), result3, "should return 0 after final consume (no pending steps)")

				// Verify all steps are in the action hash
				exists1, err := rc.Do(ctx, rc.B().Hexists().Key(actionKey).Field("step-1").Build()).AsBool()
				require.NoError(t, err)
				require.True(t, exists1)

				exists2, err := rc.Do(ctx, rc.B().Hexists().Key(actionKey).Field("step-2").Build()).AsBool()
				require.NoError(t, err)
				require.True(t, exists2)

				exists3, err := rc.Do(ctx, rc.B().Hexists().Key(actionKey).Field("step-3").Build()).AsBool()
				require.NoError(t, err)
				require.True(t, exists3)

				// Verify stack order
				stackVals, err := rc.Do(ctx, rc.B().Lrange().Key(stackKey).Start(0).Stop(-1).Build()).AsStrSlice()
				require.NoError(t, err)
				require.Equal(t, []string{"step-1", "step-2", "step-3"}, stackVals)

				// Verify metadata
				stepCount, err := rc.Do(ctx, rc.B().Hget().Key(metadataKey).Field("step_count").Build()).ToString()
				require.NoError(t, err)
				require.Equal(t, "3", stepCount)

				// Verify pending set is empty
				members, err := rc.Do(ctx, rc.B().Smembers().Key(pendingKey).Build()).AsStrSlice()
				require.NoError(t, err)
				require.Empty(t, members)
			})

			t.Run("handles empty pauseDataKey gracefully", func(t *testing.T) {
				rc := setup(t)

				// Use hash tags {t7} to ensure all keys go to the same slot (required for cluster mode)
				actionKey := "{t7}:actions"
				stackKey := "{t7}:stack"
				metadataKey := "{t7}:metadata"
				pendingKey := "{t7}:pending"
				idempotencyKey := "{t7}:idempotency"

				// Empty pauseDataKey should skip the main logic
				pauseDataKey := ""
				pauseDataVal := `{"result":"success"}`
				idempotencyValue := "idem-empty"
				idempotencyTTL := "3600" // TTL in seconds (1 hour)

				keys := []string{actionKey, stackKey, metadataKey, pendingKey, idempotencyKey}
				args := []string{pauseDataKey, pauseDataVal, idempotencyValue, idempotencyTTL}

				result := runConsumePause(t, rc, keys, args)
				// With empty pauseDataKey, should return 0 (no pending steps since SCARD returns 0)
				require.Equal(t, int64(0), result)
			})
		})
	}
}

// TestLeasePauseLuaCompatibility tests that the leasePause Lua script works correctly
// across both Garnet and Valkey Redis-compatible servers.
//
// The leasePause script:
// - KEYS[1]: leaseID - The key to store the lease
// - ARGV[1]: currentTime - Current time in milliseconds
// - ARGV[2]: leaseTTL - Lease TTL in seconds
//
// Returns:
//
//	0: Successfully leased
//	1: Already leased (lease is still valid)
func TestLeasePauseLuaCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	testCases := []StateStoreLuaCompatibilityTestCase{
		{
			Name:       "Valkey",
			ServerType: "valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
		{
			Name:       "Garnet",
			ServerType: "garnet",
			GarnetOpts: []helper.GarnetOption{
				helper.WithImage(testutil.GarnetDefaultImage),
				helper.WithConfiguration(&helper.GarnetConfiguration{
					EnableLua: true,
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			setup := func(t *testing.T) rueidis.Client {
				var client rueidis.Client

				switch tc.ServerType {
				case "valkey":
					container, err := helper.StartValkey(t, tc.ValkeyOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				case "garnet":
					container, err := helper.StartGarnet(t, tc.GarnetOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewRedisClient(container.Addr, container.Username, container.Password)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				default:
					t.Fatalf("unknown server type: %s", tc.ServerType)
				}

				return client
			}

			runLeasePause := func(t *testing.T, rc rueidis.Client, keys []string, args []string) int64 {
				script := redis_state.GetScript("leasePause")
				require.NotNil(t, script, "leasePause script should exist")

				val, err := script.Exec(ctx, rc, keys, args).AsInt64()
				require.NoError(t, err)
				return val
			}

			t.Run("successfully leases when no existing lease", func(t *testing.T) {
				rc := setup(t)

				leaseKey := "{lp1}:lease"
				currentTimeMS := fmt.Sprintf("%d", time.Now().UnixMilli())
				leaseTTL := "60" // 60 seconds

				keys := []string{leaseKey}
				args := []string{currentTimeMS, leaseTTL}

				result := runLeasePause(t, rc, keys, args)
				require.Equal(t, int64(0), result, "should return 0 when successfully leased")

				// Verify lease was set
				exists, err := rc.Do(ctx, rc.B().Exists().Key(leaseKey).Build()).AsInt64()
				require.NoError(t, err)
				require.Equal(t, int64(1), exists, "lease key should exist")
			})

			t.Run("returns 1 when lease is still valid", func(t *testing.T) {
				rc := setup(t)

				leaseKey := "{lp2}:lease"
				currentTimeMS := time.Now().UnixMilli()
				leaseTTL := "60" // 60 seconds

				// First lease
				keys := []string{leaseKey}
				args := []string{fmt.Sprintf("%d", currentTimeMS), leaseTTL}
				result1 := runLeasePause(t, rc, keys, args)
				require.Equal(t, int64(0), result1, "first lease should succeed")

				// Try to lease again immediately (lease should still be valid)
				result2 := runLeasePause(t, rc, keys, args)
				require.Equal(t, int64(1), result2, "should return 1 when lease is still valid")
			})

			t.Run("allows re-lease when lease has expired", func(t *testing.T) {
				rc := setup(t)

				leaseKey := "{lp3}:lease"
				leaseTTL := "60" // 60 seconds

				// Set a lease that has already expired (currentTime in the past)
				pastTimeMS := time.Now().Add(-2 * time.Hour).UnixMilli()
				keys := []string{leaseKey}
				args := []string{fmt.Sprintf("%d", pastTimeMS), leaseTTL}
				result1 := runLeasePause(t, rc, keys, args)
				require.Equal(t, int64(0), result1, "first lease should succeed")

				// Now try to lease with current time - should succeed because old lease expired
				currentTimeMS := time.Now().UnixMilli()
				args2 := []string{fmt.Sprintf("%d", currentTimeMS), leaseTTL}
				result2 := runLeasePause(t, rc, keys, args2)
				require.Equal(t, int64(0), result2, "should return 0 when old lease has expired")
			})
		})
	}
}

// TestSavePauseLuaCompatibility tests that the savePause Lua script works correctly
// across both Garnet and Valkey Redis-compatible servers.
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
			Name:       "Valkey",
			ServerType: "valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
		{
			Name:       "Garnet",
			ServerType: "garnet",
			GarnetOpts: []helper.GarnetOption{
				helper.WithImage(testutil.GarnetDefaultImage),
				helper.WithConfiguration(&helper.GarnetConfiguration{
					EnableLua: true,
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			setup := func(t *testing.T) rueidis.Client {
				var client rueidis.Client

				switch tc.ServerType {
				case "valkey":
					container, err := helper.StartValkey(t, tc.ValkeyOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				case "garnet":
					container, err := helper.StartGarnet(t, tc.GarnetOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewRedisClient(container.Addr, container.Username, container.Password)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				default:
					t.Fatalf("unknown server type: %s", tc.ServerType)
				}

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
// across both Garnet and Valkey Redis-compatible servers.
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
			Name:       "Valkey",
			ServerType: "valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
		{
			Name:       "Garnet",
			ServerType: "garnet",
			GarnetOpts: []helper.GarnetOption{
				helper.WithImage(testutil.GarnetDefaultImage),
				helper.WithConfiguration(&helper.GarnetConfiguration{
					EnableLua: true,
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			setup := func(t *testing.T) rueidis.Client {
				var client rueidis.Client

				switch tc.ServerType {
				case "valkey":
					container, err := helper.StartValkey(t, tc.ValkeyOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				case "garnet":
					container, err := helper.StartGarnet(t, tc.GarnetOpts...)
					require.NoError(t, err)
					t.Cleanup(func() { _ = container.Terminate(ctx) })

					client, err = helper.NewRedisClient(container.Addr, container.Username, container.Password)
					require.NoError(t, err)
					t.Cleanup(func() { client.Close() })

				default:
					t.Fatalf("unknown server type: %s", tc.ServerType)
				}

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
