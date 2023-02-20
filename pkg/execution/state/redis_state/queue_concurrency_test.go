package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestQueuePartitionConcurrency(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 50})
	defer rc.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	limit_1 := uuid.New()
	limit_10 := uuid.New()
	workflowIDs := []uuid.UUID{limit_1, limit_10}

	pkf := func(ctx context.Context, p QueuePartition) (string, int) {
		switch p.WorkflowID {
		case limit_1:
			return p.WorkflowID.String(), 1
		case limit_10:
			return p.WorkflowID.String(), 10
		default:
			// No concurrency, which means use the default concurrency limits.
			return "", 0
		}
	}

	q := NewQueue(
		rc,
		WithNumWorkers(100),
		WithPartitionConcurrencyKeyGenerator(pkf),
	)

	var (
		counter_1  int32
		counter_10 int32
	)

	// Run the queue.
	go func() {
		_ = q.Run(ctx, func(ctx context.Context, item osqueue.Item) error {
			// each job takes 2 seconds to complete.
			switch item.Identifier.WorkflowID {
			case limit_1:
				fmt.Println("Single concurrency item hit", time.Now().Truncate(time.Millisecond))
				atomic.AddInt32(&counter_1, 1)
			case limit_10:
				atomic.AddInt32(&counter_10, 1)
			}
			<-time.After(time.Second)
			return nil
		})
	}()

	at := time.Now().Add(time.Second).Truncate(time.Second)

	// Schedule 10 jobs;  it should take 20 seconds for limit_1 to finish,
	// and 2 seconds for limit_10 to finish, given each job takes 2 seconds.
	start := time.Now()
	for i := 0; i < 10; i++ {
		for _, id := range workflowIDs {
			err := q.Enqueue(ctx, osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: id,
					RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
				},
			}, at)
			require.NoError(t, err)
		}
	}

	<-time.After(time.Second)

	require.EqualValues(t, 10, atomic.LoadInt32(&counter_10), "Should have hit all 10 items with a concurrency limit of 10")
	require.EqualValues(t, 1, atomic.LoadInt32(&counter_1), "Should have only run a single job")

	// TODO: Assert that the counterPartitionConcurrencyLimitReached counter isn't crazy high - we
	// don't want to be churning on the partition.
	for i := 0; i <= 100; i++ {
		<-time.After(500 * time.Millisecond)
		if atomic.LoadInt32(&counter_1) == 10 {
			break
		}
	}

	diff := time.Now().Sub(start).Seconds()
	require.Greater(t, int(diff), 10, "10 jobs should have taken at least 10 seconds")
	require.Less(t, int(diff), 25, "10 jobs should have taken fewer than 25 seconds") // an extra 1.5x latency due to shifting of partitions for churn
}
