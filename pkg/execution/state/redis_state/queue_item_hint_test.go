package redis_state

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestItemHintReadiness(t *testing.T) {
	for _, tc := range []struct {
		name                                       string
		backlog, future, paused, migrating, denied bool
	}{
		{name: "ready"}, {name: "backlog", backlog: true},
		{name: "future", future: true}, {name: "paused", paused: true},
		{name: "migrating", migrating: true}, {name: "denied", denied: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			clock := clockwork.NewFakeClock()
			fn, acct, env := uuid.New(), uuid.New(), uuid.New()
			opts := []osqueue.QueueOpt{
				osqueue.WithClock(clock),
				osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return tc.backlog }),
				osqueue.WithPartitionPausedGetter(func(context.Context, uuid.UUID) osqueue.PartitionPausedInfo {
					return osqueue.PartitionPausedInfo{Paused: tc.paused}
				}),
			}
			if tc.denied {
				opts = append(opts, osqueue.WithDenyQueueNames(fn.String()))
			}
			_, shard := newQueue(t, rc, opts...)
			at := clock.Now()
			if tc.future {
				at = at.Add(time.Hour)
			}
			item, err := shard.EnqueueItem(t.Context(), osqueue.QueueItem{
				ID: "hint-test", AtMS: at.UnixMilli(), FunctionID: fn, WorkspaceID: env,
				Data: osqueue.Item{Kind: osqueue.KindStart, WorkspaceID: env, Identifier: state.Identifier{
					WorkflowID: fn, AccountID: acct, WorkspaceID: env, RunID: ulid.Make(),
				}},
			}, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			if tc.migrating {
				until := clock.Now().Add(time.Minute)
				require.NoError(t, shard.SetFunctionMigrate(t.Context(), osqueue.Scope{FunctionID: fn}, &until))
			}
			loaded, err := shard.(osqueue.ItemHintShard).LoadReadyItem(t.Context(), item.ID)
			if tc.name == "ready" {
				require.NoError(t, err)
				require.Equal(t, item.ID, loaded.ID)
				lease, err := shard.Lease(t.Context(), item, time.Minute, clock.Now(), osqueue.LeaseRequireReady())
				require.NoError(t, err)
				require.NotNil(t, lease)
				_, err = shard.Lease(t.Context(), item, time.Minute, clock.Now(), osqueue.LeaseRequireReady())
				require.ErrorIs(t, err, osqueue.ErrQueueItemAlreadyLeased)
			} else {
				require.ErrorIs(t, err, osqueue.ErrQueueItemNotReady)
			}
			if tc.backlog || tc.future {
				// The mutation guard also rejects a stale or bypassed point read.
				_, err = shard.Lease(t.Context(), item, time.Minute, clock.Now(), osqueue.LeaseRequireReady())
				require.ErrorIs(t, err, osqueue.ErrQueueItemNotReady)
				stored, err := shard.LoadQueueItem(t.Context(), item.ID)
				require.NoError(t, err)
				require.Nil(t, stored.LeaseID)
			}
		})
	}
}
