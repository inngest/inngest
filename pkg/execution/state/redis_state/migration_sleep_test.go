package redis_state

import (
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
	for _, tc := range []string{"collision", "legacy same run", "legacy completion", "other account", "other environment", "other function", "other kind"} {
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
			_, err = shard.EnqueueItem(t.Context(), other, at, osqueue.EnqueueOpts{PassthroughJobId: true})
			require.NoError(t, err)
			if tc == "legacy completion" {
				require.NoError(t, shard.Dequeue(t.Context(), other))
			}
			copied, err := dst.EnqueueSleepForMigration(t.Context(), item, at, osqueue.EnqueueOpts{PassthroughJobId: true})
			if tc != "collision" {
				require.Error(t, err)
				expected := 1
				if tc == "legacy completion" {
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
			again, err := dst.EnqueueSleepForMigration(t.Context(), copied, at, osqueue.EnqueueOpts{PassthroughJobId: true})
			require.ErrorIs(t, err, osqueue.ErrQueueItemExists)
			require.Equal(t, copied.ID, again.ID, "another migration must preserve the run-scoped ID")
			count, err := rc.Do(t.Context(), rc.B().Hlen().Key(shard.Client().KeyGenerator().QueueItem()).Build()).AsInt64()
			require.NoError(t, err)
			require.Equal(t, int64(2), count)
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
			copied, err := dst.EnqueueSleepForMigration(t.Context(), item, at, osqueue.EnqueueOpts{PassthroughJobId: true})
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
			_, err := dst.EnqueueSleepForMigration(t.Context(), item, at, osqueue.EnqueueOpts{PassthroughJobId: true})
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
