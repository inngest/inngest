package state_store

import (
	"testing"

	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/execution/state/testharness"
	"github.com/inngest/inngest/tests/execution/queue/helper"
	"github.com/inngest/inngest/tests/testutil"
	"github.com/stretchr/testify/require"
)

func TestValkeyStateHarness(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	for _, tc := range []struct {
		name    string
		sharded redis_state.IsShardedWithRunIdFn
	}{
		{name: "unsharded", sharded: redis_state.NeverShardOnRun},
		{name: "sharded", sharded: redis_state.AlwaysShardOnRun},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			container, err := helper.StartValkey(t, helper.WithValkeyImage(testutil.ValkeyDefaultImage))
			require.NoError(t, err)

			client, err := helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
			require.NoError(t, err)
			t.Cleanup(client.Close)

			unsharded := redis_state.NewUnshardedClient(client, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
			sharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
				UnshardedClient:        unsharded,
				FunctionRunStateClient: client,
				BatchClient:            client,
				StateDefaultKey:        redis_state.StateDefaultKey,
				QueueDefaultKey:        redis_state.QueueDefaultKey,
				FnRunIsSharded:         tc.sharded,
			})
			mgr, err := redis_state.New(ctx,
				redis_state.WithShardedClient(sharded),
				redis_state.WithPauseDeleter(redis_state.NewPauseStore(unsharded)),
			)
			require.NoError(t, err)

			testharness.CheckState(t, func() (state.Manager, func()) {
				return mgr, func() {
					require.NoError(t, client.Do(ctx, client.B().Flushdb().Build()).Error())
				}
			})
		})
	}
}
