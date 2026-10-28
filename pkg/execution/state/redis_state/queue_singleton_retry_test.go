package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestQueueSingletonConflictRetry(t *testing.T) {
	for _, backlog := range []bool{false, true} {
		t.Run(fmt.Sprintf("backlog=%t", backlog), func(t *testing.T) {
			r, rc := initRedis(t)
			defer r.Close()
			clock := clockwork.NewFakeClock()
			_, shard := newQueue(t, rc,
				osqueue.WithClock(clock),
				osqueue.WithAllowKeyQueues(func(_ context.Context, _ uuid.UUID, _, _ uuid.UUID) bool { return backlog }),
			)
			ctx := t.Context()
			kg := shard.Client().kg
			accountID, workspaceID, functionID := uuid.New(), uuid.New(), uuid.New()
			makeItem := func(id string) osqueue.QueueItem {
				return osqueue.QueueItem{
					ID: id, FunctionID: functionID, WorkspaceID: workspaceID,
					Data: osqueue.Item{
						Kind: osqueue.KindStart, WorkspaceID: workspaceID,
						Identifier: state.Identifier{
							AccountID: accountID, WorkspaceID: workspaceID, WorkflowID: functionID,
							RunID: ulid.MustNew(ulid.Now(), rand.Reader),
						},
						Singleton: &osqueue.Singleton{Key: "same-key"},
					},
				}
			}
			opts := osqueue.EnqueueOpts{PassthroughJobId: true}
			owner, err := shard.EnqueueItem(ctx, makeItem("owner"), clock.Now(), opts)
			require.NoError(t, err)
			candidate := makeItem("migrating-item")
			before := r.Dump()
			for attempt := 0; attempt < 2; attempt++ {
				_, err = shard.EnqueueItem(ctx, candidate, clock.Now(), opts)
				require.ErrorIs(t, err, osqueue.ErrQueueItemSingletonExists)
				require.Empty(t, r.HGet(kg.QueueItem(), candidate.ID))
				require.False(t, r.Exists(kg.SingletonRunKey(candidate.Data.Identifier.RunID.String())))
				require.Equal(t, before, r.Dump(), "rejected enqueue must not change Redis")
			}

			// Once the conflicting run finishes, the same job ID must enqueue
			// successfully, including ready/backlog membership and ownership.
			require.NoError(t, shard.Dequeue(ctx, owner))
			accepted, err := shard.EnqueueItem(ctx, candidate, clock.Now(), opts)
			require.NoError(t, err)
			require.Equal(t, candidate.ID, accepted.ID)
			require.Equal(t, accepted, getQueueItem(t, r, accepted.ID))
			if backlog {
				b := osqueue.ItemBacklog(ctx, accepted)
				require.True(t, hasMember(t, r, kg.BacklogSet(b.BacklogID), accepted.ID))
			} else {
				p := osqueue.ItemPartition(ctx, accepted)
				require.True(t, hasMember(t, r, partitionZsetKey(p, kg), accepted.ID))
			}
			lock, err := r.Get(kg.SingletonKey(candidate.Data.Singleton))
			require.NoError(t, err)
			require.Equal(t, candidate.Data.Identifier.RunID.String(), lock)
			runLock, err := r.Get(kg.SingletonRunKey(candidate.Data.Identifier.RunID.String()))
			require.NoError(t, err)
			require.Equal(t, kg.SingletonKey(candidate.Data.Singleton), runLock)

			// A real duplicate remains item-exists, not singleton-conflict,
			// and must not overwrite the payload or any queue membership.
			before = r.Dump()
			_, err = shard.EnqueueItem(ctx, candidate, clock.Now(), opts)
			require.ErrorIs(t, err, osqueue.ErrQueueItemExists)
			require.Equal(t, before, r.Dump())

			// The completed-item idempotency marker must still reject a retry
			// without recreating the item or claiming the released singleton.
			require.NoError(t, shard.Dequeue(ctx, accepted))
			before = r.Dump()
			_, err = shard.EnqueueItem(ctx, candidate, clock.Now(), opts)
			require.ErrorIs(t, err, osqueue.ErrQueueItemExists)
			require.Empty(t, r.HGet(kg.QueueItem(), candidate.ID))
			require.False(t, r.Exists(kg.SingletonKey(candidate.Data.Singleton)))
			require.Equal(t, before, r.Dump())
		})
	}
}
