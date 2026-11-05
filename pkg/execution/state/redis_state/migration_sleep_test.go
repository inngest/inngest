package redis_state

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestSleepMigrationIdentity(t *testing.T) {
	for _, tc := range []string{"collision", "no legacy", "legacy same run", "legacy completion", "other account", "other environment", "other function", "other kind", "invalid ID"} {
		t.Run(tc, func(t *testing.T) {
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			_, shard := newQueue(t, rc)
			dst := shard.(*queue)
			fn, env := uuid.New(), uuid.New()
			id := osqueue.HashID(t.Context(), "same-sleep")
			item := osqueue.QueueItem{ID: id, FunctionID: fn, WorkspaceID: env, Data: osqueue.Item{JobID: &id, Kind: osqueue.KindSleep, WorkspaceID: env, Identifier: state.Identifier{AccountID: uuid.New(), WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}}
			other := item
			other.Data.Identifier.RunID = ulid.Make()
			switch tc {
			case "legacy same run":
				other = item
			case "other account":
				other.Data.Identifier.AccountID = uuid.New()
			case "other environment":
				other.Data.Identifier.WorkspaceID = uuid.New()
			case "other function":
				other.FunctionID = uuid.New()
				other.Data.Identifier.WorkflowID = other.FunctionID
			case "other kind":
				other.Data.Kind = osqueue.KindEdge
			}
			at := time.Now().Add(time.Hour)
			if tc != "no legacy" && tc != "invalid ID" {
				_, err = shard.EnqueueItem(t.Context(), other, at, osqueue.EnqueueOpts{PassthroughJobId: true})
				require.NoError(t, err)
			}
			if tc == "legacy completion" {
				require.NoError(t, shard.Dequeue(t.Context(), other))
			}
			copied := item
			copied.ID = osqueue.MigrationSleepID(item)
			copied.Data.JobID = &copied.ID
			if tc == "invalid ID" {
				copied.ID = "wrong"
			}
			copied, err = dst.EnqueueItem(t.Context(), copied, at, osqueue.EnqueueOpts{PassthroughJobId: true, LegacySleepID: item.ID})
			if tc != "collision" && tc != "no legacy" {
				require.Error(t, err)
				if tc != "legacy same run" {
					require.NotErrorIs(t, err, osqueue.ErrQueueItemExists)
				}
				expected := 1
				if tc == "legacy completion" || tc == "invalid ID" {
					expected = 0
				}
				count, err := rc.Do(t.Context(), rc.B().Hlen().Key(shard.Client().KeyGenerator().QueueItem()).Build()).AsInt64()
				require.NoError(t, err)
				require.Equal(t, int64(expected), count)
				return
			}
			require.NoError(t, err)
			require.Equal(t, osqueue.MigrationSleepID(item), copied.ID)
			require.Equal(t, at.UnixMilli(), copied.AtMS)
			again, err := enqueueRunScopedSleep(t.Context(), dst, copied, at, osqueue.EnqueueOpts{PassthroughJobId: true})
			require.ErrorIs(t, err, osqueue.ErrQueueItemExists)
			require.Equal(t, copied.ID, again.ID, "another migration must preserve the run-scoped ID")
			count, err := rc.Do(t.Context(), rc.B().Hlen().Key(shard.Client().KeyGenerator().QueueItem()).Build()).AsInt64()
			require.NoError(t, err)
			expected := int64(2)
			if tc == "no legacy" {
				expected = 1
			}
			require.Equal(t, expected, count)
		})
	}
}

func TestMigratedSleepRuntimeRetry(t *testing.T) {
	for _, completed := range []bool{false, true} {
		t.Run(fmt.Sprint("completed=", completed), func(t *testing.T) {
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			_, shard := newQueue(t, rc)
			dst := shard.(*queue)
			registry, err := osqueue.NewSingleShardRegistry(shard)
			require.NoError(t, err)
			producer := osqueue.NewProducer(registry)
			fn, env, raw := uuid.New(), uuid.New(), "legacy-sleep"
			id := osqueue.HashID(t.Context(), raw)
			ttl := time.Minute
			item := osqueue.QueueItem{ID: id, FunctionID: fn, WorkspaceID: env, IdempotencyPeriod: &ttl, Data: osqueue.Item{JobID: &raw, Kind: osqueue.KindSleep, WorkspaceID: env, Identifier: state.Identifier{AccountID: uuid.New(), WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}}
			at := time.Now().Add(time.Hour)
			copied, err := enqueueRunScopedSleep(t.Context(), dst, item, at, osqueue.EnqueueOpts{PassthroughJobId: true})
			require.NoError(t, err)
			if completed {
				require.NoError(t, shard.Dequeue(t.Context(), copied))
			}
			err = producer.Enqueue(t.Context(), item.Data, at, osqueue.EnqueueOpts{})
			require.ErrorIs(t, err, osqueue.ErrQueueItemExists)
			duplicate, ok := errors.AsType[osqueue.QueueItemExistsError](err)
			require.True(t, ok)
			require.Equal(t, copied.ID, duplicate.JobID)
			_, err = shard.LoadQueueItem(t.Context(), id)
			require.ErrorIs(t, err, osqueue.ErrQueueItemNotFound)
			if completed {
				r.FastForward(ttl + time.Second)
				require.NoError(t, producer.Enqueue(t.Context(), item.Data, at, osqueue.EnqueueOpts{}))
			}
		})
	}
}

func TestSleepMigrationRacesRuntimeRetry(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	_, shard := newQueue(t, rc)
	dst := shard.(*queue)
	registry, err := osqueue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	producer := osqueue.NewProducer(registry)
	fn, env := uuid.New(), uuid.New()
	for range 20 {
		raw := uuid.NewString()
		id := osqueue.HashID(t.Context(), raw)
		item := osqueue.QueueItem{ID: id, FunctionID: fn, WorkspaceID: env, Data: osqueue.Item{JobID: &raw, Kind: osqueue.KindSleep, WorkspaceID: env, Identifier: state.Identifier{AccountID: uuid.New(), WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}}
		at := time.Now().Add(time.Hour)
		start := make(chan struct{})
		results := make(chan error, 2)
		go func() { <-start; results <- producer.Enqueue(t.Context(), item.Data, at, osqueue.EnqueueOpts{}) }()
		go func() {
			<-start
			_, err := enqueueRunScopedSleep(t.Context(), dst, item, at, osqueue.EnqueueOpts{PassthroughJobId: true})
			results <- err
		}()
		close(start)
		first, second := <-results, <-results
		if first == nil {
			require.ErrorIs(t, second, osqueue.ErrQueueItemExists)
		} else {
			require.ErrorIs(t, first, osqueue.ErrQueueItemExists)
			require.NoError(t, second)
		}
	}
	count, err := rc.Do(t.Context(), rc.B().Hlen().Key(shard.Client().KeyGenerator().QueueItem()).Build()).AsInt64()
	require.NoError(t, err)
	require.Equal(t, int64(20), count, "one physical sleep per run after racing enqueues")
}

// Exercise migration's preparation through the ordinary enqueue entry point.
func enqueueRunScopedSleep(ctx context.Context, shard osqueue.QueueShard, item osqueue.QueueItem, at time.Time, opts osqueue.EnqueueOpts) (osqueue.QueueItem, error) {
	legacyID := item.ID
	item.ID = osqueue.MigrationSleepID(item)
	item.Data.JobID = &item.ID
	if item.ID != legacyID {
		opts.LegacySleepID = legacyID
	}
	return shard.EnqueueItem(ctx, item, at, opts)
}
