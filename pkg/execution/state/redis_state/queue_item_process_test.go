package redis_state

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

type constraintApiDebugLifecycles struct {
	acquireCalls []constraintapi.OnCapacityLeaseAcquiredData
	extendCalls  []constraintapi.OnCapacityLeaseExtendedData
	releaseCalls []constraintapi.OnCapacityLeaseReleasedData
	l            sync.Mutex
}

// OnCapacityLeaseAcquired implements constraintapi.ConstraintAPILifecycleHooks.
func (c *constraintApiDebugLifecycles) OnCapacityLeaseAcquired(ctx context.Context, data constraintapi.OnCapacityLeaseAcquiredData) error {
	c.l.Lock()
	defer c.l.Unlock()
	c.acquireCalls = append(c.acquireCalls, data)
	return nil
}

// OnCapacityLeaseExtended implements constraintapi.ConstraintAPILifecycleHooks.
func (c *constraintApiDebugLifecycles) OnCapacityLeaseExtended(ctx context.Context, data constraintapi.OnCapacityLeaseExtendedData) error {
	c.l.Lock()
	defer c.l.Unlock()
	c.extendCalls = append(c.extendCalls, data)
	return nil
}

// OnCapacityLeaseReleased implements constraintapi.ConstraintAPILifecycleHooks.
func (c *constraintApiDebugLifecycles) OnCapacityLeaseReleased(ctx context.Context, data constraintapi.OnCapacityLeaseReleasedData) error {
	c.l.Lock()
	defer c.l.Unlock()
	c.releaseCalls = append(c.releaseCalls, data)
	return nil
}

func (c *constraintApiDebugLifecycles) reset() {
	c.l.Lock()
	defer c.l.Unlock()
	c.acquireCalls = nil
	c.extendCalls = nil
	c.releaseCalls = nil
}

func newConstraintAPIDebugLifecycles() *constraintApiDebugLifecycles {
	return &constraintApiDebugLifecycles{}
}

func TestQueueItemProcessWithConstraintChecks(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	shard := QueueShard{
		Name:        consts.DefaultQueueShardName,
		Kind:        string(enums.QueueShardKindRedis),
		RedisClient: NewQueueClient(rc, QueueDefaultKey),
	}
	kg := shard.RedisClient.kg

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	cmLifecycles := newConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
		constraintapi.WithNumScavengerShards(1),
		constraintapi.WithQueueShards(map[string]rueidis.Client{
			consts.DefaultQueueShardName: rc,
		}),
		constraintapi.WithQueueStateKeyPrefix("q:v1"),
		constraintapi.WithRateLimitClient(rc),
		constraintapi.WithRateLimitKeyPrefix("rl"),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.reset()
	}

	accountID := uuid.New()
	envID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
			},
		},
	}

	start := clock.Now()

	t.Run("without constraint api", func(t *testing.T) {
		reset()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := q.ItemPartition(ctx, shard, qi)

		var counter int64

		err = q.process(ctx, processItem{
			I:                        qi,
			P:                        p,
			disableConstraintUpdates: false,
			capacityLeaseID:          nil,
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, int(counter))
	})

	t.Run("with constraint api but no valid lease", func(t *testing.T) {
		reset()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := q.ItemPartition(ctx, shard, qi)

		var counter int64

		err = q.process(ctx, processItem{
			I:                        qi,
			P:                        p,
			disableConstraintUpdates: false,
			capacityLeaseID:          nil,
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			<-time.After(3 * time.Second)
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		// No extend calls should be fired
		require.Equal(t, 1, int(counter))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
	})

	t.Run("with constraint api and valid lease", func(t *testing.T) {
		reset()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := q.ItemPartition(ctx, shard, qi)

		// Acquire a lease
		resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			IdempotencyKey:       qi.ID,
			FunctionID:           fnID,
			LeaseIdempotencyKeys: []string{qi.ID},
			Amount:               1,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  5,
					FunctionConcurrency: 2,
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: kg.Concurrency("account", accountID.String()),
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						InProgressItemKey: kg.Concurrency("p", fnID.String()),
					},
				},
			},
			CurrentTime:     clock.Now(),
			Duration:        10 * time.Second,
			MaximumLifetime: time.Minute,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceExecutor,
				Location:          constraintapi.LeaseLocationItemLease,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
			Migration: constraintapi.MigrationIdentifier{
				QueueShard: shard.Name,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)

		require.Len(t, cmLifecycles.acquireCalls, 1)

		var counter int64

		err = q.process(ctx, processItem{
			I:                        qi,
			P:                        p,
			disableConstraintUpdates: true,
			capacityLeaseID:          &resp.Leases[0].LeaseID,
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
						// Ensure we tick the extend at least once
						clock.Advance(time.Second)
					}
				}
			}()

			<-time.After(3 * time.Second)
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, int(counter))

		service.Wait()

		// Expect at least 1 extend call
		require.Greater(t, len(cmLifecycles.extendCalls), 0)

		// Expect exactly 1 release call
		require.Equal(t, len(cmLifecycles.releaseCalls), 1)
	})
}

func TestQueueProcessorPreLeaseWithConstraintAPI(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	shard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

	q := NewQueue(
		shard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
	)
	ctx := context.Background()

	accountID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID: fnID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:  accountID,
				WorkflowID: fnID,
			},
		},
	}

	start := clock.Now()

	qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	p := q.ItemPartition(ctx, shard, qi)

	iter := processor{
		partition:            &p,
		items:                []*osqueue.QueueItem{&qi},
		partitionContinueCtr: 0,
		queue:                q,
		denies:               newLeaseDenyList(),
		staticTime:           q.clock.Now(),
		parallel:             false,
	}

	err = iter.process(ctx, &qi)
	require.NoError(t, err)
}
