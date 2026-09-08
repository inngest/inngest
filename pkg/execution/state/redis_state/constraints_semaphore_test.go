package redis_state

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// an app semaphore with no capacity key is the connect app with no workers.
func TestItemLeaseConstraintCheckReportsExhaustedAppSemaphore(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	r.SetTime(clock.Now())
	ctx := context.Background()

	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("default"),
		constraintapi.WithClock(clock),
	)
	require.NoError(t, err)

	accountID, envID, fnID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	constraints := osqueue.PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency: osqueue.PartitionConcurrency{
			AccountConcurrency:  10,
			FunctionConcurrency: 5,
		},
	}

	q, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
		osqueue.WithCapacityManager(cm),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return constraints
		}),
	)

	appSem := constraintapi.Semaphore{
		ID:      constraintapi.SemaphoreIDApp(appID),
		Weight:  1,
		Release: constraintapi.SemaphoreReleaseAuto,
	}

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Kind:    osqueue.KindEdge,
			Payload: json.RawMessage(`{"test":"payload"}`),
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
				AppID:       appID,
			},
			Semaphores: []constraintapi.Semaphore{appSem},
		},
	}

	qi, err := shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	sp := osqueue.ItemShadowPartition(ctx, qi)
	backlog := osqueue.ItemBacklog(ctx, qi)

	res, err := q.ItemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now())
	require.NoError(t, err)

	require.Nil(t, res.CapacityLease)
	require.Equal(t, enums.QueueConstraintSemaphore, res.LimitingConstraint)
	require.Len(t, res.ExhaustedSemaphores, 1)
	require.Equal(t, appSem.ID, res.ExhaustedSemaphores[0].ID)
	require.Equal(t, constraintapi.SemaphoreKindApp, res.ExhaustedSemaphores[0].Kind())
}

// the real LeaseItem path.  one exhausted app semaphore stops the pass after a
// single constraint API call, while an exhausted fnkey semaphore only skips the
// items that carry it and is checked once per pass.
func TestQueueProcessorIterateWithExhaustedSemaphores(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	ctx := context.Background()

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("default"),
		constraintapi.WithClock(clock),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.Reset()
	}

	accountID, envID, fnID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	constraints := osqueue.PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency: osqueue.PartitionConcurrency{
			AccountConcurrency:  10,
			FunctionConcurrency: 10,
		},
	}

	newItem := func(sems ...constraintapi.Semaphore) osqueue.QueueItem {
		return osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				Kind:    osqueue.KindEdge,
				Payload: json.RawMessage(`{"test":"payload"}`),
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
					AppID:       appID,
				},
				Semaphores: sems,
			},
		}
	}

	setup := func(t *testing.T, items ...osqueue.QueueItem) (osqueue.ProcessorIterator, *constraintapi.ConstraintApiDebugLifecycles) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return constraints
			}),
		)

		queued := make([]*osqueue.QueueItem, 0, len(items))
		for _, item := range items {
			qi, err := shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			queued = append(queued, &qi)
		}

		p := osqueue.ItemPartition(ctx, *queued[0])

		return osqueue.ProcessorIterator{
			Partition: &p,
			Items:     queued,
			Queue:     q,
			Dispatch: func(_ context.Context, item osqueue.ProcessItem) (osqueue.DispatchedItem, error) {
				q.Workers() <- item
				return osqueue.NewCompletedDispatchedItem(osqueue.DispatchedItemResult{}), nil
			},
			StaticTime: clock.Now(),
		}, cmLifecycles
	}

	t.Run("app semaphore stops the pass after one acquire", func(t *testing.T) {
		appSem := constraintapi.Semaphore{ID: constraintapi.SemaphoreIDApp(appID), Weight: 1}

		iter, lc := setup(t, newItem(appSem), newItem(appSem), newItem(appSem), newItem(appSem))

		require.NoError(t, iter.Iterate(ctx))

		require.Equal(t, 1, len(lc.AcquireCalls))
		require.Equal(t, int32(1), iter.CtrConcurrency.Load())
		require.Equal(t, int32(0), iter.CtrSuccess.Load())
		require.True(t, iter.IsSemaphoreLimitOnly.Load())
		require.True(t, iter.IsRequeuable())
	})

	t.Run("fnkey semaphore is checked once and skips only matching items", func(t *testing.T) {
		keyed := constraintapi.SemaphoreIDFnKey(fnID, "event.data.customer")
		semA := constraintapi.Semaphore{ID: keyed, EvaluatedKeyHash: "a", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}

		iter, lc := setup(t,
			newItem(semA), // exhausted, one acquire
			newItem(),     // dispatched, one acquire
			newItem(semA), // memo, no acquire
			newItem(semA), // memo, no acquire
		)

		require.NoError(t, iter.Iterate(ctx))

		require.Equal(t, 2, len(lc.AcquireCalls))
		require.Equal(t, int32(3), iter.CtrConcurrency.Load())
		require.Equal(t, int32(1), iter.CtrSuccess.Load())
		require.True(t, iter.IsSemaphoreLimitOnly.Load())
	})
}
