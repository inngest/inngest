package redis_state

import (
	"context"
	"crypto/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestQueueRunSequential(t *testing.T) {
	return
	r := miniredis.RunT(t)
	ctx := context.Background()

	q1ctx, q1cancel := context.WithCancel(ctx)

	q1 := NewQueue(redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}))
	q2 := NewQueue(redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}))

	// Run the queue.  After running this worker should claim the sequential lease.
	go func() {
		q1.Run(q1ctx, func(ctx context.Context, item osqueue.Item) error {
			return nil
		})
	}()
	go func() {
		<-time.After(10 * time.Millisecond)
		q2.Run(ctx, func(ctx context.Context, item osqueue.Item) error {
			return nil
		})
	}()

	<-time.After(20 * time.Millisecond)
	// Q1 gets lease, as it started first.
	require.NotNil(t, q1.seqLeaseID)
	// Lease is in the future.
	require.True(t, ulid.Time(q1.seqLeaseID.Time()).After(time.Now()))
	// Q2 has no lease.
	require.Nil(t, q2.seqLeaseID)

	<-time.After(SequentialLeaseDuration)

	// Q1 retains lease.
	require.NotNil(t, q1.seqLeaseID)
	require.Nil(t, q2.seqLeaseID)

	// Cancel q1, temrinating the queue with the sequential lease.
	q1cancel()

	<-time.After(SequentialLeaseDuration)

	// Q2 obtains lease.
	require.NotNil(t, q2.seqLeaseID)
	// And that the previous lease has expired.
	require.True(t, ulid.Time(q1.seqLeaseID.Time()).Before(time.Now()))
}

func TestQueueRunBasic(t *testing.T) {
	return

	r := miniredis.RunT(t)
	q := NewQueue(redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}))
	ctx := context.Background()

	idA, idB := uuid.New(), uuid.New()
	items := []QueueItem{
		{
			WorkflowID:  idA,
			MaxAttempts: 3,
			Data: osqueue.Item{
				Kind: osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID: idA,
					RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
				},
			},
		},
		{
			WorkflowID:  idB,
			MaxAttempts: 1,
			Data: osqueue.Item{
				Kind: osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID: idB,
					RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
				},
			},
		},
	}

	var handled int32
	go func() {
		q.Run(ctx, func(ctx context.Context, item osqueue.Item) error {
			logger.From(ctx).Debug().Interface("item", item).Msg("received item")
			atomic.AddInt32(&handled, 1)
			return nil
		})
	}()

	for _, item := range items {
		_, err := q.EnqueueItem(ctx, item, time.Now())
		require.NoError(t, err)
	}

	<-time.After(2 * time.Second)
	require.EqualValues(t, len(items), atomic.LoadInt32(&handled))

	// TODO: Assert queue items have been processed
	// TODO: Assert queue items have been dequeued, and peek is nil for workflows.
	// TODO: Assert metrics are correct.
}
