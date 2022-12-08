package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	mrand "math/rand"
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
	r := miniredis.RunT(t)
	ctx := context.Background()

	q1ctx, q1cancel := context.WithCancel(ctx)

	q1 := NewQueue(
		redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}),
		// We can't add more than 8128 goroutines when detecting race conditions.
		WithNumWorkers(10),
	)
	q2 := NewQueue(
		redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}),
		// We can't add more than 8128 goroutines when detecting race conditions.
		WithNumWorkers(10),
	)

	// Run the queue.  After running this worker should claim the sequential lease.
	go func() {
		_ = q1.Run(q1ctx, func(ctx context.Context, item osqueue.Item) error {
			return nil
		})
	}()
	go func() {
		<-time.After(10 * time.Millisecond)
		_ = q2.Run(ctx, func(ctx context.Context, item osqueue.Item) error {
			return nil
		})
	}()

	<-time.After(20 * time.Millisecond)
	// Q1 gets lease, as it started first.
	require.NotNil(t, q1.sequentialLease())
	// Lease is in the future.
	require.True(t, ulid.Time(q1.sequentialLease().Time()).After(time.Now()))
	// Q2 has no lease.
	require.Nil(t, q2.sequentialLease())

	<-time.After(SequentialLeaseDuration)

	// Q1 retains lease.
	require.NotNil(t, q1.sequentialLease())
	require.Nil(t, q2.sequentialLease())

	// Cancel q1, temrinating the queue with the sequential lease.
	q1cancel()

	<-time.After(SequentialLeaseDuration)

	// Q2 obtains lease.
	require.NotNil(t, q2.sequentialLease())
	// And that the previous lease has expired.
	require.True(t, ulid.Time(q1.sequentialLease().Time()).Before(time.Now()))
}

func TestQueueRunBasic(t *testing.T) {
	r := miniredis.RunT(t)
	q := NewQueue(
		redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}),
		// We can't add more than 8128 goroutines when detecting race conditions.
		WithNumWorkers(1000),
	)
	ctx, cancel := context.WithCancel(context.Background())

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
		_ = q.Run(ctx, func(ctx context.Context, item osqueue.Item) error {
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
	require.EqualValues(t, int32(len(items)), atomic.LoadInt32(&handled))
	cancel()

	<-time.After(pollTick * 2)
	r.Close()

	// TODO: Assert queue items have been processed
	// TODO: Assert queue items have been dequeued, and peek is nil for workflows.
	// XXX: Assert metrics are correct.
}

// TestQueueRunExtended runs an extended in-memory test which:
// - Enqueues 1-150 jobs every 0-100ms, for one of 1,0000 random functions
// - Each job can be scheduled from now -> 10s in the future
// - Each job can take from 0->7500ms to complete.
//
// We assert that all jobs are handled within 100ms of budget.
//
// NOTE: When this runs with the race decetor (--race), the throughput of goroutines
// is severely limited.  This means that we need to extend the time in which we can
// process jobs.
func TestQueueRunExtended(t *testing.T) {
	r := miniredis.RunT(t)
	q := NewQueue(
		redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}),
		// We can't add more than 8128 goroutines when detecting race conditions,
		// so lower the number of workers.
		WithNumWorkers(1000),
	)
	ctx, cancel := context.WithCancel(context.Background())

	mrand.Seed(time.Now().UnixMicro())

	funcs := make([]uuid.UUID, 1000)
	for n := range funcs {
		funcs[n] = uuid.New()
	}

	jobCompleteMax := int32(12_500) // ms
	delayMax := int32(15_000)       // ms

	var handled int64
	go func() {
		_ = q.Run(ctx, func(ctx context.Context, item osqueue.Item) error {
			// Wait up to N seconds to complete.
			<-time.After(time.Duration(mrand.Int31n(atomic.LoadInt32(&jobCompleteMax))) * time.Millisecond)
			// Increase handled when job is done.
			atomic.AddInt64(&handled, 1)
			return nil
		})
	}()

	enqueueDuration := 30 * time.Second

	var added int64
	go func(duration time.Duration) {
		// For N seconds enqueue items.
		after := time.After(duration)
		for {
			sleep := mrand.Intn(50)
			select {
			case <-after:
				return
			case <-time.After(time.Duration(sleep) * time.Millisecond):
				// Enqueue 1-50 N jobs
				n := mrand.Intn(49) + 1
				for i := 0; i < n; i++ {
					item := QueueItem{
						WorkflowID: funcs[mrand.Intn(len(funcs))],
					}

					// Enqueue with a delay.
					diff := mrand.Int31n(atomic.LoadInt32(&delayMax))

					_, err := q.EnqueueItem(ctx, item, time.Now().Add(time.Duration(diff)*time.Millisecond))
					require.NoError(t, err)
					atomic.AddInt64(&added, 1)
				}
			}
		}
	}(enqueueDuration)

	go func() {
		t := time.Tick(1000 * time.Millisecond)
		prev := atomic.LoadInt64(&handled)
		for {
			<-t
			next := atomic.LoadInt64(&handled)
			added := atomic.LoadInt64(&added)
			fmt.Printf(
				"Handled: %d \t Handled delta: %d \t Added: %d \t Remaining: %d\n",
				next,
				next-prev,
				added,
				added-next,
			)
			prev = next
		}
	}()

	// Wait for all items to complete
	<-time.After(enqueueDuration)

	// The default wait
	wait := atomic.LoadInt32(&delayMax) + atomic.LoadInt32(&jobCompleteMax) + 100
	// Increasing, because of the race detector
	wait = wait * 2

	// We enqueue jobs up to delayMax, and they can take up to jobCompleteMax, so add
	// 100ms of buffer.
	<-time.After(time.Duration(wait) * time.Millisecond)

	a := atomic.LoadInt64(&added)
	h := atomic.LoadInt64(&handled)
	fmt.Printf("Added %d items\n", a)
	fmt.Printf("Handled %d items\n", h)

	require.EqualValues(t, a, h, "Added %d, handled %d (delta: %d)", a, h, a-h)
	cancel()

	<-time.After(pollTick * 2)
	r.Close()
}
