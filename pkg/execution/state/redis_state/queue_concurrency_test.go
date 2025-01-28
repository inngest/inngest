package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueuePartitionConcurrency(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	limit_1 := uuid.New()
	limit_10 := uuid.New()
	workflowIDs := []uuid.UUID{limit_1, limit_10}

	// Limit function concurrency by workflow ID.
	pkf := func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
		switch *p.FunctionID {
		case limit_1:
			return PartitionConcurrencyLimits{
				AccountLimit:   NoConcurrencyLimit,
				FunctionLimit:  1,
				CustomKeyLimit: 1,
			}
		case limit_10:
			return PartitionConcurrencyLimits{
				AccountLimit:   NoConcurrencyLimit,
				FunctionLimit:  10,
				CustomKeyLimit: 10,
			}
		default:
			// No concurrency, which means use the default concurrency limits.
			return PartitionConcurrencyLimits{NoConcurrencyLimit, NoConcurrencyLimit, NoConcurrencyLimit}
		}
	}

	// Create a new lifecycle listener.  This should be invoked each time we hit limits.
	ll := testLifecycleListener{
		lock:            &sync.Mutex{},
		fnConcurrency:   map[uuid.UUID]int{},
		acctConcurrency: map[uuid.UUID]int{},
		ckConcurrency:   map[string]int{},
	}

	q := NewQueue(
		QueueShard{RedisClient: NewQueueClient(rc, QueueDefaultKey), Kind: string(enums.QueueShardKindRedis), Name: consts.DefaultQueueShardName},
		WithNumWorkers(100),
		WithConcurrencyLimitGetter(pkf),
		WithQueueLifecycles(ll),
	)

	var (
		counter_1   int32
		counter_10  int32
		jobDuration = 2 * time.Second
	)

	// Run the queue.
	go func() {
		_ = q.Run(ctx, func(ctx context.Context, _ osqueue.RunInfo, item osqueue.Item) error {
			if item.Identifier.WorkflowID == limit_1 {
				fmt.Println("Single concurrency item hit", time.Now().Truncate(time.Millisecond))
			}

			<-time.After(jobDuration / 2)
			// each job takes 2 seconds to complete.
			switch item.Identifier.WorkflowID {
			case limit_1:
				atomic.AddInt32(&counter_1, 1)
			case limit_10:
				fmt.Println("10 concurrency item hit", time.Now().Truncate(time.Millisecond))
				atomic.AddInt32(&counter_10, 1)
			}

			<-time.After(jobDuration / 2)
			if item.Identifier.WorkflowID == limit_1 {
				fmt.Println("Single concurrency item done", time.Now().Truncate(time.Millisecond))
			}
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
			}, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}
	}

	<-time.After(jobDuration)

	require.EqualValues(t, 10, atomic.LoadInt32(&counter_10), "Should have hit all 10 items with a concurrency limit of 10")
	require.EqualValues(t, int32(1), atomic.LoadInt32(&counter_1), "Should have only run a single job")

	// TODO: Assert that the counterPartitionConcurrencyLimitReached counter isn't crazy high - we
	// don't want to be churning on the partition.
	for i := 0; i <= 100; i++ {
		<-time.After(500 * time.Millisecond)
		if atomic.LoadInt32(&counter_1) == 10 {
			break
		}
	}

	require.Eventually(t, func() bool {
		ll.lock.Lock()
		defer ll.lock.Unlock()
		return ll.fnConcurrency[limit_1] > 0
	}, 5*time.Second, 50*time.Millisecond)

	diff := time.Since(start).Seconds()
	require.Greater(t, int(diff), 10, "10 jobs should have taken at least 10 seconds")
	require.Less(t, int(diff), 40, "10 jobs should have taken fewer than 40 seconds") // an extra 2x latency due to race checker
}

type testLifecycleListener struct {
	lock            *sync.Mutex
	fnConcurrency   map[uuid.UUID]int
	acctConcurrency map[uuid.UUID]int
	ckConcurrency   map[string]int
}

func (t testLifecycleListener) OnFnConcurrencyLimitReached(_ context.Context, fnID uuid.UUID) {
	t.lock.Lock()
	defer t.lock.Unlock()

	i := t.fnConcurrency[fnID]
	t.fnConcurrency[fnID] = i + 1
}

func (t testLifecycleListener) OnAccountConcurrencyLimitReached(
	_ context.Context,
	acctID uuid.UUID,
	workspaceID *uuid.UUID,
) {
	t.lock.Lock()
	defer t.lock.Unlock()

	i := t.acctConcurrency[acctID]
	t.acctConcurrency[acctID] = i + 1
}

func (t testLifecycleListener) OnCustomKeyConcurrencyLimitReached(_ context.Context, key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	i := t.ckConcurrency[key]
	t.ckConcurrency[key] = i + 1
}
