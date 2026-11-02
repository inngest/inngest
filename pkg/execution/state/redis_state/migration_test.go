package redis_state

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func migrationTestItem() osqueue.QueueItem {
	fn, env := uuid.New(), uuid.New()
	id := uuid.NewString()
	return osqueue.QueueItem{ID: id, FunctionID: fn, WorkspaceID: env, Data: osqueue.Item{
		JobID: &id, WorkspaceID: env, Kind: osqueue.KindSleep,
		Identifier: state.Identifier{AccountID: uuid.New(), WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()},
	}}
}

func TestItemsByPartitionSeq2ReportsReadFailure(t *testing.T) {
	for _, keyQueues := range []bool{false, true} {
		for _, afterItem := range []bool{false, true} {
			name := "partition"
			if keyQueues {
				name = "backlog"
			}
			if afterItem {
				name += "/after-item"
			} else {
				name += "/first-read"
			}
			t.Run(name, func(t *testing.T) {
				r, rc := initRedis(t)
				t.Cleanup(rc.Close)
				_, shard := newQueue(t, rc, osqueue.WithAllowKeyQueues(func(_ context.Context, _, _, _ uuid.UUID) bool { return keyQueues }))
				item := migrationTestItem()
				at := time.Now().Add(time.Hour)
				for i := range 2 {
					item.ID = uuid.NewString()
					_, err := shard.EnqueueItem(t.Context(), item, at.Add(time.Duration(i)*time.Millisecond), osqueue.EnqueueOpts{PassthroughJobId: true})
					require.NoError(t, err)
				}
				items, err := shard.ItemsByPartitionSeq2(t.Context(), osqueue.ScopeFromQueueItem(item), item.FunctionID.String(), time.Time{}, at.Add(time.Minute), osqueue.WithQueueItemIterBatchSize(1), osqueue.WithQueueItemIterInterval(0))
				require.NoError(t, err)
				if !afterItem {
					r.SetError("ERR injected migration read failure")
				}
				defer r.SetError("")
				var seen int
				var readErr error
				for _, err := range items {
					if err != nil {
						readErr = err
						break
					}
					seen++
					r.SetError("ERR injected migration read failure")
				}
				require.ErrorContains(t, readErr, "injected migration read failure")
				if afterItem {
					require.Equal(t, 1, seen)
				} else {
					require.Zero(t, seen)
				}
			})
		}
	}
}

func TestEnqueueSingletonMigrationIsAtomic(t *testing.T) {
	r, rc := initRedis(t)
	t.Cleanup(rc.Close)
	_, shard := newQueue(t, rc)
	ctx := t.Context()
	item := migrationTestItem()
	opts := osqueue.EnqueueOpts{PassthroughJobId: true, SingletonMigrationKey: "owned-key"}
	stored, err := shard.EnqueueItem(ctx, item, time.Now().Add(time.Hour), opts)
	require.NoError(t, err)
	require.Nil(t, stored.Data.Singleton, "continuation metadata must not become a start item")
	key, err := shard.SingletonMigrationKey(ctx, stored)
	require.NoError(t, err)
	require.Equal(t, "owned-key", key)
	second := item
	second.ID = uuid.NewString()
	_, err = shard.EnqueueItem(ctx, second, time.Now().Add(time.Hour), opts)
	require.NoError(t, err, "another continuation of the owning run is allowed")
	competitor := item
	competitor.ID = uuid.NewString()
	competitor.Data.Identifier.RunID = ulid.Make()
	_, err = shard.EnqueueItem(ctx, competitor, time.Now(), opts)
	require.ErrorIs(t, err, osqueue.ErrQueueItemSingletonExists)
	_, err = shard.LoadQueueItem(ctx, competitor.ID)
	require.ErrorIs(t, err, osqueue.ErrQueueItemNotFound)
	failed := migrationTestItem()
	require.NoError(t, r.Set(shard.Client().KeyGenerator().Idempotency(failed.ID), ""))
	_, err = shard.EnqueueItem(ctx, failed, time.Now(), osqueue.EnqueueOpts{PassthroughJobId: true, SingletonMigrationKey: "must-not-appear"})
	require.ErrorIs(t, err, osqueue.ErrQueueItemExists)
	holder, err := shard.SingletonGetRunID(ctx, osqueue.ScopeFromQueueItem(failed), "must-not-appear")
	require.NoError(t, err)
	require.Nil(t, holder, "a rejected enqueue must not acquire an orphan lock")
	// A stale reverse mapping must not transfer a different run's ownership.
	require.NoError(t, r.Set(shard.Client().KeyGenerator().SingletonKey(&osqueue.Singleton{Key: "owned-key"}), ulid.Make().String()))
	key, err = shard.SingletonMigrationKey(ctx, item)
	require.NoError(t, err)
	require.Empty(t, key)
}
