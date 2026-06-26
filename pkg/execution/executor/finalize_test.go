package executor

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// racingShard simulates a queue shard where new items are injected after
// specific sweep passes, mimicking the race between a concurrent enqueue
// and the cleanup sweep.
type racingShard struct {
	queue.ShardOperations // embed to satisfy the large interface

	mu sync.Mutex
	// items currently in the queue, keyed by ID
	items map[string]*queue.QueueItem
	// sweepCount tracks how many times RunJobs has been called
	sweepCount int
	// injectAfterSweep maps sweep number -> items to inject after that sweep completes
	injectAfterSweep map[int][]*queue.QueueItem
	// dequeueCount tracks total successful dequeues
	dequeueCount atomic.Int32
}

func (s *racingShard) RunJobs(ctx context.Context, scope queue.Scope, runID ulid.ULID, limit, offset int64) ([]queue.JobResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Collect current items as response
	var jobs []queue.JobResponse
	for _, qi := range s.items {
		jobs = append(jobs, queue.JobResponse{
			JobID: qi.ID,
			Raw:   qi,
		})
	}

	s.sweepCount++
	sweep := s.sweepCount

	// Simulate race: inject items that arrived during this sweep
	if toInject, ok := s.injectAfterSweep[sweep]; ok {
		for _, qi := range toInject {
			s.items[qi.ID] = qi
		}
	}

	return jobs, nil
}

func (s *racingShard) Dequeue(ctx context.Context, i queue.QueueItem, opts ...queue.DequeueOptionFn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[i.ID]; !ok {
		return queue.ErrQueueItemNotFound
	}
	delete(s.items, i.ID)
	s.dequeueCount.Add(1)
	return nil
}

// racingShardRegistry wraps racingShard to satisfy ShardRegistry.
type racingShardRegistry struct {
	shard *racingQueueShard
}

func (r *racingShardRegistry) Primary() queue.QueueShard { return r.shard }
func (r *racingShardRegistry) ByName(name string) (queue.QueueShard, error) {
	return r.shard, nil
}
func (r *racingShardRegistry) ByGroup(string) []queue.QueueShard { return nil }
func (r *racingShardRegistry) Resolve(_ context.Context, _ queue.Scope, _ *string) (queue.QueueShard, error) {
	return r.shard, nil
}
func (r *racingShardRegistry) ForEach(ctx context.Context, fn func(context.Context, queue.QueueShard) error) error {
	return fn(ctx, r.shard)
}

// racingQueueShard wraps racingShard to implement QueueShard (Name/Kind/ShardAssignmentConfig).
type racingQueueShard struct {
	*racingShard
}

func (s *racingQueueShard) Name() string               { return "test" }
func (s *racingQueueShard) Kind() enums.QueueShardKind { return enums.QueueShardKindRedis }
func (s *racingQueueShard) ShardAssignmentConfig() queue.ShardAssignmentConfig {
	return queue.ShardAssignmentConfig{}
}

func TestFinalizeRemoveJobs_CatchesPostSweepEnqueue(t *testing.T) {
	// Setup: an initial item exists in the queue. After the first sweep removes
	// it, a new item is injected (simulating a concurrent enqueue during the
	// race window). The bounded-loop should catch it on the second pass.

	initialItem := &queue.QueueItem{ID: "item-1"}
	racedItem := &queue.QueueItem{ID: "item-raced"}

	shard := &racingShard{
		items: map[string]*queue.QueueItem{"item-1": initialItem},
		injectAfterSweep: map[int][]*queue.QueueItem{
			1: {racedItem}, // inject after first sweep
		},
	}
	queueShard := &racingQueueShard{racingShard: shard}

	e := &executor{
		log:    logger.VoidLogger(),
		shards: &racingShardRegistry{shard: queueShard},
	}

	runID := ulid.Make()
	opts := execution.FinalizeOpts{
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID:      runID,
				FunctionID: uuid.New(),
				Tenant: sv2.Tenant{
					AccountID: uuid.New(),
					AppID:     uuid.New(),
					EnvID:     uuid.New(),
				},
			},
		},
	}

	e.finalizeRemoveJobs(context.Background(), opts)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	require.Empty(t, shard.items, "all items should be removed, including those enqueued during the race window")
	require.Equal(t, int32(2), shard.dequeueCount.Load(), "should have dequeued 2 items total (initial + raced)")
}

func TestFinalizeMetricTagsIncludesAccountPlan(t *testing.T) {
	accountID := uuid.New()
	opts := execution.FinalizeOpts{
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				Tenant: sv2.Tenant{
					AccountID: accountID,
				},
			},
		},
		Optional: execution.FinalizeOptional{
			Reason: "test_reason",
		},
	}

	e := &executor{
		accountPlanMetricTagResolver: func(ctx context.Context, id uuid.UUID) string {
			require.Equal(t, accountID, id)
			return "self_serve"
		},
	}

	tags := e.finalizeMetricTags(context.Background(), enums.StepStatusCompleted, opts)
	require.Equal(t, map[string]any{
		"account_plan": "self_serve",
		"reason":       "test_reason",
		"status":       "Completed",
	}, tags)
}

func TestFinalizeMetricTagsDefaultsUnknownAccountPlan(t *testing.T) {
	opts := execution.FinalizeOpts{}

	t.Run("missing resolver", func(t *testing.T) {
		tags := (&executor{}).finalizeMetricTags(context.Background(), enums.StepStatusCompleted, opts)
		require.Equal(t, runStateAccountPlanUnknown, tags["account_plan"])
	})

	t.Run("unknown resolver value", func(t *testing.T) {
		e := &executor{
			accountPlanMetricTagResolver: func(context.Context, uuid.UUID) string {
				return "pro"
			},
		}
		tags := e.finalizeMetricTags(context.Background(), enums.StepStatusCompleted, opts)
		require.Equal(t, runStateAccountPlanUnknown, tags["account_plan"])
	})
}

func TestFinalizeRemoveJobs_CatchesMultipleRaceWindows(t *testing.T) {
	// Verifies the bounded loop handles items injected across multiple sweeps.
	// Sweep 1: removes item-1, item-2 injected during sweep
	// Sweep 2: removes item-2, item-3 injected during sweep
	// Sweep 3: removes item-3, no more injections

	shard := &racingShard{
		items: map[string]*queue.QueueItem{"item-1": {ID: "item-1"}},
		injectAfterSweep: map[int][]*queue.QueueItem{
			1: {{ID: "item-2"}},
			2: {{ID: "item-3"}},
		},
	}
	queueShard := &racingQueueShard{racingShard: shard}

	e := &executor{
		log:    logger.VoidLogger(),
		shards: &racingShardRegistry{shard: queueShard},
	}

	opts := execution.FinalizeOpts{
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID:      ulid.Make(),
				FunctionID: uuid.New(),
				Tenant: sv2.Tenant{
					AccountID: uuid.New(),
					AppID:     uuid.New(),
					EnvID:     uuid.New(),
				},
			},
		},
	}

	e.finalizeRemoveJobs(context.Background(), opts)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	require.Empty(t, shard.items, "all items should be removed across 3 sweeps")
	require.Equal(t, int32(3), shard.dequeueCount.Load())
	require.Equal(t, 3, shard.sweepCount, "should have performed exactly 3 sweeps")
}

func TestFinalizeRemoveJobs_BoundsAtMaxSweeps(t *testing.T) {
	// If items keep appearing beyond maxSweeps, the loop still terminates.
	// Sweep 1: removes item-1, item-2 injected
	// Sweep 2: removes item-2, item-3 injected
	// Sweep 3: removes item-3, item-4 injected (but no sweep 4)

	shard := &racingShard{
		items: map[string]*queue.QueueItem{"item-1": {ID: "item-1"}},
		injectAfterSweep: map[int][]*queue.QueueItem{
			1: {{ID: "item-2"}},
			2: {{ID: "item-3"}},
			3: {{ID: "item-4"}}, // injected during final sweep — won't be caught
		},
	}
	queueShard := &racingQueueShard{racingShard: shard}

	e := &executor{
		log:    logger.VoidLogger(),
		shards: &racingShardRegistry{shard: queueShard},
	}

	opts := execution.FinalizeOpts{
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID:      ulid.Make(),
				FunctionID: uuid.New(),
				Tenant: sv2.Tenant{
					AccountID: uuid.New(),
					AppID:     uuid.New(),
					EnvID:     uuid.New(),
				},
			},
		},
	}

	e.finalizeRemoveJobs(context.Background(), opts)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// The loop is bounded at 3, so item-4 which was injected during sweep 3
	// remains because there's no sweep 4.
	require.Len(t, shard.items, 1, "item injected during final sweep remains (bounded loop)")
	require.Contains(t, shard.items, "item-4")
	require.Equal(t, 3, shard.sweepCount, "must not exceed maxSweeps")
}

func TestFinalizeRemoveJobs_NoItemsNoSweep(t *testing.T) {
	// When no items exist, only a single sweep should occur and it should
	// terminate immediately without sleeping.

	shard := &racingShard{
		items:            map[string]*queue.QueueItem{},
		injectAfterSweep: map[int][]*queue.QueueItem{},
	}
	queueShard := &racingQueueShard{racingShard: shard}

	e := &executor{
		log:    logger.VoidLogger(),
		shards: &racingShardRegistry{shard: queueShard},
	}

	opts := execution.FinalizeOpts{
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID:      ulid.Make(),
				FunctionID: uuid.New(),
				Tenant: sv2.Tenant{
					AccountID: uuid.New(),
					AppID:     uuid.New(),
					EnvID:     uuid.New(),
				},
			},
		},
	}

	e.finalizeRemoveJobs(context.Background(), opts)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	require.Equal(t, 1, shard.sweepCount, "should only scan once when queue is empty")
	require.Equal(t, int32(0), shard.dequeueCount.Load())
}
