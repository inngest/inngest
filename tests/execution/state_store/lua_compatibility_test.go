package state_store

import (
	"context"
	"testing"

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

func TestStateStoreLuaCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	testCases := []StateStoreLuaCompatibilityTestCase{
		{
			Name:       "Valkey State Store",
			ServerType: "valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
		{
			Name:       "Garnet State Store",
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
				// Start the appropriate server based on test case
				var client rueidis.Client

				switch tc.ServerType {
				case "valkey":
					container, err := helper.StartValkey(t, tc.ValkeyOpts...)
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
					container, err := helper.StartGarnet(t, tc.GarnetOpts...)
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
					t.Fatalf("unknown server type: %s", tc.ServerType)
				}

				// Create unsharded client for state management
				unsharded := redis_state.NewUnshardedClient(client, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)

				// Create state manager
				mgr, err := redis_state.New(
					ctx,
					redis_state.WithUnshardedClient(unsharded),
					redis_state.WithShardedClient(redis_state.NewShardedClient(redis_state.ShardedClientOpts{
						UnshardedClient:        unsharded,
						FunctionRunStateClient: client,
						BatchClient:            client,
						StateDefaultKey:        redis_state.StateDefaultKey,
						QueueDefaultKey:        redis_state.QueueDefaultKey,
						FnRunIsSharded:         redis_state.AlwaysShardOnRun,
					})),
				)
				require.NoError(t, err)
				return mgr
			}

			serverType := tc.ServerType

			t.Run("metadata function cjson consistency", func(t *testing.T) {
				mgr := setup(t)

				// This test specifically targets the Metadata() function and newRunMetadata parsing
				// to test the exact scenario that caused "invalid function status stored in run metadata: \"5.0\""
				accountID := uuid.New()
				workflowID := uuid.New()
				runID := ulid.Make()

				identifier := state.Identifier{
					AccountID:       accountID,
					WorkspaceID:     uuid.New(),
					AppID:           uuid.New(),
					WorkflowID:      workflowID,
					WorkflowVersion: 1,
					RunID:           runID,
				}

				batchData := []map[string]any{
					{
						"name": "test/metadata.function",
						"data": map[string]any{
							"testField": "direct_metadata_test",
						},
						"id": ulid.Make().String(),
					},
				}

				input := state.Input{
					Identifier:     identifier,
					EventBatchData: batchData,
				}

				// Create state - this triggers the new.lua script which uses cjson.decode
				// The status field gets set to RunStatusScheduled (5) via the Lua script
				_, err := mgr.New(ctx, input)
				require.NoError(t, err, "Failed to create state on %s", serverType)

				// Directly call Metadata() function - this calls metadata() -> newRunMetadata()
				// This is the exact function that was failing with "invalid function status stored in run metadata: \"5.0\""
				metadata, err := mgr.Metadata(ctx, accountID, runID)
				require.NoError(t, err, "Metadata() should parse numeric fields correctly on %s", serverType)

				// Verify that the metadata was parsed correctly despite potential cjson float conversion
				require.NotNil(t, metadata, "Metadata should be loaded successfully on %s", serverType)
				require.Equal(t, runID.String(), metadata.Identifier.RunID.String(), "RunID should match on %s", serverType)
				require.Equal(t, identifier.WorkflowVersion, metadata.Identifier.WorkflowVersion, "WorkflowVersion should be preserved on %s", serverType)

				// The fact that Metadata() succeeds means newRunMetadata() parsed all numeric fields correctly:
				// - status (the problematic field that was "5.0" instead of "5")
				// - version, state_size, event_size, step_count
				// If the "5.0" issue were present, Metadata() would fail with:
				// "invalid function status stored in run metadata: \"5.0\""
			})
		})
	}
}