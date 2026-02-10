package queue

import (
	"context"
	"iter"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// mockQueueProcessor implements QueueProcessor for testing
type mockQueueProcessor struct {
	shard     QueueShard
	clock     clockwork.Clock
	sem       util.TrackingSemaphore
	opts      *QueueOptions
	workers   chan ProcessItem
	seqLease  *ulid.ULID
	shadowCh  chan ShadowPartitionChanMsg
	shadowMu  sync.Mutex
	shadowMap map[string]ShadowContinuation

	// constraintResultFunc controls what ItemLeaseConstraintCheck returns
	constraintResultFunc func() enums.QueueConstraint
	leaseCount           int32
}

func (m *mockQueueProcessor) Shard() QueueShard                                   { return m.shard }
func (m *mockQueueProcessor) Clock() clockwork.Clock                              { return m.clock }
func (m *mockQueueProcessor) Semaphore() util.TrackingSemaphore                   { return m.sem }
func (m *mockQueueProcessor) Options() *QueueOptions                              { return m.opts }
func (m *mockQueueProcessor) Workers() chan ProcessItem                           { return m.workers }
func (m *mockQueueProcessor) SequentialLease() *ulid.ULID                         { return m.seqLease }
func (m *mockQueueProcessor) ShadowPartitionWorkers() chan ShadowPartitionChanMsg { return m.shadowCh }
func (m *mockQueueProcessor) AddShadowContinue(ctx context.Context, p *QueueShadowPartition, ctr uint) {
}

func (m *mockQueueProcessor) GetShadowContinuations() map[string]ShadowContinuation {
	m.shadowMu.Lock()
	defer m.shadowMu.Unlock()
	return m.shadowMap
}

func (m *mockQueueProcessor) ClearShadowContinuations() {
	m.shadowMu.Lock()
	defer m.shadowMu.Unlock()
	m.shadowMap = make(map[string]ShadowContinuation)
}

// mockShardForIterator implements the minimal QueueShard interface methods used by ProcessorIterator
type mockShardForIterator struct {
	name string
}

func (m *mockShardForIterator) Name() string {
	return m.name
}

func (m *mockShardForIterator) Kind() enums.QueueShardKind {
	return enums.QueueShardKindRedis
}

func (m *mockShardForIterator) Lease(ctx context.Context, item QueueItem, duration time.Duration, now time.Time, denies *LeaseDenies, options ...LeaseOptionFn) (*ulid.ULID, error) {
	id := ulid.Make()
	return &id, nil
}

func (m *mockShardForIterator) Requeue(ctx context.Context, i QueueItem, at time.Time, opts ...RequeueOptionFn) error {
	return nil
}

// Implement all other required ShardOperations methods as stubs
func (m *mockShardForIterator) EnqueueItem(ctx context.Context, i QueueItem, at time.Time, opts EnqueueOpts) (QueueItem, error) {
	return i, nil
}

func (m *mockShardForIterator) Peek(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*QueueItem, error) {
	return nil, nil
}

func (m *mockShardForIterator) PeekRandom(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*QueueItem, error) {
	return nil, nil
}

func (m *mockShardForIterator) ExtendLease(ctx context.Context, i QueueItem, leaseID ulid.ULID, duration time.Duration, opts ...ExtendLeaseOptionFn) (*ulid.ULID, error) {
	return nil, nil
}

func (m *mockShardForIterator) RequeueByJobID(ctx context.Context, jobID string, at time.Time) error {
	return nil
}

func (m *mockShardForIterator) Dequeue(ctx context.Context, i QueueItem, opts ...DequeueOptionFn) error {
	return nil
}

func (m *mockShardForIterator) PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error) {
	return nil, nil
}

func (m *mockShardForIterator) PartitionLease(ctx context.Context, p *QueuePartition, duration time.Duration, opts ...PartitionLeaseOpt) (*ulid.ULID, int, error) {
	return nil, 0, nil
}

func (m *mockShardForIterator) PartitionRequeue(ctx context.Context, p *QueuePartition, at time.Time, forceAt bool) error {
	return nil
}

func (m *mockShardForIterator) Scavenge(ctx context.Context, limit int) (int, error) {
	return 0, nil
}

func (m *mockShardForIterator) ActiveCheck(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *mockShardForIterator) Instrument(ctx context.Context) error {
	return nil
}

func (m *mockShardForIterator) ItemsByPartition(ctx context.Context, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	return nil, nil
}

func (m *mockShardForIterator) ItemsByBacklog(ctx context.Context, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	return nil, nil
}

func (m *mockShardForIterator) SetFunctionMigrate(ctx context.Context, fnID uuid.UUID, migrateLockUntil *time.Time) error {
	return nil
}

func (m *mockShardForIterator) ResetAttemptsByJobID(ctx context.Context, jobID string) error {
	return nil
}

func (m *mockShardForIterator) PeekEWMA(ctx context.Context, fnID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockShardForIterator) SetPeekEWMA(ctx context.Context, fnID *uuid.UUID, val int64) error {
	return nil
}

func (m *mockShardForIterator) PartitionSize(ctx context.Context, partitionID string, until time.Time) (int64, error) {
	return 0, nil
}

func (m *mockShardForIterator) ConfigLease(ctx context.Context, key string, duration time.Duration, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error) {
	return nil, nil
}

func (m *mockShardForIterator) AccountPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockShardForIterator) PeekAccountPartitions(ctx context.Context, accountID uuid.UUID, peekLimit int64, peekUntil time.Time, sequential bool) ([]*QueuePartition, error) {
	return nil, nil
}

func (m *mockShardForIterator) PeekGlobalPartitions(ctx context.Context, peekLimit int64, peekUntil time.Time, sequential bool) ([]*QueuePartition, error) {
	return nil, nil
}

func (m *mockShardForIterator) BacklogRefillConstraintCheck(ctx context.Context, shadowPart *QueueShadowPartition, backlog *QueueBacklog, constraints PartitionConstraintConfig, items []*QueueItem, operationIdempotencyKey string, now time.Time) (*BacklogRefillConstraintCheckResult, error) {
	return nil, nil
}

func (m *mockShardForIterator) RemoveQueueItem(ctx context.Context, partitionID string, itemID string) error {
	return nil
}

func (m *mockShardForIterator) LoadQueueItem(ctx context.Context, itemID string) (*QueueItem, error) {
	return nil, nil
}

func (m *mockShardForIterator) LeaseBacklogForNormalization(ctx context.Context, bl *QueueBacklog) error {
	return nil
}

func (m *mockShardForIterator) ExtendBacklogNormalizationLease(ctx context.Context, now time.Time, bl *QueueBacklog) error {
	return nil
}

func (m *mockShardForIterator) ShadowPartitionPeekNormalizeBacklogs(ctx context.Context, sp *QueueShadowPartition, limit int64) ([]*QueueBacklog, error) {
	return nil, nil
}

func (m *mockShardForIterator) BacklogNormalizePeek(ctx context.Context, b *QueueBacklog, limit int64) (*PeekResult[QueueItem], error) {
	return nil, nil
}

func (m *mockShardForIterator) PeekGlobalNormalizeAccounts(ctx context.Context, until time.Time, limit int64) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockShardForIterator) PeekGlobalShadowPartitionAccounts(ctx context.Context, sequential bool, until time.Time, limit int64) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockShardForIterator) ShadowPartitionRequeue(ctx context.Context, sp *QueueShadowPartition, requeueAt *time.Time) error {
	return nil
}

func (m *mockShardForIterator) ShadowPartitionLease(ctx context.Context, sp *QueueShadowPartition, duration time.Duration) (*ulid.ULID, error) {
	return nil, nil
}

func (m *mockShardForIterator) ShadowPartitionExtendLease(ctx context.Context, sp *QueueShadowPartition, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	return nil, nil
}

func (m *mockShardForIterator) ShadowPartitionPeek(ctx context.Context, sp *QueueShadowPartition, sequential bool, until time.Time, limit int64, opts ...PeekOpt) ([]*QueueBacklog, int, error) {
	return nil, 0, nil
}

func (m *mockShardForIterator) BacklogPrepareNormalize(ctx context.Context, b *QueueBacklog, sp *QueueShadowPartition) error {
	return nil
}

func (m *mockShardForIterator) BacklogPeek(ctx context.Context, b *QueueBacklog, from time.Time, until time.Time, limit int64, opts ...PeekOpt) ([]*QueueItem, int, error) {
	return nil, 0, nil
}

func (m *mockShardForIterator) BacklogRefill(ctx context.Context, b *QueueBacklog, sp *QueueShadowPartition, refillUntil time.Time, refillItems []string, latestConstraints PartitionConstraintConfig, options ...BacklogRefillOptionFn) (*BacklogRefillResult, error) {
	return nil, nil
}

func (m *mockShardForIterator) BacklogRequeue(ctx context.Context, backlog *QueueBacklog, sp *QueueShadowPartition, requeueAt time.Time) error {
	return nil
}

func (m *mockShardForIterator) BacklogsByPartition(ctx context.Context, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueBacklog], error) {
	return nil, nil
}

func (m *mockShardForIterator) BacklogSize(ctx context.Context, backlogID string) (int64, error) {
	return 0, nil
}

func (m *mockShardForIterator) PeekShadowPartitions(ctx context.Context, accountID *uuid.UUID, sequential bool, peekLimit int64, until time.Time) ([]*QueueShadowPartition, error) {
	return nil, nil
}

func (m *mockShardForIterator) IsMigrationLocked(ctx context.Context, fnID uuid.UUID) (*time.Time, error) {
	return nil, nil
}

func (m *mockShardForIterator) TotalSystemQueueDepth(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockShardForIterator) DequeueByJobID(ctx context.Context, jobID string) error {
	return nil
}

func (m *mockShardForIterator) ItemByID(ctx context.Context, jobID string) (*QueueItem, error) {
	return nil, nil
}

func (m *mockShardForIterator) ItemExists(ctx context.Context, jobID string) (bool, error) {
	return false, nil
}

func (m *mockShardForIterator) ItemsByRunID(ctx context.Context, runID ulid.ULID) ([]*QueueItem, error) {
	return nil, nil
}

func (m *mockShardForIterator) PartitionBacklogSize(ctx context.Context, partitionID string) (int64, error) {
	return 0, nil
}

func (m *mockShardForIterator) PartitionByID(ctx context.Context, partitionID string) (*PartitionInspectionResult, error) {
	return nil, nil
}

func (m *mockShardForIterator) UnpauseFunction(ctx context.Context, acctID, fnID uuid.UUID) error {
	return nil
}

func (m *mockShardForIterator) OutstandingJobCount(ctx context.Context, workspaceID, workflowID uuid.UUID, runID ulid.ULID) (int, error) {
	return 0, nil
}

func (m *mockShardForIterator) RunningCount(ctx context.Context, functionID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockShardForIterator) StatusCount(ctx context.Context, workflowID uuid.UUID, status string) (int64, error) {
	return 0, nil
}

func (m *mockShardForIterator) RunJobs(ctx context.Context, workspaceID, workflowID uuid.UUID, runID ulid.ULID, limit, offset int64) ([]JobResponse, error) {
	return nil, nil
}

func (m *mockQueueProcessor) BacklogRefillConstraintCheck(ctx context.Context, shadowPart *QueueShadowPartition, backlog *QueueBacklog, constraints PartitionConstraintConfig, items []*QueueItem, operationIdempotencyKey string, now time.Time) (*BacklogRefillConstraintCheckResult, error) {
	return nil, nil
}

func (m *mockQueueProcessor) ItemLeaseConstraintCheck(ctx context.Context, shadowPart *QueueShadowPartition, backlog *QueueBacklog, constraints PartitionConstraintConfig, item *QueueItem, now time.Time) (ItemLeaseConstraintCheckResult, error) {
	atomic.AddInt32(&m.leaseCount, 1)

	// Add some delay to increase chance of race
	time.Sleep(time.Microsecond * 50)

	var constraint enums.QueueConstraint
	if m.constraintResultFunc != nil {
		constraint = m.constraintResultFunc()
	} else {
		constraint = enums.QueueConstraintNotLimited
	}

	return ItemLeaseConstraintCheckResult{
		LimitingConstraint:   constraint,
		SkipConstraintChecks: true,
	}, nil
}

// TestProcessorIteratorCounterRaceCondition tests for race conditions when
// ProcessorIterator processes items in parallel mode.
//
// This test verifies that the counter increments (CtrSuccess, CtrConcurrency, CtrRateLimit)
// in ProcessorIterator.Process() are not thread-safe when Parallel=true.
//
// The race condition occurs because:
// - Multiple goroutines call Process() concurrently via errgroup in Iterate()
// - Each goroutine increments counters like `p.CtrSuccess++` without synchronization
// - This causes lost updates when multiple goroutines read-modify-write simultaneously
//
// Expected behavior with race: Counter values may be less than expected due to lost updates.
// The Go race detector will also report data races on these counter fields.
//
// Run with: go test -race -run TestProcessorIteratorCounterRaceCondition
func TestProcessorIteratorCounterRaceCondition(t *testing.T) {
	ctx := context.Background()

	numItems := 100
	numWorkers := int32(numItems)

	accountID := uuid.New()
	fnID := uuid.New()
	envID := uuid.New()

	// Create mock shard
	shard := &mockShardForIterator{
		name: "test-shard",
	}

	// Create a buffered workers channel to receive processed items
	workers := make(chan ProcessItem, numItems)

	// Create QueueOptions with defaults
	opts := NewQueueOptions()

	// Create mock processor with constraintResultFunc set on the processor,
	// which now owns ItemLeaseConstraintCheck
	mockProc := &mockQueueProcessor{
		shard:     shard,
		clock:     clockwork.NewRealClock(),
		sem:       util.NewTrackingSemaphore(int(numWorkers)),
		workers:   workers,
		shadowMap: make(map[string]ShadowContinuation),
		opts:      opts,
		constraintResultFunc: func() enums.QueueConstraint {
			return enums.QueueConstraintNotLimited
		},
	}

	// Create test partition
	partition := &QueuePartition{
		ID:         fnID.String(),
		AccountID:  accountID,
		EnvID:      &envID,
		FunctionID: &fnID,
	}

	// Create test items
	items := make([]*QueueItem, numItems)
	for i := 0; i < numItems; i++ {
		runID := ulid.Make()
		items[i] = &QueueItem{
			ID:          ulid.Make().String(),
			FunctionID:  fnID,
			WorkspaceID: envID,
			AtMS:        time.Now().UnixMilli(),
			Data: Item{
				Kind: KindEdge,
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
					RunID:      runID,
				},
			},
		}
	}

	// Create ProcessorIterator with Parallel = true
	iter := ProcessorIterator{
		Partition:            partition,
		Items:                items,
		PartitionContinueCtr: 0,
		Queue:                mockProc,
		Denies:               NewLeaseDenyList(),
		StaticTime:           time.Now(),
		Parallel:             true, // Enable parallel processing
	}

	// Run iteration - this is where the race condition would occur
	err := iter.Iterate(ctx)
	require.NoError(t, err)

	// Drain workers channel
	close(workers)
	receivedCount := 0
	for range workers {
		receivedCount++
	}

	// Verify counters match expected values
	ctrSuccess := iter.CtrSuccess.Load()
	ctrConcurrency := iter.CtrConcurrency.Load()
	ctrRateLimit := iter.CtrRateLimit.Load()

	t.Logf("CtrSuccess: %d, CtrConcurrency: %d, CtrRateLimit: %d",
		ctrSuccess, ctrConcurrency, ctrRateLimit)
	t.Logf("Items processed: %d, Items received by workers: %d", numItems, receivedCount)

	// With atomic operations, counter values should now be correct
	require.Equal(t, int32(numItems), ctrSuccess,
		"CtrSuccess should equal number of items when all leases succeed")
	require.Equal(t, int32(0), ctrConcurrency,
		"CtrConcurrency should be 0 when no concurrency limits hit")
	require.Equal(t, int32(0), ctrRateLimit,
		"CtrRateLimit should be 0 when no rate limits hit")
}

// TestProcessorIteratorCounterRaceConditionMixed tests race conditions with
// mixed constraint results (success, throttle, concurrency limits).
func TestProcessorIteratorCounterRaceConditionMixed(t *testing.T) {
	ctx := context.Background()

	numItems := 100
	numWorkers := int32(numItems)

	accountID := uuid.New()
	fnID := uuid.New()
	envID := uuid.New()

	// Track which constraint each item will hit
	var callCount int32

	shard := &mockShardForIterator{
		name: "test-shard",
	}

	workers := make(chan ProcessItem, numItems)
	opts := NewQueueOptions()

	// Set constraintResultFunc on the processor mock, which now owns ItemLeaseConstraintCheck
	mockProc := &mockQueueProcessor{
		shard:     shard,
		clock:     clockwork.NewRealClock(),
		sem:       util.NewTrackingSemaphore(int(numWorkers)),
		workers:   workers,
		shadowMap: make(map[string]ShadowContinuation),
		opts:      opts,
		constraintResultFunc: func() enums.QueueConstraint {
			count := atomic.AddInt32(&callCount, 1)
			// Add some delay to increase chance of race
			time.Sleep(time.Microsecond * 10)

			switch count % 3 {
			case 1:
				return enums.QueueConstraintNotLimited
			case 2:
				return enums.QueueConstraintThrottle
			default:
				return enums.QueueConstraintCustomConcurrencyKey1
			}
		},
	}

	partition := &QueuePartition{
		ID:         fnID.String(),
		AccountID:  accountID,
		EnvID:      &envID,
		FunctionID: &fnID,
	}

	items := make([]*QueueItem, numItems)
	for i := 0; i < numItems; i++ {
		runID := ulid.Make()
		items[i] = &QueueItem{
			ID:          ulid.Make().String(),
			FunctionID:  fnID,
			WorkspaceID: envID,
			AtMS:        time.Now().UnixMilli(),
			Data: Item{
				Kind: KindEdge,
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
					RunID:      runID,
				},
			},
		}
	}

	iter := ProcessorIterator{
		Partition:            partition,
		Items:                items,
		PartitionContinueCtr: 0,
		Queue:                mockProc,
		Denies:               NewLeaseDenyList(),
		StaticTime:           time.Now(),
		Parallel:             true,
	}

	// Run iteration
	err := iter.Iterate(ctx)
	// We expect nil error because throttle/concurrency errors are handled internally
	require.NoError(t, err)

	close(workers)
	receivedCount := 0
	for range workers {
		receivedCount++
	}

	ctrSuccess := iter.CtrSuccess.Load()
	ctrConcurrency := iter.CtrConcurrency.Load()
	ctrRateLimit := iter.CtrRateLimit.Load()

	t.Logf("Actual - CtrSuccess: %d, CtrRateLimit: %d, CtrConcurrency: %d",
		ctrSuccess, ctrRateLimit, ctrConcurrency)

	// With atomic operations, the total should now add up correctly
	totalCounted := ctrSuccess + ctrRateLimit + ctrConcurrency
	t.Logf("Total counted: %d, Expected: %d", totalCounted, numItems)

	// With atomic operations, this should now pass
	require.Equal(t, int32(numItems), totalCounted,
		"Total counted items should equal number of items processed")
}

// TestProcessorIteratorIsCustomKeyLimitOnlyRace tests race condition on IsCustomKeyLimitOnly flag
func TestProcessorIteratorIsCustomKeyLimitOnlyRace(t *testing.T) {
	ctx := context.Background()

	numItems := 100
	numWorkers := int32(numItems)

	accountID := uuid.New()
	fnID := uuid.New()
	envID := uuid.New()

	var callCount int32

	shard := &mockShardForIterator{
		name: "test-shard",
	}

	workers := make(chan ProcessItem, numItems)
	opts := NewQueueOptions()

	// Set constraintResultFunc on the processor mock, which now owns ItemLeaseConstraintCheck
	mockProc := &mockQueueProcessor{
		shard:     shard,
		clock:     clockwork.NewRealClock(),
		sem:       util.NewTrackingSemaphore(int(numWorkers)),
		workers:   workers,
		shadowMap: make(map[string]ShadowContinuation),
		opts:      opts,
		constraintResultFunc: func() enums.QueueConstraint {
			count := atomic.AddInt32(&callCount, 1)
			time.Sleep(time.Microsecond * 10)

			// Alternate between custom key limit and function concurrency limit
			if count%2 == 0 {
				return enums.QueueConstraintCustomConcurrencyKey1
			}
			return enums.QueueConstraintFunctionConcurrency
		},
	}

	partition := &QueuePartition{
		ID:         fnID.String(),
		AccountID:  accountID,
		EnvID:      &envID,
		FunctionID: &fnID,
	}

	items := make([]*QueueItem, numItems)
	for i := 0; i < numItems; i++ {
		runID := ulid.Make()
		items[i] = &QueueItem{
			ID:          ulid.Make().String(),
			FunctionID:  fnID,
			WorkspaceID: envID,
			AtMS:        time.Now().UnixMilli(),
			Data: Item{
				Kind: KindEdge,
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
					RunID:      runID,
				},
			},
		}
	}

	iter := ProcessorIterator{
		Partition:            partition,
		Items:                items,
		PartitionContinueCtr: 0,
		Queue:                mockProc,
		Denies:               NewLeaseDenyList(),
		StaticTime:           time.Now(),
		Parallel:             true,
	}

	err := iter.Iterate(ctx)
	require.NoError(t, err)

	close(workers)

	isCustomKeyLimitOnly := iter.IsCustomKeyLimitOnly.Load()
	ctrConcurrency := iter.CtrConcurrency.Load()

	// With function concurrency mixed in, IsCustomKeyLimitOnly should be false
	t.Logf("IsCustomKeyLimitOnly: %v", isCustomKeyLimitOnly)
	t.Logf("CtrConcurrency: %d", ctrConcurrency)

	// We expect IsCustomKeyLimitOnly to be false since we're hitting function concurrency limits
	// With atomic operations, this should now be deterministic
	require.False(t, isCustomKeyLimitOnly,
		"IsCustomKeyLimitOnly should be false when function concurrency limits are hit")
}
