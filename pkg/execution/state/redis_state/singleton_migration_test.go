package redis_state

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestSingletonMigrationSourceDrained(t *testing.T) {
	r, rc := initRedis(t)
	t.Cleanup(rc.Close)
	_, shard := newQueue(t, rc)
	scope := osqueue.ScopeFromQueueItem(migrationTestItem())
	checker := shard.(interface {
		SingletonMigrationSourceDrained(context.Context, osqueue.Scope) (bool, error)
	})
	kg := shard.Client().kg
	for _, key := range []string{
		kg.PartitionQueueSet(enums.PartitionTypeDefault, scope.FunctionID.String(), ""),
		kg.PartitionScavengerIndex(scope.FunctionID.String()),
		kg.ShadowPartitionSet(scope.FunctionID.String()),
	} {
		// A past score must not hide an expired lease or stale backlog pointer.
		_, err := r.ZAdd(key, 1, "outstanding")
		require.NoError(t, err)
		drained, err := checker.SingletonMigrationSourceDrained(t.Context(), scope)
		require.NoError(t, err)
		require.False(t, drained)
		r.Del(key)
	}
	drained, err := checker.SingletonMigrationSourceDrained(t.Context(), scope)
	require.NoError(t, err)
	require.True(t, drained)
}

func TestSingletonMigrationHoldSurvivesLocalCompletionAndExpiry(t *testing.T) {
	r, rc := initRedis(t)
	t.Cleanup(rc.Close)
	_, shard := newQueue(t, rc)
	hold := shard.(osqueue.SingletonMigrationOperations)
	item := migrationTestItem()
	scope := osqueue.ScopeFromQueueItem(item)
	id := ulid.Make().String()
	ctx := t.Context()
	require.NoError(t, hold.BeginSingletonMigration(ctx, scope, id))
	require.NoError(t, hold.BeginSingletonMigration(ctx, scope, id))
	require.ErrorIs(t, hold.BeginSingletonMigration(ctx, scope, "other"), osqueue.ErrSingletonMigrationConflict)
	stored, err := shard.EnqueueItem(ctx, item, time.Now(), osqueue.EnqueueOpts{PassthroughJobId: true, SingletonMigrationKey: "key"})
	require.NoError(t, err)
	require.NoError(t, shard.Dequeue(ctx, stored))
	// A dead migrator must not turn its scanner-lock expiry into a second run.
	r.FastForward(24 * time.Hour)
	owner, err := shard.SingletonGetRunID(ctx, scope, "key")
	require.NoError(t, err)
	require.Equal(t, &item.Data.Identifier.RunID, owner)
	competitor := item
	competitor.ID = uuid.NewString()
	competitor.Data.Identifier.RunID = ulid.Make()
	_, err = shard.EnqueueItem(ctx, competitor, time.Now(), osqueue.EnqueueOpts{PassthroughJobId: true, SingletonMigrationKey: "key"})
	require.ErrorIs(t, err, osqueue.ErrQueueItemSingletonExists)
	done, err := hold.FinishSingletonMigration(ctx, scope, "other")
	require.ErrorIs(t, err, osqueue.ErrSingletonMigrationConflict)
	require.False(t, done)
	done, err = hold.FinishSingletonMigration(ctx, scope, id)
	require.NoError(t, err)
	require.True(t, done)
	done, err = hold.FinishSingletonMigration(ctx, scope, id)
	require.NoError(t, err)
	require.True(t, done, "a lost finish acknowledgement must be retryable")
	owner, err = shard.SingletonGetRunID(ctx, scope, "key")
	require.NoError(t, err)
	require.Nil(t, owner)
	_, err = shard.EnqueueItem(ctx, competitor, time.Now(), osqueue.EnqueueOpts{PassthroughJobId: true, SingletonMigrationKey: "key"})
	require.NoError(t, err)
}

func TestSingletonMigrationFinishRechecksOwnershipAndRemainingJobs(t *testing.T) {
	for _, stillRunning := range []bool{false, true} {
		name := "new-owner"
		if stillRunning {
			name = "more-jobs"
		}
		t.Run(name, func(t *testing.T) {
			_, rc := initRedis(t)
			t.Cleanup(rc.Close)
			_, shard := newQueue(t, rc)
			hold := shard.(osqueue.SingletonMigrationOperations)
			item := migrationTestItem()
			scope := osqueue.ScopeFromQueueItem(item)
			ctx := t.Context()
			require.NoError(t, hold.BeginSingletonMigration(ctx, scope, "migration"))
			opts := osqueue.EnqueueOpts{PassthroughJobId: true, SingletonMigrationKey: "key"}
			stored, err := shard.EnqueueItem(ctx, item, time.Now(), opts)
			require.NoError(t, err)
			require.NoError(t, shard.Dequeue(ctx, stored))
			expected := item.Data.Identifier.RunID
			if stillRunning {
				item.ID = uuid.NewString()
				_, err = shard.EnqueueItem(ctx, item, time.Now(), opts)
				require.NoError(t, err)
			} else {
				// Explicit cancellation is allowed to replace the owner; cleanup
				// for the migrating run must not delete its replacement's lock.
				_, err = shard.SingletonReleaseRunID(ctx, scope, "key")
				require.NoError(t, err)
				item.ID = uuid.NewString()
				item.Data.Identifier.RunID = ulid.Make()
				expected = item.Data.Identifier.RunID
				_, err = shard.EnqueueItem(ctx, item, time.Now(), opts)
				require.NoError(t, err)
			}
			done, err := hold.FinishSingletonMigration(ctx, scope, "migration")
			require.NoError(t, err)
			require.True(t, done)
			owner, err := shard.SingletonGetRunID(ctx, scope, "key")
			require.NoError(t, err)
			require.Equal(t, &expected, owner)
			loaded, err := shard.LoadQueueItem(ctx, item.ID)
			require.NoError(t, err)
			require.NoError(t, shard.Dequeue(ctx, *loaded))
			owner, err = shard.SingletonGetRunID(ctx, scope, "key")
			require.NoError(t, err)
			require.Nil(t, owner, "ordinary completion resumes after migration")
		})
	}
}

func TestSingletonMigrationCleanupIsBoundedAndFenced(t *testing.T) {
	r, rc := initRedis(t)
	t.Cleanup(rc.Close)
	_, shard := newQueue(t, rc)
	hold := shard.(osqueue.SingletonMigrationOperations)
	item := migrationTestItem()
	scope := osqueue.ScopeFromQueueItem(item)
	ctx := t.Context()
	require.NoError(t, hold.BeginSingletonMigration(ctx, scope, "first"))
	for range singletonMigrationCleanupLimit + 1 {
		item.ID = uuid.NewString()
		item.Data.Identifier.RunID = ulid.Make()
		stored, err := shard.EnqueueItem(ctx, item, time.Now(), osqueue.EnqueueOpts{PassthroughJobId: true, SingletonMigrationKey: uuid.NewString()})
		require.NoError(t, err)
		require.NoError(t, shard.Dequeue(ctx, stored))
	}
	done, err := hold.FinishSingletonMigration(ctx, scope, "first")
	require.NoError(t, err)
	require.False(t, done)
	_, pending := singletonMigrationKeys(shard.Client().KeyGenerator(), scope)
	members, err := r.Members(pending)
	require.NoError(t, err)
	require.Len(t, members, 1)
	done, err = hold.FinishSingletonMigration(ctx, scope, "first")
	require.NoError(t, err)
	require.True(t, done)
	require.NoError(t, hold.BeginSingletonMigration(ctx, scope, "second"))
	done, err = hold.FinishSingletonMigration(ctx, scope, "first")
	require.ErrorIs(t, err, osqueue.ErrSingletonMigrationConflict)
	require.False(t, done)
}
