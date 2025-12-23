package redis_state

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/VividCortex/ewma"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
	"gonum.org/v1/gonum/stat/sampleuv"
)

const (
	pkgName = "redis_state.state.execution.inngest"
)

type QueueManager interface {
	osqueue.JobQueueReader
	osqueue.Queue
	osqueue.QueueDirectAccess

	DequeueByJobID(ctx context.Context, jobID string, opts ...QueueOpOpt) error
	Dequeue(ctx context.Context, queueShard RedisQueueShard, i osqueue.QueueItem, opts ...dequeueOptionFn) error
	Requeue(ctx context.Context, queueShard RedisQueueShard, i osqueue.QueueItem, at time.Time, opts ...requeueOptionFn) error
	RequeueByJobID(ctx context.Context, queueShard RedisQueueShard, jobID string, at time.Time) error

	// ResetAttemptsByJobID sets retries to zero given a single job ID.  This is important for
	// checkpointing;  a single job becomes shared amongst many  steps.
	ResetAttemptsByJobID(ctx context.Context, shard string, jobID string) error

	// ItemsByPartition returns a queue item iterator for a function within a specific time range
	ItemsByPartition(ctx context.Context, queueShard RedisQueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*osqueue.QueueItem], error)
	// ItemsByBacklog returns a queue item iterator for a backlog within a specific time range
	ItemsByBacklog(ctx context.Context, queueShard RedisQueueShard, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*osqueue.QueueItem], error)
	// BacklogsByPartition returns an iterator for the partition's backlogs
	BacklogsByPartition(ctx context.Context, queueShard RedisQueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueBacklog], error)
	// BacklogSize retrieves the number of items in the specified backlog
	BacklogSize(ctx context.Context, queueShard RedisQueueShard, backlogID string) (int64, error)
	// PartitionByID retrieves the partition by the partition ID
	PartitionByID(ctx context.Context, queueShard RedisQueueShard, partitionID string) (*PartitionInspectionResult, error)
	// ItemByID retrieves the queue item by the jobID
	ItemByID(ctx context.Context, jobID string, opts ...QueueOpOpt) (*osqueue.QueueItem, error)
	// ItemExists checks if an item with jobID exists in the queue
	ItemExists(ctx context.Context, jobID string, opts ...QueueOpOpt) (bool, error)
	// ItemsByRunID retrieves all queue items via runID
	//
	// NOTE
	// The queue technically shouldn't know about runIDs, so we should make this more generic with certain type of indices in the future
	ItemsByRunID(ctx context.Context, runID ulid.ULID, opts ...QueueOpOpt) ([]*osqueue.QueueItem, error)

	// PartitionBacklogSize returns the point in time backlog size of the partition.
	// This will sum the size of all backlogs in that partition
	PartitionBacklogSize(ctx context.Context, partitionID string) (int64, error)

	// Total queue depth of all partitions including backlog and ready state items
	TotalSystemQueueDepth(ctx context.Context) (int64, error)

	// Shard returns the shard with the provided name if available
	Shard(ctx context.Context, shardName string) (RedisQueueShard, bool)
}

type RedisQueueShard struct {
	name        string
	RedisClient *QueueClient

	q *queue
}

func (q RedisQueueShard) Name() string {
	return q.name
}

func (q RedisQueueShard) Kind() enums.QueueShardKind {
	return enums.QueueShardKindRedis
}

func (q RedisQueueShard) Processor() osqueue.QueueProcessor {
	return q.q
}

func NewRedisQueue(options osqueue.QueueOptions, name string, queueClient *QueueClient) osqueue.QueueShard {
	q := &queue{
		itemIndexer:  QueueItemIndexerFunc,
		QueueOptions: options,
	}

	return RedisQueueShard{
		name:        name,
		RedisClient: queueClient,
		q:           q,
	}
}

type queue struct {
	osqueue.QueueOptions

	// itemIndexer returns indexes for a given queue item.
	itemIndexer QueueItemIndexer
}

// zsetKey represents the key used to store the zset for this partition's items.
// For default partitions, this is different to the ID (for backwards compatibility, it's just
// the fn ID without prefixes)
func partitionZsetKey(qp osqueue.QueuePartition, kg QueueKeyGenerator) string {
	// For system partitions, return zset using custom queueName
	if qp.IsSystem() {
		return kg.PartitionQueueSet(enums.PartitionTypeDefault, qp.Queue(), "")
	}

	// Backwards compatibility with old fn queues
	if qp.FunctionID != nil {
		// return the top-level function queue.
		return kg.PartitionQueueSet(enums.PartitionTypeDefault, qp.FunctionID.String(), "")
	}

	if qp.ID == "" {
		// return a blank queue key.  This is used for nil queue partitions.
		return kg.PartitionQueueSet(enums.PartitionTypeDefault, "-", "")
	}

	// qp.ID is already a properly defined key (concurrency key queues).
	return qp.ID
}

// concurrencyKey returns the single concurrency key for the given partition, depending
// on the partition type.  This is used to check the partition's in-progress items whilst
// requeueing partitions.
func partitionConcurrencyKey(qp osqueue.QueuePartition, kg QueueKeyGenerator) string {
	return fnConcurrencyKey(qp, kg)
}

// fnConcurrencyKey returns the concurrency key for a function scope limit, on the
// entire function (not custom keys)
func fnConcurrencyKey(qp osqueue.QueuePartition, kg QueueKeyGenerator) string {
	// Enable system partitions to use the queueName override instead of the fnId
	if qp.IsSystem() {
		return kg.Concurrency("p", qp.Queue())
	}

	if qp.FunctionID == nil {
		return kg.Concurrency("p", "-")
	}
	return kg.Concurrency("p", qp.FunctionID.String())
}

// acctConcurrencyKey returns the concurrency key for the account limit, on the
// entire account (not custom keys)
func acctConcurrencyKey(qp osqueue.QueuePartition, kg QueueKeyGenerator) string {
	// Enable system partitions to use the queueName override instead of the accountId
	if qp.IsSystem() {
		return kg.Concurrency("account", qp.Queue())
	}
	if qp.AccountID == uuid.Nil {
		return kg.Concurrency("account", "-")
	}
	return kg.Concurrency("account", qp.AccountID.String())
}

func partitionAccountInProgressLeasesKey(qp osqueue.QueuePartition, kg QueueKeyGenerator, cm constraintapi.RolloutKeyGenerator) string {
	if cm == nil {
		return kg.Concurrency("", "")
	}
	if qp.IsSystem() {
		return kg.Concurrency("", "")
	}
	if qp.AccountID == uuid.Nil {
		return kg.Concurrency("", "")
	}
	return cm.KeyInProgressLeasesAccount(qp.AccountID)
}

func shadowPartitionAccountInProgressLeasesKey(sp osqueue.QueueShadowPartition, kg QueueKeyGenerator, cm constraintapi.RolloutKeyGenerator) string {
	if cm == nil {
		return kg.Concurrency("", "")
	}
	if sp.SystemQueueName != nil {
		return kg.Concurrency("", "")
	}
	if sp.AccountID == nil {
		return kg.Concurrency("", "")
	}
	return cm.KeyInProgressLeasesAccount(*sp.AccountID)
}

func partitionFunctionInProgressLeasesKey(qp osqueue.QueuePartition, kg QueueKeyGenerator, cm constraintapi.RolloutKeyGenerator) string {
	if cm == nil {
		return kg.Concurrency("", "")
	}
	// Enable system partitions to use the queueName override instead of the fnId
	if qp.IsSystem() {
		return kg.Concurrency("", "")
	}
	if qp.FunctionID == nil || qp.AccountID == uuid.Nil {
		return kg.Concurrency("", "")
	}
	return cm.KeyInProgressLeasesFunction(qp.AccountID, *qp.FunctionID)
}

func shadowPartitionFunctionInProgressLeasesKey(sp osqueue.QueueShadowPartition, kg QueueKeyGenerator, cm constraintapi.RolloutKeyGenerator) string {
	if cm == nil {
		return kg.Concurrency("", "")
	}
	// Enable system partitions to use the queueName override instead of the fnId
	if sp.SystemQueueName != nil {
		return kg.Concurrency("", "")
	}
	if sp.FunctionID == nil || sp.AccountID == nil {
		return kg.Concurrency("", "")
	}
	return cm.KeyInProgressLeasesFunction(*sp.AccountID, *sp.FunctionID)
}

func (q *queue) EnqueueItem(ctx context.Context, shard osqueue.QueueShard, i osqueue.QueueItem, at time.Time, opts osqueue.EnqueueOpts) (osqueue.QueueItem, error) {
	l := logger.StdlibLogger(ctx)

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "EnqueueItem"), redis_telemetry.ScopeQueue)

	if shard.Kind() != enums.QueueShardKindRedis {
		return osqueue.QueueItem{}, fmt.Errorf("unsupported queue shard kind for EnqueueItem: %s", shard.Kind)
	}

	redisShard, ok := shard.(RedisQueueShard)
	if !ok {
		return osqueue.QueueItem{}, fmt.Errorf("expected RedisQueueShard: %s", shard.Kind)
	}

	kg := redisShard.RedisClient.kg

	if len(i.ID) == 0 {
		i.SetID(ctx, ulid.MustNew(ulid.Now(), rnd).String())
	} else {
		if !opts.PassthroughJobId {
			i.SetID(ctx, i.ID)
		}
	}

	now := q.Clock.Now()

	// XXX: If the length of ID >= max, error.
	if i.WallTimeMS == 0 {
		i.WallTimeMS = at.UnixMilli()
	}

	if at.Before(now) {
		// Normalize to now to minimize latency.
		i.WallTimeMS = now.UnixMilli()
	}

	// Add the At timestamp, if not included.
	if i.AtMS == 0 {
		i.AtMS = at.UnixMilli()
	}

	if i.Data.JobID == nil {
		i.Data.JobID = &i.ID
	}

	partitionTime := at
	if at.Before(now) {
		// We don't want to enqueue partitions (pointers to fns) before now.
		// Doing so allows users to stay at the front of the queue for
		// leases.
		partitionTime = q.Clock.Now()
	}

	i.EnqueuedAt = now.UnixMilli()

	defaultPartition := osqueue.ItemPartition(ctx, shard, i)

	isSystemPartition := defaultPartition.IsSystem()

	if defaultPartition.AccountID == uuid.Nil && !isSystemPartition {
		l.Warn("attempting to enqueue item to non-system partition without account ID", "item", i)
	}

	enqueueToBacklogs := q.itemEnableKeyQueues(ctx, i)

	var backlog osqueue.QueueBacklog
	var shadowPartition osqueue.QueueShadowPartition
	if enqueueToBacklogs {
		backlog = osqueue.ItemBacklog(ctx, i)
		shadowPartition = osqueue.ItemShadowPartition(ctx, i)
	}

	keys := []string{
		kg.QueueItem(),            // Queue item
		kg.PartitionItem(),        // Partition item, map
		kg.GlobalPartitionIndex(), // Global partition queue
		kg.GlobalAccountIndex(),
		kg.AccountPartitionIndex(i.Data.Identifier.AccountID), // new queue items always contain the account ID
		kg.Idempotency(i.ID),

		// Add all 3 partition sets
		partitionZsetKey(defaultPartition, kg),

		// Key queues v2
		kg.BacklogSet(backlog.BacklogID),
		kg.BacklogMeta(),
		kg.GlobalShadowPartitionSet(),
		kg.ShadowPartitionSet(shadowPartition.PartitionID),
		kg.ShadowPartitionMeta(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(i.Data.Identifier.AccountID), // will be empty for system queues

		// Key queue Normalization
		kg.BacklogSet(opts.NormalizeFromBacklogID),
		kg.PartitionNormalizeSet(shadowPartition.PartitionID),
		kg.AccountNormalizeSet(i.Data.Identifier.AccountID),
		kg.GlobalAccountNormalizeSet(),

		// Singletons
		kg.SingletonRunKey(i.Data.Identifier.RunID.String()),
		kg.SingletonKey(i.Data.Singleton),
	}
	// Append indexes
	for _, idx := range q.itemIndexer(ctx, i, redisShard.RedisClient.kg) {
		if idx != "" {
			keys = append(keys, idx)
		}
	}

	enqueueToBacklogsVal := "0"
	if enqueueToBacklogs {
		enqueueToBacklogsVal = "1"
	}

	args, err := StrSlice([]any{
		i,
		i.ID,
		at.UnixMilli(),
		partitionTime.Unix(),
		now.UnixMilli(),
		defaultPartition,
		defaultPartition.ID,
		i.Data.Identifier.AccountID.String(),
		i.Data.Identifier.RunID.String(),

		enqueueToBacklogsVal,
		shadowPartition,
		backlog,
		backlog.BacklogID,

		opts.NormalizeFromBacklogID,
	})
	if err != nil {
		return i, err
	}

	l.Trace("enqueue item",
		"id", i.ID,
		"kind", i.Data.Kind,
		"time", at.Format(time.StampMilli),
		"partition_time", partitionTime.Format(time.StampMilli),
		"partition", shadowPartition.PartitionID,
		"backlog", enqueueToBacklogs,
	)

	status, err := scripts["queue/enqueue"].Exec(
		redis_telemetry.WithScriptName(ctx, "enqueue"),
		redisShard.RedisClient.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return i, fmt.Errorf("error enqueueing item: %w", err)
	}
	switch status {
	case 0:
		// Hint to executor that we should refill if the item has no delay
		refillSoon := i.ExpectedDelay() < osqueue.ShadowPartitionLookahead
		if enqueueToBacklogs && refillSoon {
			q.addShadowContinue(ctx, &shadowPartition, 0)
		}

		return i, nil
	case 1:
		return i, osqueue.ErrQueueItemExists
	case 2:
		return i, osqueue.ErrQueueItemSingletonExists
	default:
		return i, fmt.Errorf("unknown response enqueueing item: %v (%T)", status, status)
	}
}

// dropPartitionPointerIfEmpty atomically drops a pointer queue member if the associated
// ZSET is empty. This is used to ensure that we don't have pointers to empty ZSETs, in case
// the cleanup process fails.
func (q *queue) dropPartitionPointerIfEmpty(ctx context.Context, shard RedisQueueShard, keyIndex, keyPartition, indexMember string) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SetFunctionPaused"), redis_telemetry.ScopeQueue)

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return nil
	}

	keys := []string{keyIndex, keyPartition}
	args, err := StrSlice([]any{
		indexMember,
	})
	if err != nil {
		return err
	}

	status, err := scripts["queue/dropPartitionPointerIfEmpty"].Exec(
		redis_telemetry.WithScriptName(ctx, "dropPartitionPointerIfEmpty"),
		shard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error dropping pointer %q from %q if %q was empty: %w", indexMember, keyIndex, keyPartition, err)
	}
	switch status {
	case 0, 1:
		return nil
	default:
		return fmt.Errorf("unknown response dropping pointer if empty: %d", status)
	}
}

func (q *queue) SetFunctionMigrate(ctx context.Context, sourceShard string, fnID uuid.UUID, migrateLockUntil *time.Time) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SetFunctionMigrate"), redis_telemetry.ScopeQueue)

	if q.queueShardClients == nil {
		return fmt.Errorf("no queue shard clients are available")
	}

	shard, ok := q.queueShardClients[sourceShard]
	if !ok {
		return fmt.Errorf("no queue shard available for '%s'", sourceShard)
	}

	client := shard.RedisClient.Client()
	kg := shard.RedisClient.KeyGenerator()

	key := kg.QueueMigrationLock(fnID)
	if migrateLockUntil == nil {
		cmd := client.B().Del().Key(key).Build()
		err := client.Do(ctx, cmd).Error()
		if err != nil {
			return fmt.Errorf("could not set migration lock: %w", err)
		}
	} else {
		lockID, err := ulid.New(ulid.Timestamp(*migrateLockUntil), crand.Reader)
		if err != nil {
			return fmt.Errorf("could not generate lockID: %w", err)
		}

		cmd := client.B().Set().Key(key).Value(lockID.String()).Exat(*migrateLockUntil).Build()
		err = client.Do(ctx, cmd).Error()
		if err != nil {
			return fmt.Errorf("could not remove migration lock: %w", err)
		}
	}

	return nil
}

func (q *queue) Migrate(ctx context.Context, sourceShardName string, fnID uuid.UUID, limit int64, concurrency int, handler osqueue.QueueMigrationHandler) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "MigrationPeek"), redis_telemetry.ScopeQueue)

	shard, ok := q.queueShardClients[sourceShardName]
	if !ok {
		return -1, fmt.Errorf("no queue shard available for '%s'", sourceShardName)
	}

	from := time.Time{}
	// setting it to 5 years ahead should be enough to cover all queue items in the partition
	until := q.clock.Now().Add(24 * time.Hour * 365 * 5)
	items, err := q.ItemsByPartition(ctx, shard, fnID.String(), from, until,
		WithQueueItemIterBatchSize(limit),
	)
	if err != nil {
		// the partition doesn't exist, meaning there are no workloads
		if errors.Is(err, rueidis.Nil) {
			return 0, nil
		}

		return -1, fmt.Errorf("error preparing partition iteration: %w", err)
	}

	// Should process in order because we don't want out of order execution when moved over
	var processed int64

	process := func(qi *osqueue.QueueItem) error {
		if err := handler(ctx, qi); err != nil {
			return err
		}

		if err := q.Dequeue(ctx, shard, *qi); err != nil {
			q.log.ReportError(err, "error dequeueing queue item after migration")
		}

		atomic.AddInt64(&processed, 1)
		return nil
	}

	if concurrency > 0 {
		eg := errgroup.Group{}
		eg.SetLimit(concurrency)

		for qi := range items {
			i := qi
			eg.Go(func() error {
				return process(i)
			})
		}

		err := eg.Wait()
		if err != nil {
			return atomic.LoadInt64(&processed), err
		}

		return atomic.LoadInt64(&processed), nil
	}

	for qi := range items {
		if err := process(qi); err != nil {
			return processed, err
		}
	}

	return atomic.LoadInt64(&processed), nil
}

func (q *queue) RemoveQueueItem(ctx context.Context, shardName string, partitionKey string, itemID string) error {
	queueShard, ok := q.queueShardClients[shardName]
	if !ok {
		return fmt.Errorf("queue shard not found %q", shardName)
	}
	return q.removeQueueItem(ctx, queueShard, partitionKey, itemID)
}

// removeQueueItem attempts to remove a specific item in the target queue shard
// and also remove it from the queue item hash as well
func (q *queue) removeQueueItem(ctx context.Context, shard RedisQueueShard, partitionKey string, itemID string) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "removeQueueItem"), redis_telemetry.ScopeQueue)

	keys := []string{
		partitionKey,
		shard.RedisClient.kg.QueueItem(),
	}
	args := []string{itemID}

	code, err := scripts["queue/removeItem"].Exec(
		ctx,
		shard.RedisClient.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error deleting queue item: %w", err)
	}

	switch code {
	case 0:
		q.log.Debug("removed queue item", "item_id", itemID)

		return nil
	default:
		return fmt.Errorf("unknown status when attempting to remove item: %d", code)
	}
}

func (q *queue) LoadQueueItem(ctx context.Context, shardName string, itemID string) (*osqueue.QueueItem, error) {
	queueShard, ok := q.queueShardClients[shardName]
	if !ok {
		return nil, fmt.Errorf("queue shard not found %q", shardName)
	}

	kg := queueShard.RedisClient.KeyGenerator()
	client := queueShard.RedisClient.Client()

	queueItemStr, err := client.Do(ctx, client.B().Hget().Key(kg.QueueItem()).Field(itemID).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, ErrQueueItemNotFound
		}

		return nil, fmt.Errorf("could not load queue item: %w", err)
	}

	qi := &osqueue.QueueItem{}
	if err := json.Unmarshal([]byte(queueItemStr), qi); err != nil {
		return nil, fmt.Errorf("error unmarshalling loaded queue item: %w", err)
	}

	return qi, nil
}

// Peek takes n items from a queue, up until QueuePeekMax.  For peeking workflow/
// function jobs the queue name must be the ID of the workflow;  each workflow has
// its own queue of jobs using its ID as the queue name.
//
// If limit is -1, this will return the first unleased item - representing the next available item in the
// queue.
func (q *queue) Peek(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*osqueue.QueueItem, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Peek"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for Peek: %s", q.primaryQueueShard.Kind)
	}

	if partition == nil {
		return nil, fmt.Errorf("expected partition to be set")
	}

	// Check whether limit is -1, peeking next available time
	isPeekNext := limit == -1

	if limit > AbsoluteQueuePeekMax {
		// Lua's max unpack() length is 8000; don't allow users to peek more than
		// 1k at a time regardless.
		limit = AbsoluteQueuePeekMax
	}
	if limit > q.peekMax {
		limit = q.peekMax
	}
	if limit <= 0 {
		limit = q.peekMin
	}
	if isPeekNext {
		limit = 1
	}

	partitionKey := partition.zsetKey(q.primaryQueueShard.RedisClient.kg)
	return q.peek(
		ctx,
		q.primaryQueueShard,
		peekOpts{
			Limit:        limit,
			Until:        until,
			PartitionKey: partitionKey,
			PartitionID:  partition.ID,
		},
	)
}

func (q *queue) PeekRandom(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*osqueue.QueueItem, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Peek"), redis_telemetry.ScopeQueue)
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for Peek: %s", q.primaryQueueShard.Kind)
	}
	if partition == nil {
		return nil, fmt.Errorf("expected partition to be set")
	}
	if limit > AbsoluteQueuePeekMax {
		// Lua's max unpack() length is 8000; don't allow users to peek more than
		// 1k at a time regardless.
		limit = AbsoluteQueuePeekMax
	}
	if limit > q.peekMax {
		limit = q.peekMax
	}
	if limit <= 0 {
		limit = q.peekMin
	}
	partitionKey := partition.zsetKey(q.primaryQueueShard.RedisClient.kg)
	return q.peek(
		ctx,
		q.primaryQueueShard,
		peekOpts{
			Limit:        limit,
			Until:        until,
			PartitionKey: partitionKey,
			PartitionID:  partition.ID,
			Random:       true,
		},
	)
}

type peekOpts struct {
	PartitionID  string
	PartitionKey string
	Random       bool
	From         *time.Time
	Until        time.Time
	Limit        int64
}

func (q *queue) peek(ctx context.Context, shard RedisQueueShard, opts peekOpts) ([]*osqueue.QueueItem, error) {
	from := "-inf"
	if opts.From != nil && !opts.From.IsZero() {
		from = strconv.Itoa(int(opts.From.UnixMilli()))
	}

	until := "+inf"
	if opts.Until.UnixMilli() > 0 {
		until = strconv.Itoa(int(opts.Until.UnixMilli()))
	}

	randomOffset := "0"
	if opts.Random {
		randomOffset = "1"
	}

	keys := []string{
		opts.PartitionKey,
		shard.RedisClient.kg.QueueItem(),
	}
	args, err := StrSlice([]any{
		from,
		until,
		opts.Limit,
		randomOffset,
	})
	if err != nil {
		return nil, err
	}

	peekRet, err := scripts["queue/peek"].Exec(
		redis_telemetry.WithScriptName(ctx, "peek"),
		shard.RedisClient.unshardedRc,
		keys,
		args,
	).ToAny()
	if err != nil {
		return nil, fmt.Errorf("error peeking queue items: %w", err)
	}

	returnedSet, ok := peekRet.([]any)
	if !ok {
		return nil, fmt.Errorf("unknown return type from peek: %T", peekRet)
	}

	var potentiallyMissingItems, allQueueItemIds []any
	if len(returnedSet) == 2 {
		potentiallyMissingItems, ok = returnedSet[0].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected first item in set returned from peek: %T", peekRet)
		}

		allQueueItemIds, ok = returnedSet[1].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected first item in set returned from peek: %T", peekRet)
		}
	} else if len(returnedSet) != 0 {
		return nil, fmt.Errorf("expected zero or two items in set returned by peek: %v", returnedSet)
	}

	items := make([]any, 0, len(allQueueItemIds))
	missingQueueItems := make([]string, 0, len(allQueueItemIds))
	for idx, itemId := range allQueueItemIds {
		item := potentiallyMissingItems[idx]
		if item == nil {
			if itemId == nil {
				return nil, fmt.Errorf("encountered nil queue item key in partition queue %q", opts.PartitionKey)
			}

			str, ok := itemId.(string)
			if !ok {
				return nil, fmt.Errorf("encountered non-string queue item key in partition queue %q", opts.PartitionKey)
			}

			missingQueueItems = append(missingQueueItems, str)
		} else {
			items = append(items, item)
		}
	}

	if len(missingQueueItems) > 0 {
		q.log.Warn("encountered missing queue items in partition queue",
			"key", opts.PartitionKey,
			"items", missingQueueItems,
		)

		eg := errgroup.Group{}
		for _, missingItemId := range missingQueueItems {
			id := missingItemId
			eg.Go(func() error {
				return q.removeQueueItem(ctx, shard, opts.PartitionKey, id)
			})
		}

		if err := eg.Wait(); err != nil {
			return nil, fmt.Errorf("error cleaning up nil partitions in account pointer queue: %w", err)
		}
	}

	return util.ParallelDecode(items, func(val any, _ int) (*osqueue.QueueItem, bool, error) {
		if val == nil {
			q.log.Error("nil item value in peek response", "partition", opts.PartitionKey)
			return nil, true, nil
		}

		str, ok := val.(string)
		if !ok {
			return nil, false, fmt.Errorf("non-string value in peek response: %T", val)
		}

		if str == "" {
			return nil, false, fmt.Errorf("received empty string in decode queue item from peek")
		}

		qi := &osqueue.QueueItem{}
		if err := json.Unmarshal(unsafe.Slice(unsafe.StringData(str), len(str)), qi); err != nil {
			return nil, false, fmt.Errorf("error unmarshalling peeked queue item: %w", err)
		}

		now := q.clock.Now()
		if qi.IsLeased(now) {
			metrics.IncrQueuePeekLeaseContentionCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					// "partition_id": opts.PartitionID,
					"queue_shard": shard.Name,
				},
			})

			// Leased item, don't return.
			return nil, true, nil
		}

		// The nested osqueue.Item never has an ID set;  always re-set it
		qi.Data.JobID = &qi.ID
		return qi, false, nil
	})
}

func (q *queue) ResetAttemptsByJobID(ctx context.Context, shardName string, jobID string) error {
	queueShard, ok := q.queueShardClients[shardName]
	if !ok {
		return fmt.Errorf("queue shard not found %q", shardName)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ResetAttemptsByJobID"), redis_telemetry.ScopeQueue)

	if queueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for RequeueByJobID: %s", queueShard.Kind)
	}

	// NOTE: We expect that the job ID is the hashed, stored ID in the queue already.

	keys := []string{
		queueShard.RedisClient.kg.QueueItem(),
	}

	args, err := StrSlice([]any{jobID})
	if err != nil {
		return err
	}
	status, err := scripts["queue/resetAttempts"].Exec(
		redis_telemetry.WithScriptName(ctx, "requeueByID"),
		queueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		q.log.Error("error requeueing queue item by JobID",
			"error", err,
			"job_id", jobID,
		)
		return fmt.Errorf("error requeueing item: %w", err)
	}
	switch status {
	case 0:
		return nil
	case -1:
		return ErrQueueItemNotFound
	default:
		return fmt.Errorf("unknown requeue by id response: %d", status)
	}
}

// RequeueByJobID requeues a job for a specific time given a partition name and job ID.
//
// If the queue item referenced by the job ID is not outstanding (ie. it has a lease, is in
// progress, or doesn't exist) this returns an error.
//
// Note: This only works with items that directly go into ready queues (system queues).
func (q *queue) RequeueByJobID(ctx context.Context, queueShard RedisQueueShard, jobID string, at time.Time) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "RequeueByJobID"), redis_telemetry.ScopeQueue)

	if queueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for RequeueByJobID: %s", queueShard.Kind)
	}

	jobID = osqueue.HashID(ctx, jobID)

	// Find the queue item so that we can fetch the shard info.
	i := osqueue.QueueItem{}
	if err := queueShard.RedisClient.unshardedRc.Do(ctx, queueShard.RedisClient.unshardedRc.B().Hget().Key(queueShard.RedisClient.kg.QueueItem()).Field(jobID).Build()).DecodeJSON(&i); err != nil {
		return err
	}

	// Don't requeue before now.
	now := q.clock.Now()
	if at.Before(now) {
		at = now
	}

	// Remove all items from all partitions.  For this, we need all partitions for
	// the queue item instead of just the partition passed via args.
	//
	// This is because a single queue item may be present in more than one queue.
	fnPartition := q.ItemPartition(ctx, queueShard, i)

	keys := []string{
		queueShard.RedisClient.kg.QueueItem(),
		queueShard.RedisClient.kg.PartitionItem(), // Partition item, map
		queueShard.RedisClient.kg.GlobalPartitionIndex(),
		queueShard.RedisClient.kg.GlobalAccountIndex(),
		queueShard.RedisClient.kg.AccountPartitionIndex(i.Data.Identifier.AccountID),

		fnPartition.zsetKey(queueShard.RedisClient.kg),
	}

	args, err := StrSlice([]any{
		jobID,
		strconv.Itoa(int(at.UnixMilli())),
		strconv.Itoa(int(now.UnixMilli())),
		fnPartition,
		fnPartition.ID,
		i.Data.Identifier.AccountID.String(),
	})
	if err != nil {
		return err
	}
	status, err := scripts["queue/requeueByID"].Exec(
		redis_telemetry.WithScriptName(ctx, "requeueByID"),
		queueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		q.log.Error("error requeueing queue item by JobID",
			"error", err,
			"item", i,
			"fnPartition", fnPartition,
		)
		return fmt.Errorf("error requeueing item: %w", err)
	}
	switch status {
	case 0:
		return nil
	case -1:
		return ErrQueueItemNotFound
	case -2:
		return ErrQueueItemAlreadyLeased
	default:
		return fmt.Errorf("unknown requeue by id response: %d", status)
	}
}

func (q *queue) itemEnableKeyQueues(ctx context.Context, item osqueue.QueueItem) bool {
	isSystem := item.QueueName != nil || item.Data.QueueName != nil
	if isSystem {
		return false
	}

	if item.Data.Identifier.AccountID != uuid.Nil && q.allowKeyQueues != nil {
		return q.allowKeyQueues(ctx, item.Data.Identifier.AccountID, item.FunctionID)
	}

	return false
}

// Lease temporarily dequeues an item from the queue by obtaining a lease, preventing
// other workers from working on this queue item at the same time.
//
// Obtaining a lease updates the vesting time for the queue item until now() +
// lease duration. This returns the newly acquired lease ID on success.
func (q *queue) Lease(
	ctx context.Context,
	item osqueue.QueueItem,
	leaseDuration time.Duration,
	now time.Time,
	denies *osqueue.LeaseDenies,
	options ...leaseOptionFn,
) (*ulid.ULID, error) {
	o := &leaseOptions{}
	for _, opt := range options {
		opt(o)
	}

	if o.backlog.BacklogID == "" {
		o.backlog = q.ItemBacklog(ctx, item)
	}

	if o.sp.PartitionID == "" {
		o.sp = q.ItemShadowPartition(ctx, item)
	}

	if o.constraints.FunctionVersion == 0 {
		o.constraints = q.partitionConstraintConfigGetter(ctx, o.sp.Identifier())
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Lease"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for Lease: %s", q.primaryQueueShard.Kind)
	}

	kg := q.primaryQueueShard.RedisClient.kg

	enableKeyQueues := q.itemEnableKeyQueues(ctx, item)

	refilledFromBacklog := enableKeyQueues && item.RefilledFrom != ""

	// Disable constraint checks and updates under certain circumstances
	// - For system queues
	// - When a valid capacity lease is held
	checkConstraints := !o.disableConstraintChecks

	if checkConstraints {
		if item.Data.Throttle != nil && denies != nil && denies.denyThrottle(item.Data.Throttle.Key) {
			return nil, ErrQueueItemThrottled
		}

		// Check to see if this key has already been denied in the lease iteration.
		// If partition concurrency limits were encountered previously, fail early.
		if denies != nil && denies.denyConcurrency(item.FunctionID.String()) {
			// Note that we do not need to wrap the key as the key is already present.
			return nil, ErrPartitionConcurrencyLimit
		}

		// Same for account concurrency limits
		if denies != nil && denies.denyConcurrency(item.Data.Identifier.AccountID.String()) {
			return nil, ErrAccountConcurrencyLimit
		}
	}

	if checkConstraints {
		// Check to see if this key has already been denied in the lease iteration.
		// If so, fail early.
		if denies != nil && len(o.backlog.ConcurrencyKeys) > 0 && denies.denyConcurrency(o.backlog.customConcurrencyKeyID(1)) {
			return nil, ErrConcurrencyLimitCustomKey
		}

		// Check to see if this key has already been denied in the lease iteration.
		// If so, fail early.
		if denies != nil && len(o.backlog.ConcurrencyKeys) > 1 && denies.denyConcurrency(o.backlog.customConcurrencyKeyID(2)) {
			return nil, ErrConcurrencyLimitCustomKey
		}
	}

	leaseID, err := ulid.New(ulid.Timestamp(now.Add(leaseDuration).UTC()), rnd)
	if err != nil {
		return nil, fmt.Errorf("error generating id: %w", err)
	}

	refilledFromBacklogVal := "0"
	if refilledFromBacklog {
		refilledFromBacklogVal = "1"
	}

	checkConstraintsVal := "0"
	if checkConstraints {
		checkConstraintsVal = "1"
	}

	checkThrottle := checkConstraints && o.constraints.Throttle != nil && item.Data.Throttle != nil

	enableThrottleInstrumentation := checkThrottle &&
		o.sp.AccountID != nil &&
		o.sp.FunctionID != nil &&
		q.enableThrottleInstrumentation != nil &&
		q.enableThrottleInstrumentation(ctx, *o.sp.AccountID, *o.sp.FunctionID)

	// Check if throttle is outdated
	if outdatedThrottleReason := o.constraints.HasOutdatedThrottle(item); outdatedThrottleReason != enums.OutdatedThrottleReasonNone {
		// TODO: Re-evaluate throttle with event data
		metrics.IncrQueueThrottleKeyExpressionMismatchCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"reason": outdatedThrottleReason.String(),
			},
		})
	}

	keys := []string{
		kg.QueueItem(),
		kg.ConcurrencyIndex(),

		o.sp.readyQueueKey(kg),

		// In progress (concurrency) ZSETs
		o.sp.accountInProgressKey(kg),
		o.sp.inProgressKey(kg),
		o.backlog.customKeyInProgress(kg, 1),
		o.backlog.customKeyInProgress(kg, 2),

		// Active set keys (ready + in progress)
		o.sp.accountActiveKey(kg),
		o.sp.activeKey(kg),
		o.backlog.customKeyActive(kg, 1),
		o.backlog.customKeyActive(kg, 2),
		o.backlog.activeKey(kg),

		// Active run sets
		kg.RunActiveSet(item.Data.Identifier.RunID), // Set for active items in run
		o.sp.accountActiveRunKey(kg),                // Set for active runs in account
		o.sp.activeRunKey(kg),                       // Set for active runs in partition
		o.backlog.customKeyActiveRuns(kg, 1),        // Set for active runs with custom concurrency key 1
		o.backlog.customKeyActiveRuns(kg, 2),        // Set for active runs with custom concurrency key 2

		kg.ThrottleKey(item.Data.Throttle),

		// Constraint API rollout
		o.sp.acctInProgressLeasesKey(kg, q.capacityManager),
		o.sp.fnInProgressLeasesKey(kg, q.capacityManager),
		o.backlog.inProgressLeasesCustomKey(q.capacityManager, kg, o.sp.AccountID, 1),
		o.backlog.inProgressLeasesCustomKey(q.capacityManager, kg, o.sp.AccountID, 2),
		q.keyConstraintCheckIdempotency(o.sp.AccountID, item.ID),

		kg.PartitionScavengerIndex(o.sp.PartitionID),
	}

	partConcurrency := o.constraints.Concurrency.FunctionConcurrency
	if o.sp.SystemQueueName != nil {
		partConcurrency = o.constraints.Concurrency.SystemConcurrency
	}

	marshaledConstraints, err := json.Marshal(o.constraints)
	if err != nil {
		return nil, fmt.Errorf("could not marshal constraints: %w", err)
	}

	args, err := StrSlice([]any{
		item.ID,
		o.sp.PartitionID,
		item.Data.Identifier.AccountID,
		item.Data.Identifier.RunID.String(),

		leaseID.String(),
		now.UnixMilli(),

		// Concurrency limits
		o.constraints.Concurrency.AccountConcurrency,
		partConcurrency,
		o.constraints.CustomConcurrencyLimit(1),
		o.constraints.CustomConcurrencyLimit(2),
		string(marshaledConstraints),

		// Key queues v2
		refilledFromBacklogVal,

		checkConstraintsVal,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/lease"].Exec(
		redis_telemetry.WithScriptName(ctx, "lease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).ToInt64()
	if err != nil {
		return nil, fmt.Errorf("error leasing queue item: %w", err)
	}

	itemDelay := item.ExpectedDelay()
	metrics.HistogramQueueOperationDelay(ctx, itemDelay, metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.primaryQueueShard.Name,
			"op":          "item",
		},
	},
	)

	l := q.log.With("item_delay", itemDelay.String())

	refillDelay := item.RefillDelay()
	metrics.HistogramQueueOperationDelay(ctx, refillDelay, metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.primaryQueueShard.Name,
			"op":          "refill",
		},
	},
	)
	l = l.With("refill_delay", refillDelay.String())

	// leaseDelay is the time between refilling and leasing
	leaseDelay := item.LeaseDelay(now)
	metrics.HistogramQueueOperationDelay(ctx, leaseDelay, metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.primaryQueueShard.Name,
			"op":          "lease",
		},
	},
	)
	l = l.With("lease_delay", leaseDelay.String())

	l.Trace("leasing item",
		"id", item.ID,
		"kind", item.Data.Kind,
		"lease_id", leaseID.String(),
		"partition_id", o.sp.PartitionID,
		"item_delay", itemDelay.String(),
		"refilled", refilledFromBacklog,
		"check", checkConstraints,
		"status", status,
	)

	switch status {
	case 0, 1:
		if enableThrottleInstrumentation {
			statusStr := "allowed"
			if status == 1 {
				statusStr = "burst"
			}
			metrics.IncrQueueThrottleStatus(ctx, 1, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"account_id":  *o.sp.AccountID,
					"function_id": *o.sp.FunctionID,
					"status":      statusStr,
				},
			})
		}

		return &leaseID, nil
	case -1:
		return nil, ErrQueueItemNotFound
	case -2:
		return nil, ErrQueueItemAlreadyLeased
	case -3:
		// This partition is reused for function partitions without keys, system partions,
		// and potentially concurrency key partitions. Errors should be returned based on
		// the partition type

		if o.sp.SystemQueueName != nil {
			return nil, newKeyError(ErrSystemConcurrencyLimit, o.sp.PartitionID)
		}

		return nil, newKeyError(ErrPartitionConcurrencyLimit, item.FunctionID.String())
	case -4:
		return nil, newKeyError(ErrConcurrencyLimitCustomKey, o.backlog.customConcurrencyKeyID(1))
	case -5:
		return nil, newKeyError(ErrConcurrencyLimitCustomKey, o.backlog.customConcurrencyKeyID(2))
	case -6:
		return nil, newKeyError(ErrAccountConcurrencyLimit, item.Data.Identifier.AccountID.String())
	case -7:
		if enableThrottleInstrumentation {
			status := "throttled"
			metrics.IncrQueueThrottleStatus(ctx, 1, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"account_id":  *o.sp.AccountID,
					"function_id": *o.sp.FunctionID,
					"status":      status,
				},
			})
		}

		if o.constraints.Throttle == nil {
			// This should never happen, as the throttle key is nil.
			return nil, fmt.Errorf("lease attempted throttle with nil throttle config: %#v", item)
		}
		return nil, newKeyError(ErrQueueItemThrottled, item.Data.Throttle.Key)
	default:
		return nil, fmt.Errorf("unknown response leasing item: %d", status)
	}
}

// ExtendLease extens the lease for a given queue item, given the queue item is currently
// leased with the given ID.  This returns a new lease ID if the lease is successfully ended.
//
// The existing lease ID must be passed in so that we can guarantee that the worker
// renewing the lease still owns the lease.
//
// Renewing a lease updates the vesting time for the queue item until now() +
// lease duration. This returns the newly acquired lease ID on success.
func (q *queue) ExtendLease(ctx context.Context, i osqueue.QueueItem, leaseID ulid.ULID, duration time.Duration, options ...extendLeaseOptionFn) (*ulid.ULID, error) {
	o := &extendLeaseOptions{}
	for _, opt := range options {
		opt(o)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ExtendLease"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ExtendLease: %s", q.primaryQueueShard.Kind)
	}

	kg := q.primaryQueueShard.RedisClient.kg

	newLeaseID, err := ulid.New(ulid.Timestamp(q.clock.Now().Add(duration).UTC()), rnd)
	if err != nil {
		return nil, fmt.Errorf("error generating id: %w", err)
	}

	backlog := q.ItemBacklog(ctx, i)
	partition := q.ItemShadowPartition(ctx, i)

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.QueueItem(),
		// And pass in the key queue's concurrency keys.
		partition.inProgressKey(kg),
		backlog.customKeyInProgress(kg, 1),
		backlog.customKeyInProgress(kg, 2),
		partition.accountInProgressKey(kg),
		q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex(),
		kg.PartitionScavengerIndex(partition.PartitionID),
	}

	updateConstraintStateVal := "1"
	if o.disableConstraintUpdates {
		updateConstraintStateVal = "0"
	}

	args, err := StrSlice([]any{
		i.ID,
		leaseID.String(),
		newLeaseID.String(),
		partition.PartitionID,
		updateConstraintStateVal,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/extendLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "extendLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error extending lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case 1:
		return nil, ErrQueueItemNotFound
	case 2:
		return nil, ErrQueueItemNotLeased
	case 3:
		return nil, ErrQueueItemLeaseMismatch
	default:
		return nil, fmt.Errorf("unknown response extending lease: %d", status)
	}
}

func (q *queue) peekGlobalNormalizeAccounts(ctx context.Context, until time.Time, limit int64) ([]uuid.UUID, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for peekGlobalNormalizeAccounts: %s", q.primaryQueueShard.Kind)
	}

	rc := q.primaryQueueShard.RedisClient

	p := peeker[QueueBacklog]{
		q:                      q,
		opName:                 "peekGlobalNormalizeAccounts",
		max:                    NormalizeAccountPeekMax,
		isMillisecondPrecision: true,
	}

	return p.peekUUIDPointer(ctx, rc.kg.GlobalAccountNormalizeSet(), true, until, limit)
}

// PartitionLease leases a partition for a given workflow ID.  It returns the new lease ID.
//
// NOTE: This does not check the queue/partition name against allow or denylists;  it assumes
// that the worker always wants to lease the given queue.  Filtering must be done when peeking
// when running a worker.
func (q *queue) PartitionLease(
	ctx context.Context,
	p *QueuePartition,
	duration time.Duration,
	options ...partitionLeaseOpt,
) (*ulid.ULID, int, error) {
	o := &partitionLeaseOptions{}
	for _, opt := range options {
		opt(o)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PartitionLease"), redis_telemetry.ScopeQueue)

	shard := q.primaryQueueShard

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return nil, 0, fmt.Errorf("unsupported queue shard kind for PartitionLease: %s", shard.Kind)
	}

	kg := shard.RedisClient.kg

	// Fetch partition constraints with a timeout
	dbCtx, dbCtxCancel := context.WithTimeout(ctx, dbReadTimeout)
	constraints := q.partitionConstraintConfigGetter(dbCtx, p.Identifier())

	if dbCtx.Err() == context.DeadlineExceeded {
		metrics.IncrQueueDatabaseContextTimeoutCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"operation": "partition_constraint_config_getter",
			},
		})
	}

	dbCtxCancel()

	var accountLimit, functionLimit int
	if p.IsSystem() {
		accountLimit = constraints.Concurrency.SystemConcurrency
		functionLimit = constraints.Concurrency.SystemConcurrency
	} else {
		accountLimit = constraints.Concurrency.AccountConcurrency
		functionLimit = constraints.Concurrency.FunctionConcurrency
	}

	// XXX: Check for function throttling prior to leasing;  if it's throttled we can requeue
	// the pointer and back off.  A question here is enqueuing new items onto the partition
	// will reset the pointer update, leading to thrash.
	now := q.clock.Now()
	leaseExpires := now.Add(duration).UTC().Truncate(time.Millisecond)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpires), rnd)
	if err != nil {
		return nil, 0, fmt.Errorf("error generating id: %w", err)
	}

	disableLeaseChecks := p.IsSystem()
	if o.disableLeaseChecks {
		disableLeaseChecks = o.disableLeaseChecks
	}

	disableLeaseChecksVal := "0"
	if disableLeaseChecks {
		disableLeaseChecksVal = "1"
	}

	keys := []string{
		kg.PartitionItem(),
		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		// NOTE: Old partitions will _not_ have an account ID until the next enqueue on the new code.
		// Until this, we may not use account queues at all, as we cannot properly clean up
		// here without knowing the Account ID
		kg.AccountPartitionIndex(p.AccountID),

		// These concurrency keys are for fast checking of partition
		// concurrency limits prior to leasing, as an optimization.
		p.acctConcurrencyKey(kg),
		p.fnConcurrencyKey(kg),

		p.acctInProgressLeasesKey(kg, q.capacityManager),
		p.fnInProgressLeasesKey(kg, q.capacityManager),
	}

	args, err := StrSlice([]any{
		p.Queue(),
		leaseID.String(),
		now.UnixMilli(),
		leaseExpires.Unix(),
		accountLimit,
		functionLimit,
		now.Add(PartitionConcurrencyLimitRequeueExtension).Unix(),
		p.AccountID.String(),
		disableLeaseChecksVal,
	})
	if err != nil {
		return nil, 0, err
	}

	result, err := scripts["queue/partitionLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "partitionLease"),
		shard.RedisClient.unshardedRc,
		keys,
		args,
	).AsIntSlice()
	if err != nil {
		return nil, 0, fmt.Errorf("error leasing partition: %w", err)
	}
	if len(result) == 0 {
		return nil, 0, fmt.Errorf("unknown partition lease result: %v", result)
	}

	q.log.Trace("leased partition",
		"partition", p.Queue(),
		"lease_id", leaseID.String(),
		"status", result[0],
		"expires", leaseExpires.Format(time.StampMilli),
	)

	switch result[0] {
	case -1:
		return nil, 0, ErrAccountConcurrencyLimit
	case -2:
		return nil, 0, ErrPartitionConcurrencyLimit
	case -3:
		return nil, 0, ErrPartitionNotFound
	case -4:
		return nil, 0, ErrPartitionAlreadyLeased
	default:
		limit := functionLimit
		if len(result) == 2 {
			limit = int(result[1])
		}

		// Update the partition's last indicator.
		if result[0] > p.Last {
			p.Last = result[0]
		}

		// result is the available concurrency within this partition
		return &leaseID, limit, nil
	}
}

// GlobalPartitionPeek returns up to PartitionSelectionMax partition items from the queue. This
// returns the indexes of partitions.
//
// If sequential is set to true this returns partitions in order from earliest to latest
// available lease times. Otherwise, this shuffles all partitions and picks partitions
// randomly, with higher priority partitions more likely to be selected.  This reduces
// lease contention amongst multiple shared-nothing workers.
func (q *queue) PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error) {
	return q.partitionPeek(ctx, q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), sequential, until, limit, nil)
}

func (q *queue) partitionSize(ctx context.Context, partitionKey string, until time.Time) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "partitionSize"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return 0, fmt.Errorf("unsupported queue shard kind for partitionSize: %s", q.primaryQueueShard.Kind)
	}

	cmd := q.primaryQueueShard.RedisClient.Client().B().Zcount().Key(partitionKey).Min("-inf").Max(strconv.Itoa(int(until.Unix()))).Build()
	return q.primaryQueueShard.RedisClient.Client().Do(ctx, cmd).AsInt64()
}

// TotalSystemQueueDepth calculates and returns the aggregate queue depth across all                           │ │
// partitions in the system. This method provides a comprehensive view of system-wide                          │ │
// queue backlog by collecting size information from every partition queue.
// The method uses the instrumentation iterator to efficiently gather partition data.
func (q *queue) TotalSystemQueueDepth(ctx context.Context) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "TotalSystemQueueDepth"), redis_telemetry.ScopeQueue)

	_, queueItemCount, err := q.QueueIterator(ctx, QueueIteratorOpts{})
	if err != nil {
		return 0, fmt.Errorf("failed to instrument queue: %w", err)
	}

	return queueItemCount, nil
}

// cleanupNilPartitionInAccount is invoked when we peek a missing partition in the account partitions pointer zset.
// This happens when old executors process default function partitions that were enqueued on a new new-runs instance,
// which, in addition to the global partition pointer, enqueued the partition in the account partitions queue of queues.
// This ensures we gracefully handle inconsistencies created by the backwards compatible (keep using global partitions pointer _and_ account partitions) key queues implementation.
func (q *queue) cleanupNilPartitionInAccount(ctx context.Context, accountId uuid.UUID, partitionKey string) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "cleanupNilPartitionInAccount"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for cleanupNilPartitionInAccount: %s", q.primaryQueueShard.Kind)
	}

	// Log because this should only happen as long as we run old code
	q.log.Warn("removing account partitions pointer to missing partition",
		"partition", partitionKey,
		"account_id", accountId.String(),
	)

	cmd := q.primaryQueueShard.RedisClient.Client().B().Zrem().Key(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(accountId)).Member(partitionKey).Build()
	if err := q.primaryQueueShard.RedisClient.Client().Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to remove nil partition from account partitions pointer queue: %w", err)
	}

	// Atomically check whether account partitions is empty and remove from global accounts ZSET
	err := q.cleanupEmptyAccount(ctx, accountId)
	if err != nil {
		return fmt.Errorf("failed to check for and clean up empty account: %w", err)
	}

	return nil
}

// cleanupEmptyAccount is invoked when we peek an account without any partitions in the account pointer zset.
// This happens when old executors process default function partitions and .
// This ensures we gracefully handle inconsistencies created by the backwards compatible (keep using global partitions pointer _and_ account partitions) key queues implementation.
func (q *queue) cleanupEmptyAccount(ctx context.Context, accountId uuid.UUID) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "cleanupEmptyAccount"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for cleanupEmptyAccount: %s", q.primaryQueueShard.Kind)
	}

	if accountId == uuid.Nil {
		q.log.Warn("attempted to clean up empty account pointer with nil account ID")
		return nil
	}

	status, err := scripts["queue/cleanupEmptyAccount"].Exec(
		redis_telemetry.WithScriptName(ctx, "cleanupEmptyAccount"),
		q.primaryQueueShard.RedisClient.Client(),
		[]string{
			q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(),
			q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(accountId),
		},
		[]string{
			accountId.String(),
		},
	).ToInt64()
	if err != nil {
		return fmt.Errorf("failed to check for empty account: %w", err)
	}

	if status == 1 {
		// Log because this should only happen as long as we run old code
		q.log.Warn("removed empty account pointer", "account_id", accountId.String())
	}

	return nil
}

// partitionPeek returns pending queue partitions within the global partition pointer _or_ account partition pointer ZSET.
func (q *queue) partitionPeek(ctx context.Context, partitionKey string, sequential bool, until time.Time, limit int64, accountId *uuid.UUID) ([]*QueuePartition, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "partitionPeek"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for partitionPeek: %s", q.primaryQueueShard.Kind)
	}

	shard := q.primaryQueueShard
	client := shard.RedisClient.Client()
	kg := shard.RedisClient.kg

	if limit > PartitionPeekMax {
		return nil, ErrPartitionPeekMaxExceedsLimits
	}
	if limit <= 0 {
		limit = PartitionPeekMax
	}

	// TODO(tony): If this is an allowlist, only peek the given partitions.  Use ZMSCORE
	// to fetch the scores for all allowed partitions, then filter where score <= until.
	// Call an HMGET to get the partitions.
	ms := until.UnixMilli()

	isSequential := 0
	if sequential {
		isSequential = 1
	}

	args, err := StrSlice([]any{
		ms,
		limit,
		isSequential,
	})
	if err != nil {
		return nil, err
	}

	peekRet, err := scripts["queue/partitionPeek"].Exec(
		redis_telemetry.WithScriptName(ctx, "partitionPeek"),
		client,
		[]string{
			partitionKey,
			kg.PartitionItem(),
		},
		args,
	).ToAny()
	// NOTE: We use ToAny to force return a []any, allowing us to update the slice value with
	// a JSON-decoded item without allocations
	if err != nil {
		return nil, fmt.Errorf("error peeking partition items: %w", err)
	}
	returnedSet, ok := peekRet.([]any)
	if !ok {
		return nil, fmt.Errorf("unknown return type from partitionPeek: %T", peekRet)
	}

	var potentiallyMissingPartitions, allPartitionIds []any
	if len(returnedSet) == 3 {
		potentiallyMissingPartitions, ok = returnedSet[1].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected second item in set returned from partitionPeek: %T", peekRet)
		}

		allPartitionIds, ok = returnedSet[2].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected third item in set returned from partitionPeek: %T", peekRet)
		}
	} else if len(returnedSet) != 0 {
		return nil, fmt.Errorf("expected zero or three items in set returned by partitionPeek: %v", returnedSet)
	}

	encoded := make([]any, 0)
	missingPartitions := make([]string, 0)
	if len(potentiallyMissingPartitions) > 0 {
		for idx, partitionId := range allPartitionIds {
			if potentiallyMissingPartitions[idx] == nil {
				if partitionId == nil {
					return nil, fmt.Errorf("encountered nil partition key in pointer queue %q", partitionKey)
				}

				str, ok := partitionId.(string)
				if !ok {
					return nil, fmt.Errorf("encountered non-string partition key in pointer queue %q", partitionKey)
				}

				missingPartitions = append(missingPartitions, str)
			} else {
				encoded = append(encoded, potentiallyMissingPartitions[idx])
			}
		}
	}

	weights := []float64{}
	items := make([]*QueuePartition, len(encoded))
	fnIDs := make([]uuid.UUID, 0, len(encoded))
	fnIDsMu := sync.Mutex{}

	// Use parallel decoding as per Peek
	partitions, err := util.ParallelDecode(encoded, func(val any, _ int) (*QueuePartition, bool, error) {
		if val == nil {
			q.log.Error("encountered nil partition item in pointer queue",
				"encoded", encoded,
				"missing", missingPartitions,
				"key", partitionKey,
			)
			return nil, false, fmt.Errorf("encountered nil partition item in pointer queue %q", partitionKey)
		}

		str, ok := val.(string)
		if !ok {
			return nil, false, fmt.Errorf("unknown type in partition peek: %T", val)
		}

		item := &QueuePartition{}

		if err := json.Unmarshal(unsafe.Slice(unsafe.StringData(str), len(str)), item); err != nil {
			return nil, false, fmt.Errorf("error reading partition item: %w", err)
		}
		// Track the fn ID for partitions seen.  This allows us to do fast lookups of paused functions
		// to prevent peeking/working on these items as an optimization.
		if item.FunctionID != nil {
			fnIDsMu.Lock()
			fnIDs = append(fnIDs, *item.FunctionID)
			fnIDsMu.Unlock()
		}
		return item, false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error decoding partitions: %w", err)
	}

	if len(missingPartitions) > 0 {
		if accountId == nil {
			return nil, fmt.Errorf("encountered missing partitions in partition pointer queue %q", partitionKey)
		}

		eg := errgroup.Group{}
		for _, partitionId := range missingPartitions {
			id := partitionId
			eg.Go(func() error {
				return q.cleanupNilPartitionInAccount(ctx, *accountId, id)
			})
		}

		if err := eg.Wait(); err != nil {
			return nil, fmt.Errorf("error cleaning up nil partitions in account pointer queue: %w", err)
		}
	}

	migrateLocks := make(map[uuid.UUID]time.Time)
	migrateLocksMu := &sync.Mutex{}

	// mget all migrating states
	if len(fnIDs) > 0 {
		migrateLockKeys := make([]string, len(fnIDs))
		for i, fnID := range fnIDs {
			key := kg.QueueMigrationLock(fnID)
			migrateLockKeys[i] = key
		}

		vals, err := client.Do(ctx, client.B().Mget().Key(migrateLockKeys...).Build()).ToAny()
		if err == nil {
			// If this is an error, just ignore the error and continue.  The executor should gracefully handle
			// accidental attempts at paused functions, as we cannot do this optimization for account or env-level
			// partitions.
			vals, ok := vals.([]any)
			if !ok {
				return nil, fmt.Errorf("unknown return type from mget fnMeta: %T", vals)
			}

			_, _ = util.ParallelDecode(vals, func(encoded any, idx int) (any, bool, error) {
				str, ok := encoded.(string)
				if !ok {
					// the lock did not exist, no need to return anything
					return nil, true, nil
				}

				lockedUntil, err := ulid.Parse(str)
				if err != nil {
					return nil, false, fmt.Errorf("could not parse lock ULID: %w", err)
				}

				migrateLocksMu.Lock()
				fnID := fnIDs[idx]
				migrateLocks[fnID] = lockedUntil.Timestamp()
				migrateLocksMu.Unlock()

				return nil, true, nil
			})
		}
	}

	ignored := 0
	for n, item := range partitions {
		// NOTE: Nil partitions were already reported above. If we got to this point, they're
		// in the account partition pointer and should simply be skipped.
		// This happens when rolling back from a newer deployment with account-queue
		// support to the previous version.
		if item == nil {
			ignored++
			continue
		}

		if item.FunctionID != nil {
			// Check paused status from database with a timeout
			// PartitionPausedGetter does not return errors and simply returns a zero value of
			// info.Paused = false when it encounters an error.
			dbCtx, dbCtxCancel := context.WithTimeout(ctx, dbReadTimeout)
			info := q.partitionPausedGetter(dbCtx, *item.FunctionID)

			if dbCtx.Err() == context.DeadlineExceeded {
				metrics.IncrQueueDatabaseContextTimeoutCounter(ctx, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"operation": "partition_paused_getter",
					},
				})
			}

			dbCtxCancel()

			if info.Paused {
				// Only push back partition if the partition is marked as paused in the database.
				// If the in-memory cache is stale, we don't want to accidentally push back the partition
				// in case the function was unpaused in the last 60s.
				if !info.Stale {
					// Function is pulled up when it is unpaused, so we can push it back for a long time (see SetFunctionPaused)
					err := q.PartitionRequeue(ctx, shard, item, q.clock.Now().Truncate(time.Second).Add(PartitionPausedRequeueExtension), true)
					if err != nil && !errors.Is(err, ErrPartitionGarbageCollected) {
						q.log.Error("failed to push back paused partition", "error", err, "partition", item)
					} else {
						q.log.Trace("pushed back paused partition", "partition", item.Queue())
					}
				}

				ignored++
				continue
			}

			if lockedUntil, ok := migrateLocks[*item.FunctionID]; ok {
				err := q.PartitionRequeue(ctx, shard, item, lockedUntil, true)
				if err != nil && !errors.Is(err, ErrPartitionGarbageCollected) {
					q.log.Error("failed to push back migrating partition", "error", err, "partition", item)
				} else {
					q.log.Trace("pushed back migrating partition", "partition", item.Queue())
				}
				// skip this since the executor is not responsible for migrating queues
				ignored++
				continue
			}
		}

		// NOTE: The queue does two conflicting things:  we peek ahead of now() to fetch partitions
		// shortly available, and we also requeue partitions if there are concurrency conflicts.
		//
		// We want to ignore any partitions requeued because of conflicts, as this will cause needless
		// churn every peek MS.
		if item.ForceAtMS > ms {
			ignored++
			continue
		}

		// If we have an allowlist, only accept this partition if its in the allowlist.
		if len(q.allowQueues) > 0 && !checkList(item.Queue(), q.allowQueueMap, q.allowQueuePrefixes) {
			// This is not in the allowlist specified, so do not allow this partition to be used.
			ignored++
			continue
		}

		// Ignore any denied queues if they're explicitly in the denylist.  Because
		// we allocate the len(encoded) amount, we also want to track the number of
		// ignored queues to use the correct index when setting our items;  this ensures
		// that we don't access items with an index and get nil pointers.
		if len(q.denyQueues) > 0 && checkList(item.Queue(), q.denyQueueMap, q.denyQueuePrefixes) {
			// This is in the denylist explicitly set, so continue
			ignored++
			continue
		}

		items[n-ignored] = item
		partPriority := q.ppf(ctx, *item)
		weights = append(weights, float64(10-partPriority))
	}

	// Remove any ignored items from the slice.
	items = items[0 : len(items)-ignored]

	// Some scanners run sequentially, ensuring we always work on the functions with
	// the oldest run at times in order, no matter the priority.
	if sequential {
		n := int(math.Min(float64(len(items)), float64(PartitionSelectionMax)))
		return items[0:n], nil
	}

	// We want to weighted shuffle the resulting array random.  This means that many
	// shared nothing scanners can query for outstanding partitions and receive a
	// randomized order favouring higher-priority queue items.  This reduces the chances
	// of contention when leasing.
	w := sampleuv.NewWeighted(weights, rnd)
	result := make([]*QueuePartition, len(items))
	for n := range result {
		idx, ok := w.Take()
		if !ok {
			return nil, util.ErrWeightedSampleRead
		}
		result[n] = items[idx]
	}

	return result, nil
}

func (q *queue) accountPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]uuid.UUID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "accountPeek"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for accountPeek: %s", q.primaryQueueShard.Kind)
	}

	if limit > AccountPeekMax {
		return nil, ErrAccountPeekMaxExceedsLimits
	}
	if limit <= 0 {
		limit = AccountPeekMax
	}

	ms := until.UnixMilli()

	isSequential := 0
	if sequential {
		isSequential = 1
	}

	args, err := StrSlice([]any{
		ms,
		limit,
		isSequential,
	})
	if err != nil {
		return nil, err
	}

	peekRet, err := scripts["queue/accountPeek"].Exec(
		redis_telemetry.WithScriptName(ctx, "accountPeek"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		[]string{
			q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(),
		},
		args,
	).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error peeking accounts: %w", err)
	}

	items := make([]uuid.UUID, len(peekRet))

	for i, s := range peekRet {
		parsed, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("could not parse account id from global account queue: %w", err)
		}

		items[i] = parsed
	}

	weights := make([]float64, len(items))
	for i := range items {
		accountPriority := q.apf(ctx, items[i])
		weights[i] = float64(10 - accountPriority)
	}

	// Some scanners run sequentially, ensuring we always work on the accounts with
	// the oldest run at times in order, no matter the priority.
	if sequential {
		n := int(math.Min(float64(len(items)), float64(PartitionSelectionMax)))
		return items[0:n], nil
	}

	// We want to weighted shuffle the resulting array random.  This means that many
	// shared nothing scanners can query for outstanding partitions and receive a
	// randomized order favouring higher-priority queue items.  This reduces the chances
	// of contention when leasing.
	w := sampleuv.NewWeighted(weights, rnd)
	result := make([]uuid.UUID, len(items))
	for n := range result {
		idx, ok := w.Take()
		if !ok {
			return nil, util.ErrWeightedSampleRead
		}
		result[n] = items[idx]
	}

	return result, nil
}

func checkList(check string, exact, prefixes map[string]*struct{}) bool {
	for k := range exact {
		if check == k {
			return true
		}
	}
	for k := range prefixes {
		if strings.HasPrefix(check, k) {
			return true
		}
	}
	return false
}

// PartitionRequeue requeues a parition with a new score, ensuring that the partition will be
// read at (or very close to) the given time.
//
// This is used after peeking and passing all queue items onto workers; we then take the next
// unleased available time for the queue item and requeue the partition.
//
// forceAt is used to enforce the given queue time.  This is used when partitions are at a
// concurrency limit;  we don't want to scan the partition next time, so we force the partition
// to be at a specific time instead of taking the earliest available queue item time
func (q *queue) PartitionRequeue(ctx context.Context, shard RedisQueueShard, p *QueuePartition, at time.Time, forceAt bool) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PartitionRequeue"), redis_telemetry.ScopeQueue)

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for PartitionRequeue: %s", shard.Kind)
	}

	kg := shard.RedisClient.kg

	functionId := uuid.Nil
	if p.FunctionID != nil {
		functionId = *p.FunctionID
	}

	keys := []string{
		kg.PartitionItem(),
		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		// NOTE: Old partitions will _not_ have an account ID until the next enqueue on the new code.
		// Until this, we may not use account queues at all, as we cannot properly clean up
		// here without knowing the Account ID
		kg.AccountPartitionIndex(p.AccountID),

		// NOTE: Partition metadata was replaced with function metadata and is being phased out
		// We clean up all remaining partition metadata on completely empty partitions here
		// and are adding function metadata on enqueue to migrate to the new system
		kg.PartitionMeta(p.Queue()),
		kg.FnMetadata(functionId),

		p.zsetKey(kg), // Partition ZSET itself
		p.concurrencyKey(kg),
		kg.QueueItem(),

		// Backlogs in shadow partition
		kg.ShadowPartitionSet(p.ID),
	}
	force := 0
	if forceAt {
		force = 1
	}
	args, err := StrSlice([]any{
		p.Queue(),
		at.UnixMilli(),
		force,
		p.AccountID.String(),
	})
	if err != nil {
		return err
	}
	status, err := scripts["queue/partitionRequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "partitionRequeue"),
		shard.RedisClient.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error requeueing partition: %w", err)
	}

	leaseID := "n/a"
	if p.LeaseID != nil {
		leaseID = p.LeaseID.String()
	}

	q.log.Trace("requeued partition",
		"partition", p.Queue(),
		"status", status,
		"lease_id", leaseID,
		"at", at.Format(time.StampMilli),
	)

	switch status {
	case 0:
		return nil
	case 1:
		return ErrPartitionNotFound
	case 2, 3:
		return ErrPartitionGarbageCollected
	default:
		return fmt.Errorf("unknown response requeueing item: %d", status)
	}
}

// PartitionReprioritize reprioritizes a workflow's QueueItems within the queue.
func (q *queue) PartitionReprioritize(ctx context.Context, queueName string, priority uint) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PartitionReprioritize"), redis_telemetry.ScopeQueue)

	if priority > PriorityMin {
		return ErrPriorityTooLow
	}
	if priority < PriorityMax {
		return ErrPriorityTooHigh
	}

	args, err := StrSlice([]any{
		queueName,
		priority,
	})
	if err != nil {
		return err
	}

	keys := []string{q.primaryQueueShard.RedisClient.kg.PartitionItem()}
	status, err := scripts["queue/partitionReprioritize"].Exec(
		redis_telemetry.WithScriptName(ctx, "partitionReprioritize"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error enqueueing item: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return ErrPartitionNotFound
	default:
		return fmt.Errorf("unknown response reprioritizing partition: %d", status)
	}
}

func (q *queue) InProgress(ctx context.Context, prefix string, concurrencyKey string) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "InProgress"), redis_telemetry.ScopeQueue)

	s := q.clock.Now().UnixMilli()
	cmd := q.primaryQueueShard.RedisClient.unshardedRc.B().Zcount().
		Key(q.primaryQueueShard.RedisClient.kg.Concurrency(prefix, concurrencyKey)).
		Min(fmt.Sprintf("%d", s)).
		Max("+inf").
		Build()
	return q.primaryQueueShard.RedisClient.unshardedRc.Do(ctx, cmd).AsInt64()
}

func (q *queue) Instrument(ctx context.Context) error {
	_, _, err := q.QueueIterator(ctx, QueueIteratorOpts{
		OnPartitionProcessed: func(ctx context.Context, partitionKey, queueKey string, itemCount int64, queueShard RedisQueueShard) {
			// Handle individual partition instrumentation
			// NOTE: tmp workaround for cardinality issues
			// ideally we want to instrument everything, but until there's a better way to do this, we primarily care only
			// about large size partitions
			if itemCount > 10_000 {
				metrics.GaugePartitionSize(ctx, itemCount, metrics.GaugeOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						// NOTE: potentially high cardinality but this gives better clarify of stuff
						// this is potentially useless for key queues
						"partition":   partitionKey,
						"queue_shard": queueShard.Name,
					},
				})
			}
		},
		OnIterationComplete: func(ctx context.Context, totalPartitions, totalQueueItems int64, queueShard RedisQueueShard) {
			// Handle the final metrics reporting
			metrics.GaugeGlobalPartitionSize(ctx, totalPartitions, metrics.GaugeOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": queueShard.Name,
				},
			})
		},
	})
	if err != nil {
		return fmt.Errorf("failed to iterate queue partitions during instrumentation: %w", err)
	}

	return nil
}

// isKeyPreviousConcurrencyPointerItem checks whether given string conforms to fully-qualified key as concurrency index item
func isKeyConcurrencyPointerItem(partition string) bool {
	return strings.HasPrefix(partition, "{")
}

// ConfigLease allows a worker to lease config keys for sequential or scavenger processing.
// Leasing this key works similar to leasing partitions or queue items:
//
//   - If the key isn't leased, a new lease is accepted.
//   - If the lease is expired, a new lease is accepted.
//   - If the key is leased, you must pass in the existing lease ID to renew the lease.  Mismatches do not
//     grant a lease.
//
// This returns the new lease ID on success.
//
// If the sequential key is leased, this allows a worker to peek partitions sequentially.
func (q *queue) ConfigLease(ctx context.Context, key string, duration time.Duration, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ConfigLease: %s", q.primaryQueueShard.Kind)
	}

	if duration > ConfigLeaseMax {
		return nil, ErrConfigLeaseExceedsLimits
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ConfigLease"), redis_telemetry.ScopeQueue)

	now := q.clock.Now()
	newLeaseID, err := ulid.New(ulid.Timestamp(now.Add(duration)), rnd)
	if err != nil {
		return nil, err
	}

	var existing string
	if len(existingLeaseID) > 0 && existingLeaseID[0] != nil {
		existing = existingLeaseID[0].String()
	}

	args, err := StrSlice([]any{
		now.UnixMilli(),
		newLeaseID.String(),
		existing,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/configLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "configLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		[]string{key},
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error claiming config lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case 1:
		return nil, ErrConfigAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response claiming config lease: %d", status)
	}
}

// peekEWMA returns the calculated EWMA value from the list
// nolint:unused // this code remains to be enabled on demand
func (q *queue) peekEWMA(ctx context.Context, fnID uuid.UUID) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "peekEWMA"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return 0, fmt.Errorf("unsupported queue shard kind for peekEWMA: %s", q.primaryQueueShard.Kind)
	}

	// retrieves the list from redis
	cmd := q.primaryQueueShard.RedisClient.Client().B().Lrange().Key(q.primaryQueueShard.RedisClient.KeyGenerator().ConcurrencyFnEWMA(fnID)).Start(0).Stop(-1).Build()
	strlist, err := q.primaryQueueShard.RedisClient.Client().Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return 0, fmt.Errorf("error reading function concurrency EWMA values: %w", err)
	}

	// return early
	if len(strlist) == 0 {
		return 0, nil
	}

	hasNonZero := false
	vals := make([]float64, len(strlist))
	for i, s := range strlist {
		v, _ := strconv.ParseFloat(s, 64)
		vals[i] = v
		if v > 0 {
			hasNonZero = true
		}
	}

	if !hasNonZero {
		// short-circuit.
		return 0, nil
	}

	// create a simple EWMA, add all the numbers in it and get the final value
	// NOTE: we don't need variable since we don't want to maintain this in memory
	mavg := ewma.NewMovingAverage()
	for _, v := range vals {
		mavg.Add(v)
	}

	// round up to the nearest integer
	return int64(math.Round(mavg.Value())), nil
}

// setPeekEWMA add the new value to the existing list.
// if the length of the list exceeds the predetermined size, pop out the first item
func (q *queue) setPeekEWMA(ctx context.Context, fnID *uuid.UUID, val int64) error {
	if fnID == nil {
		return nil
	}

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for setPeekEWMA: %s", q.primaryQueueShard.Kind)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "setPeekEWMA"), redis_telemetry.ScopeQueue)

	listSize := q.peekEWMALen
	if listSize == 0 {
		listSize = QueuePeekEWMALen
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ConcurrencyFnEWMA(*fnID),
	}
	args, err := StrSlice([]any{
		val,
		listSize,
	})
	if err != nil {
		return err
	}

	_, err = scripts["queue/setPeekEWMA"].Exec(
		redis_telemetry.WithScriptName(ctx, "setPeekEWMA"),
		q.primaryQueueShard.RedisClient.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error updating function concurrency EWMA: %w", err)
	}

	return nil
}

// addContinue adds a continuation for the given partition.  This hints that the queue should
// peek and process this partition on the next loop, allowing us to hint that a partition
// should be processed when a step finishes (to decrease inter-step latency on non-connect
// workloads).
func (q *queue) addContinue(ctx context.Context, p *QueuePartition, ctr uint) {
	if !q.runMode.Continuations {
		// continuations are not enabled.
		return
	}

	if ctr >= q.continuationLimit {
		q.removeContinue(ctx, p, true)
		return
	}

	q.continuesLock.Lock()
	defer q.continuesLock.Unlock()

	// If this is the first continuation, check if we're on a cooldown, or if we're
	// beyond capacity.
	if ctr == 1 {
		if len(q.continues) > consts.QueueContinuationMaxPartitions {
			metrics.IncrQueueContinuationMaxCapcityCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
			return
		}
		if t, ok := q.continueCooldown[p.Queue()]; ok && t.After(time.Now()) {
			metrics.IncrQueueContinuationCooldownCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
			return
		}

		// Remove the continuation cooldown.
		delete(q.continueCooldown, p.Queue())
	}

	c, ok := q.continues[p.Queue()]
	if !ok || c.count < ctr {
		// Update the continue count if it doesn't exist, or the current counter
		// is higher.  This ensures that we always have the highest continuation
		// count stored for queue processing.
		q.continues[p.Queue()] = continuation{partition: p, count: ctr}
		metrics.IncrQueueContinuationAddedCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
	}
}

func (q *queue) removeContinue(ctx context.Context, p *QueuePartition, cooldown bool) {
	if !q.runMode.Continuations {
		// continuations are not enabled.
		return
	}

	// This is over the limit for conntinuing the partition, so force it to be
	// removed in every case.
	q.continuesLock.Lock()
	defer q.continuesLock.Unlock()

	metrics.IncrQueueContinuationRemovedCounter(ctx, metrics.CounterOpt{PkgName: pkgName})

	delete(q.continues, p.Queue())

	if cooldown {
		// Add a cooldown, preventing this partition from being added as a continuation
		// for a given period of time.
		//
		// Note that this isn't shared across replicas;  cooldowns
		// only exist in the current replica.
		q.continueCooldown[p.Queue()] = time.Now().Add(
			consts.QueueContinuationCooldownPeriod,
		)
	}
}

func newLeaseDenyList() *leaseDenies {
	return &leaseDenies{
		lock:        &sync.RWMutex{},
		concurrency: map[string]struct{}{},
		throttle:    map[string]struct{}{},
	}
}
