package constraintlifecycle

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type constraintHitRecord struct {
	Constraint   enums.QueueConstraint
	ItemMetadata map[string]any
}

type mockNotifier struct {
	mu      sync.Mutex
	records []constraintHitRecord
}

func (m *mockNotifier) OnConstraintHit(ctx context.Context, constraint enums.QueueConstraint, itemMetadata map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, constraintHitRecord{Constraint: constraint, ItemMetadata: itemMetadata})
	return nil
}

func (m *mockNotifier) getRecords() []constraintHitRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]constraintHitRecord, len(m.records))
	copy(cp, m.records)
	return cp
}

func TestConstraintNotifierCalledOnFunctionConcurrencyHit(t *testing.T) {
	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelDebug))
	ctx = logger.WithStdlib(ctx, l)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Redis setup
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	// Fake clock with background advancement
	timeTick := 100 * time.Millisecond
	clock := clockwork.NewFakeClock()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.Tick(timeTick):
				clock.Advance(timeTick)
				r.FastForward(timeTick)
				r.SetTime(clock.Now())
			}
		}
	}()

	// Capacity manager
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
	)
	require.NoError(t, err)

	// Mock notifier
	notifier := &mockNotifier{}

	accountID := uuid.New()
	workspaceID := uuid.New()
	fnID := uuid.New()

	options := []queue.QueueOpt{
		queue.WithClock(clock),
		queue.WithRunMode(queue.QueueRunMode{
			Sequential:                        true,
			Scavenger:                         true,
			Partition:                         true,
			Account:                           true,
			AccountWeight:                     85,
			ShadowPartition:                   true,
			AccountShadowPartition:            true,
			AccountShadowPartitionWeight:      85,
			NormalizePartition:                true,
			ShadowContinuationSkipProbability: consts.QueueContinuationSkipProbability,
			Continuations:                     true,
			ShadowContinuations:               true,
		}),
		queue.WithAllowKeyQueues(func(ctx context.Context, acctID, envID, fnID uuid.UUID) bool {
			return true
		}),
		queue.WithPollTick(150 * time.Millisecond),
		queue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
			return queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					SystemConcurrency:   consts.DefaultConcurrencyLimit,
					AccountConcurrency:  consts.DefaultConcurrencyLimit,
					FunctionConcurrency: 1, // limit to 1 to trigger constraint
				},
			}
		}),
		queue.WithCapacityManager(cm),
		queue.WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) bool { return true }),
		queue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		queue.WithConstraintNotifier(notifier),
	}

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, options...)

	q, err := queue.New(ctx, "test", shard, nil, nil, options...)
	require.NoError(t, err)

	// Enqueue 2 items for the same function, scheduled at now
	for i := 0; i < 2; i++ {
		jobID := uuid.New().String()
		runID := ulid.MustNew(ulid.Now(), nil)
		err := q.Enqueue(ctx, queue.Item{
			JobID:       &jobID,
			WorkspaceID: workspaceID,
			Identifier: state.Identifier{
				RunID:       runID,
				AccountID:   accountID,
				WorkspaceID: workspaceID,
				WorkflowID:  fnID,
			},
			Kind: queue.KindStart,
		}, clock.Now(), queue.EnqueueOpts{
			PassthroughJobId: true,
		})
		require.NoError(t, err)
	}

	// Synchronization: first item blocks until we release holdCh
	firstStarted := make(chan struct{}, 1)
	holdCh := make(chan struct{})
	var processed atomic.Int32

	runFunc := func(ctx context.Context, ri queue.RunInfo, i queue.Item) (queue.RunResult, error) {
		n := processed.Add(1)
		if n == 1 {
			firstStarted <- struct{}{}
			<-holdCh // hold first item's concurrency slot
		}
		return queue.RunResult{}, nil
	}

	// Start queue processor
	go func() {
		_ = q.Run(ctx, runFunc)
	}()

	// Wait for first item to start processing
	select {
	case <-firstStarted:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for first item to start")
	}

	// Wait for the scanner to attempt the second item and hit the constraint
	require.Eventually(t, func() bool {
		return len(notifier.getRecords()) > 0
	}, 30*time.Second, 200*time.Millisecond, "notifier was never called for constraint hit")

	// Assert the constraint hit is for function concurrency
	records := notifier.getRecords()
	require.GreaterOrEqual(t, len(records), 1)
	require.Equal(t, enums.QueueConstraintFunctionConcurrency, records[0].Constraint)

	// Unblock the first item so the second can process
	close(holdCh)

	// Wait for both items to complete
	require.Eventually(t, func() bool {
		return processed.Load() >= 2
	}, 30*time.Second, 200*time.Millisecond, "second item was never processed")

	cancel()
}
