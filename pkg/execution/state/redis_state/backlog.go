package redis_state

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/redis/rueidis"
	"go.opentelemetry.io/otel/attribute"
)

// readyQueueKey returns the ZSET key to the ready queue
func shadowPartitionReadyQueueKey(sp osqueue.QueueShadowPartition, kg QueueKeyGenerator) string {
	return kg.PartitionQueueSet(enums.PartitionTypeDefault, sp.PartitionID, "")
}

// inProgressKey returns the key storing the in progress set for the shadow partition
func shadowPartitionInProgressKey(sp osqueue.QueueShadowPartition, kg QueueKeyGenerator) string {
	return kg.Concurrency("p", sp.PartitionID)
}

// activeKey returns the key storing the active set for the shadow partition
func shadowPartitionActiveKey(sp osqueue.QueueShadowPartition, kg QueueKeyGenerator) string {
	return kg.ActiveSet("p", sp.PartitionID)
}

// accountInProgressKey returns the key storing the in progress set for the shadow partition's account
func shadowPartitionAccountInProgressKey(sp osqueue.QueueShadowPartition, kg QueueKeyGenerator) string {
	// Do not track account concurrency for system queues
	if sp.SystemQueueName != nil {
		return kg.Concurrency("", "")
	}

	// This should never be unset
	if sp.AccountID == nil {
		return kg.Concurrency("account", "")
	}

	return kg.Concurrency("account", sp.AccountID.String())
}

// accountActiveKey returns the key storing the active set for the shadow partition's account
func shadowPartitionAccountActiveKey(sp osqueue.QueueShadowPartition, kg QueueKeyGenerator) string {
	// Do not track account concurrency for system queues
	if sp.SystemQueueName != nil {
		return kg.ActiveSet("", "")
	}

	// This should never be unset
	if sp.AccountID == nil {
		return kg.ActiveSet("account", "")
	}

	return kg.ActiveSet("account", sp.AccountID.String())
}

func shadowPartitionAccountActiveRunKey(sp osqueue.QueueShadowPartition, kg QueueKeyGenerator) string {
	// Do not track account run concurrency for system queues
	if sp.SystemQueueName != nil {
		return kg.ActiveRunsSet("", "")
	}

	// This should never be unset
	if sp.AccountID == nil {
		return kg.ActiveRunsSet("account", "")
	}

	return kg.ActiveRunsSet("account", sp.AccountID.String())
}

func shadowPartitionActiveRunKey(sp osqueue.QueueShadowPartition, kg QueueKeyGenerator) string {
	return kg.ActiveRunsSet("p", sp.PartitionID)
}

// customKeyInProgress returns the key to the "in progress" ZSET
func backlogCustomKeyInProgress(b osqueue.QueueBacklog, kg QueueKeyGenerator, n int) string {
	if n < 0 || n > len(b.ConcurrencyKeys) {
		return kg.Concurrency("", "")
	}

	key := b.ConcurrencyKeys[n-1]
	return backlogConcurrencyKey(key, kg)
}

func backlogConcurrencyKey(bck osqueue.BacklogConcurrencyKey, kg QueueKeyGenerator) string {
	// Concurrency accounting keys are made up of three parts:
	// - The scope (account, environment, function) to apply the concurrency limit on
	// - The entity (account ID, envID, or function ID) based on the scope
	// - The dynamic key value (hashed evaluated expression)
	return kg.Concurrency("custom", bck.CanonicalKeyID)
}

// customKeyActive returns the key to the active set for the given custom concurrency key
func backlogCustomKeyActive(b osqueue.QueueBacklog, kg QueueKeyGenerator, n int) string {
	if n < 0 || n > len(b.ConcurrencyKeys) {
		return kg.ActiveSet("", "")
	}

	key := b.ConcurrencyKeys[n-1]
	return backlogConcurrencyKeyActiveKey(key, kg)
}

// customKeyActiveRuns returns the key to the active runs counter for the given custom concurrency key
func backlogCustomKeyActiveRuns(b osqueue.QueueBacklog, kg QueueKeyGenerator, n int) string {
	if n < 0 || n > len(b.ConcurrencyKeys) {
		return kg.ActiveRunsSet("", "")
	}

	key := b.ConcurrencyKeys[n-1]
	return backlogConcurrencyKeyActiveRunsKey(key, kg)
}

func backlogInProgressLeasesCustomKey(b osqueue.QueueBacklog, cm constraintapi.RolloutKeyGenerator, kg QueueKeyGenerator, accountID *uuid.UUID, n int) string {
	if cm == nil {
		return kg.Concurrency("", "")
	}

	if accountID == nil {
		return kg.Concurrency("", "")
	}

	if n < 0 || n > len(b.ConcurrencyKeys) {
		return kg.Concurrency("", "")
	}

	key := b.ConcurrencyKeys[n-1]
	return backlogConcurrencyKeyInProgressLeasesKey(key, cm, *accountID)
}

func backlogConcurrencyKeyActiveKey(bck osqueue.BacklogConcurrencyKey, kg QueueKeyGenerator) string {
	// Concurrency accounting keys are made up of three parts:
	// - The scope (account, environment, function) to apply the concurrency limit on
	// - The entity (account ID, envID, or function ID) based on the scope
	// - The dynamic key value (hashed evaluated expression)
	return kg.ActiveSet("custom", bck.CanonicalKeyID)
}

func backlogConcurrencyKeyActiveRunsKey(bck osqueue.BacklogConcurrencyKey, kg QueueKeyGenerator) string {
	return kg.ActiveRunsSet("custom", bck.CanonicalKeyID)
}

func backlogConcurrencyKeyInProgressLeasesKey(bck osqueue.BacklogConcurrencyKey, cm constraintapi.RolloutKeyGenerator, accountID uuid.UUID) string {
	return cm.KeyInProgressLeasesCustom(accountID, bck.Scope, bck.EntityID, bck.HashedKeyExpression, bck.HashedValue)
}

// activeKey returns backlog compound active key
func backlogActiveKey(b osqueue.QueueBacklog, kg QueueKeyGenerator) string {
	return kg.ActiveSet("compound", b.BacklogID)
}

func (q *queue) BacklogRefill(
	ctx context.Context,
	b *osqueue.QueueBacklog,
	sp *osqueue.QueueShadowPartition,
	refillUntil time.Time,
	refillItems []string,
	latestConstraints osqueue.PartitionConstraintConfig,
	options ...osqueue.BacklogRefillOptionFn,
) (*osqueue.BacklogRefillResult, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "BacklogRefill"), redis_telemetry.ScopeQueue)

	o := &osqueue.BacklogRefillOptions{}
	for _, opt := range options {
		opt(o)
	}

	kg := q.RedisClient.kg

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	partitionID := sp.Identifier()
	ctx, span := q.ConditionalTracer.NewSpan(ctx, "queue.BacklogRefill", partitionID.AccountID, partitionID.EnvID)
	defer span.End()
	span.SetAttributes(attribute.String("partition_id", sp.PartitionID))
	span.SetAttributes(attribute.String("backlog_id", b.BacklogID))

	nowMS := q.Clock.Now().UnixMilli()

	var (
		keyThrottleState                             string
		throttleLimit, throttleBurst, throttlePeriod int
	)
	if latestConstraints.Throttle != nil && b.Throttle != nil {
		// NOTE: The Throttle state key must be generated to match the Redis key used in the Lease and Constraint API implementation
		keyThrottleState = kg.ThrottleKey(&osqueue.Throttle{Key: b.Throttle.ThrottleKey})
		throttleLimit = latestConstraints.Throttle.Limit
		throttleBurst = latestConstraints.Throttle.Burst
		throttlePeriod = latestConstraints.Throttle.Period
	}

	keys := []string{
		kg.ShadowPartitionMeta(),
		kg.BacklogMeta(),

		kg.BacklogSet(b.BacklogID),
		kg.ShadowPartitionSet(sp.PartitionID),
		kg.GlobalShadowPartitionSet(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(accountID),

		shadowPartitionReadyQueueKey(*sp, kg),
		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		kg.AccountPartitionIndex(accountID),

		kg.QueueItem(),

		// Constraint-related accounting keys
		shadowPartitionAccountActiveKey(*sp, kg), // account active
		shadowPartitionActiveKey(*sp, kg),        // partition active
		backlogCustomKeyActive(*b, kg, 1),        // custom key 1
		backlogCustomKeyActive(*b, kg, 2),        // custom key 2
		backlogActiveKey(*b, kg),                 // compound key (active for this backlog)

		// Active run sets
		// kg.RunActiveSet(i.Data.Identifier.RunID), -> dynamically constructed in script for each item
		shadowPartitionAccountActiveRunKey(*sp, kg), // Set for active runs in account
		shadowPartitionActiveRunKey(*sp, kg),        // Set for active runs in partition
		backlogCustomKeyActiveRuns(*b, kg, 1),       // Set for active runs with custom concurrency key 1
		backlogCustomKeyActiveRuns(*b, kg, 2),       // Set for active runs with custom concurrency key 2

		kg.BacklogActiveCheckSet(),
		kg.BacklogActiveCheckCooldown(b.BacklogID),

		kg.PartitionNormalizeSet(sp.PartitionID),

		// Constraint API rollout
		q.keyConstraintCheckIdempotency(sp.AccountID, o.ConstraintCheckIdempotencyKey),
	}

	// Don't check constraints if
	// - key queues have been disabled for this function (refill as quickly as possible)
	// - capacity leases were successfully acquired
	checkConstraints := sp.KeyQueuesEnabled(ctx, &q.QueueOptions)
	if o.DisableConstraintChecks {
		checkConstraints = false
	}

	checkConstraintsVal := "1"
	if !checkConstraints {
		checkConstraintsVal = "0"
	}

	// Enable conditional spot checking (probability in queue settings + feature flag)
	refillProbability, _ := q.ActiveSpotCheckProbability(ctx, accountID)
	shouldSpotCheckActiveSet := checkConstraints && rand.Intn(100) <= refillProbability

	// Ensure capacityLeaseIDs is never nil to avoid JSON marshaling to "null"
	capacityLeaseIDs := o.CapacityLeases
	if capacityLeaseIDs == nil {
		capacityLeaseIDs = []osqueue.CapacityLease{}
	}

	args, err := StrSlice([]any{
		b.BacklogID,
		sp.PartitionID,
		accountID,
		refillUntil.UnixMilli(),
		refillItems,
		nowMS,

		latestConstraints.Concurrency.AccountConcurrency,
		latestConstraints.Concurrency.FunctionConcurrency,
		latestConstraints.CustomConcurrencyLimit(1),
		latestConstraints.CustomConcurrencyLimit(2),

		keyThrottleState,
		throttleLimit,
		throttleBurst,
		throttlePeriod,

		kg.QueuePrefix(),
		checkConstraintsVal,
		shouldSpotCheckActiveSet,

		capacityLeaseIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("could not serialize args: %w", err)
	}

	res, err := scripts["queue/backlogRefill"].Exec(
		redis_telemetry.WithScriptName(ctx, "backlogRefill"),
		q.RedisClient.unshardedRc,
		keys,
		args,
	).ToAny()
	if err != nil {
		return nil, fmt.Errorf("error refilling backlog: %w", err)
	}

	returnTuple, ok := res.([]any)
	if !ok || len(returnTuple) != 8 {
		return nil, fmt.Errorf("expected return tuple to include 8 items")
	}

	status, ok := returnTuple[0].(int64)
	if !ok {
		return nil, fmt.Errorf("missing status in returned tuple")
	}

	refillCount, ok := returnTuple[1].(int64)
	if !ok {
		return nil, fmt.Errorf("missing refillCount in returned tuple")
	}

	backlogCountUntil, ok := returnTuple[2].(int64)
	if !ok {
		return nil, fmt.Errorf("missing backlogCount in returned tuple")
	}

	backlogCountTotal, ok := returnTuple[3].(int64)
	if !ok {
		return nil, fmt.Errorf("missing backlogCount in returned tuple")
	}

	capacity, ok := returnTuple[4].(int64)
	if !ok {
		return nil, fmt.Errorf("missing capacity in returned tuple")
	}

	refill, ok := returnTuple[5].(int64)
	if !ok {
		return nil, fmt.Errorf("missing refill in returned tuple")
	}

	rawRefilledItemIDs, ok := returnTuple[6].([]any)
	if !ok {
		return nil, fmt.Errorf("missing refilled item IDs in returned tuple")
	}

	refilledItemIDs := make([]string, len(rawRefilledItemIDs))
	for i, d := range rawRefilledItemIDs {
		itemID, ok := d.(string)
		if ok {
			refilledItemIDs[i] = itemID
		}
	}

	var retryAt time.Time
	retryAtMillis, ok := returnTuple[7].(int64)
	if !ok {
		return nil, fmt.Errorf("missing retryAt in returned tuple")
	}

	if retryAtMillis > nowMS {
		retryAt = time.UnixMilli(retryAtMillis)
	}

	refillResult := &osqueue.BacklogRefillResult{
		Refilled:          int(refillCount),
		TotalBacklogCount: int(backlogCountTotal),
		BacklogCountUntil: int(backlogCountUntil),
		Capacity:          int(capacity),
		Refill:            int(refill),
		RefilledItems:     refilledItemIDs,
		RetryAt:           retryAt,
	}

	switch status {
	case 0:
		return refillResult, nil
	case 1:
		refillResult.Constraint = enums.QueueConstraintAccountConcurrency
		return refillResult, nil
	case 2:
		refillResult.Constraint = enums.QueueConstraintFunctionConcurrency
		return refillResult, nil
	case 3:
		refillResult.Constraint = enums.QueueConstraintCustomConcurrencyKey1
		return refillResult, nil
	case 4:
		refillResult.Constraint = enums.QueueConstraintCustomConcurrencyKey2
		return refillResult, nil
	case 5:
		refillResult.Constraint = enums.QueueConstraintThrottle
		return refillResult, nil
	default:
		return nil, fmt.Errorf("unknown status refilling backlog: %v (%T)", status, status)
	}
}

func (q *queue) BacklogRequeue(ctx context.Context, backlog *osqueue.QueueBacklog, sp *osqueue.QueueShadowPartition, requeueAt time.Time) error {
	l := logger.StdlibLogger(ctx)

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "BacklogRequeue"), redis_telemetry.ScopeQueue)

	kg := q.RedisClient.kg

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	partitionID := sp.Identifier()
	ctx, span := q.ConditionalTracer.NewSpan(ctx, "queue.BacklogRequeue", partitionID.AccountID, partitionID.EnvID)
	defer span.End()
	span.SetAttributes(attribute.String("partition_id", sp.PartitionID))
	span.SetAttributes(attribute.String("backlog_id", backlog.BacklogID))

	keys := []string{
		kg.ShadowPartitionMeta(),
		kg.BacklogMeta(),
		kg.ShadowPartitionMeta(),

		kg.GlobalShadowPartitionSet(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(accountID),
		kg.ShadowPartitionSet(sp.PartitionID),
		kg.BacklogSet(backlog.BacklogID),

		kg.PartitionNormalizeSet(sp.PartitionID),
	}
	args, err := StrSlice([]any{
		accountID,
		sp.PartitionID,
		backlog.BacklogID,
		requeueAt.UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/backlogRequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "backlogRequeue"),
		q.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("could not requeue backlog: %w", err)
	}

	l.Trace("requeued backlog",
		"id", backlog.BacklogID,
		"partition", sp.PartitionID,
		"time", requeueAt.Format(time.StampMilli),
		"successive_throttle", backlog.SuccessiveThrottleConstrained,
		"successive_concurrency", backlog.SuccessiveCustomConcurrencyConstrained,
		"status", status,
	)

	switch status {
	case 0, 1:
		return nil
	case -1:
		return osqueue.ErrBacklogNotFound
	default:
		return fmt.Errorf("unknown response requeueing backlog: %v (%T)", status, status)
	}
}

func (q *queue) BacklogPrepareNormalize(ctx context.Context, b *osqueue.QueueBacklog, sp *osqueue.QueueShadowPartition) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "BacklogPrepareNormalize"), redis_telemetry.ScopeQueue)

	kg := q.RedisClient.kg

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	keys := []string{
		kg.BacklogMeta(),
		kg.ShadowPartitionMeta(),

		kg.BacklogSet(b.BacklogID),
		kg.ShadowPartitionSet(sp.PartitionID),
		kg.GlobalShadowPartitionSet(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(accountID),

		kg.GlobalAccountNormalizeSet(),
		kg.AccountNormalizeSet(accountID),
		kg.PartitionNormalizeSet(sp.PartitionID),
	}
	args, err := StrSlice([]any{
		b.BacklogID,
		sp.PartitionID,
		accountID,
		// order normalize by timestamp
		q.Clock.Now().UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/backlogPrepareNormalize"].Exec(
		redis_telemetry.WithScriptName(ctx, "backlogPrepareNormalize"),
		q.RedisClient.unshardedRc,
		keys,
		args,
	).ToInt64()
	if err != nil {
		return fmt.Errorf("error preparing backlog normalization: %w", err)
	}

	switch status {
	case 1:
		return nil
	case -1:
		return osqueue.ErrBacklogGarbageCollected
	default:
		return fmt.Errorf("unknown status preparing backlog normalization: %v (%T)", status, status)
	}
}

// BacklogPeek peeks item from the given backlog.
//
// Pointers to missing items will be removed from the backlog.
func (q *queue) BacklogPeek(ctx context.Context, b *osqueue.QueueBacklog, from time.Time, until time.Time, limit int64, opts ...osqueue.PeekOpt) ([]*osqueue.QueueItem, int, error) {
	l := logger.StdlibLogger(ctx)
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "backlogPeek"), redis_telemetry.ScopeQueue)

	opt := osqueue.PeekOption{}
	for _, apply := range opts {
		apply(&opt)
	}

	if b == nil {
		return nil, 0, fmt.Errorf("expected backlog to be provided")
	}

	if limit > osqueue.AbsoluteQueuePeekMax || limit > q.PeekMax {
		limit = q.PeekMax
	}
	if limit <= 0 {
		limit = q.PeekMin
	}

	var fromTime *time.Time
	if !from.IsZero() {
		fromTime = &from
	}

	l = l.With(
		"method", "backlogPeek",
		"backlog", b,
		"from", from,
		"until", until,
		"limit", limit,
	)

	rc := q.RedisClient
	backlogSet := rc.kg.BacklogSet(b.BacklogID)

	p := peeker[osqueue.QueueItem]{
		q:               q,
		opName:          "backlogPeek",
		keyMetadataHash: rc.kg.QueueItem(),
		max:             q.PeekMax,
		maker: func() *osqueue.QueueItem {
			return &osqueue.QueueItem{}
		},
		handleMissingItems:     CleanupMissingPointers(ctx, backlogSet, rc.Client(), l),
		isMillisecondPrecision: true,
		fromTime:               fromTime,
	}

	res, err := p.peek(ctx, backlogSet, true, until, limit, opts...)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, 0, osqueue.ErrBacklogPeekMaxExceedsLimits
		}
		return nil, 0, fmt.Errorf("error peeking backlog queue items, %w", err)
	}

	return res.Items, res.TotalCount, nil
}

// NOTE: this function only work with key queues
func (q *queue) BacklogsByPartition(ctx context.Context, partitionID string, from time.Time, until time.Time, opts ...osqueue.QueueIterOpt) (iter.Seq[*osqueue.QueueBacklog], error) {
	l := logger.StdlibLogger(ctx)
	opt := osqueue.QueueIterOptions{
		BatchSize:                 1000,
		Interval:                  50 * time.Millisecond,
		EnableMillisecondIncrease: true,
	}
	for _, apply := range opts {
		apply(&opt)
	}

	l = l.With(
		"method", "BacklogsByPartition",
		"partitionID", partitionID,
		"from", from,
		"until", until,
	)

	kg := q.RedisClient.kg

	return func(yield func(*osqueue.QueueBacklog) bool) {
		hashKey := kg.BacklogMeta()
		ptFrom := from

		for {
			var iterated int

			peeker := peeker[osqueue.QueueBacklog]{
				q:                      q,
				max:                    opt.BatchSize,
				opName:                 "backlogsByPartition",
				isMillisecondPrecision: true,
				handleMissingItems: func(pointers []string) error {
					// don't interfere, clean up will happen in normal processing anyways
					return nil
				},
				maker: func() *osqueue.QueueBacklog {
					return &osqueue.QueueBacklog{}
				},
				keyMetadataHash: hashKey,
				fromTime:        &ptFrom,
			}

			isSequential := true
			res, err := peeker.peek(ctx, kg.ShadowPartitionSet(partitionID), isSequential, until, opt.BatchSize)
			if err != nil {
				l.Error("error peeking backlogs for partition", "partition_id", partitionID, "err", err)
				return
			}

			for _, bl := range res.Items {
				if bl == nil {
					continue
				}

				if !yield(bl) {
					return
				}

				iterated++
			}

			ptFrom = time.UnixMilli(res.Cursor)

			l.Trace("iterated backlogs in partition", "count", iterated)

			// didn't process anything, exit loop
			if iterated == 0 {
				break
			}

			if opt.EnableMillisecondIncrease {
				// shift the starting point 1ms so it doesn't try to grab the same stuff again
				// NOTE: this could result skipping items if the previous batch of items are all on
				// the same millisecond
				ptFrom = ptFrom.Add(time.Millisecond)
			}

			// wait a little before processing the next batch
			<-time.After(opt.Interval)
		}
	}, nil
}

func (q *queue) PartitionBacklogSize(ctx context.Context, partitionID string) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "partitionBacklogSize"), redis_telemetry.ScopeQueue)

	l := logger.StdlibLogger(ctx).With(
		"method", "PartitionBacklogSize",
		"partition_id", partitionID,
	)

	var count int64
	until := q.Clock.Now().Add(24 * time.Hour * 365) // 1y ahead

	log := l.With("shard", q.name)

	backlogs, err := q.BacklogsByPartition(ctx, partitionID, time.Time{}, until)
	if err != nil {
		return 0, fmt.Errorf("could not prepare backlog iterator: %w", err)
	}

	bwg := sync.WaitGroup{}
	for bl := range backlogs {
		bwg.Add(1)
		backlogID := bl.BacklogID

		go func() {
			defer bwg.Done()

			size, err := q.BacklogSize(ctx, backlogID)
			if errors.Is(err, context.Canceled) {
				return
			}
			if err != nil {
				log.ReportError(err, "error retrieving backlog size",
					logger.WithErrorReportTags(map[string]string{
						"backlog":   bl.BacklogID,
						"partition": bl.ShadowPartitionID,
					}),
				)
				return
			}
			atomic.AddInt64(&count, size)
		}()
	}
	bwg.Wait()

	return count, nil
}

func (q *queue) BacklogSize(ctx context.Context, backlogID string) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "backlogSize"), redis_telemetry.ScopeQueue)

	rc := q.RedisClient.Client()
	cmd := rc.B().Zcard().Key(q.RedisClient.kg.BacklogSet(backlogID)).Build()
	count, err := rc.Do(ctx, cmd).AsInt64()
	if rueidis.IsRedisNil(err) {
		return 0, nil
	}
	return count, err
}
