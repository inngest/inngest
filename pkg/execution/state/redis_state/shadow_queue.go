package redis_state

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
)

const (
	AbsoluteShadowPartitionPeekMax int64 = 10 * ShadowPartitionPeekMaxBacklogs

	ShadowPartitionAccountPeekMax  = int64(30)
	ShadowPartitionPeekMax         = int64(300) // same as PartitionPeekMax for now
	ShadowPartitionPeekMinBacklogs = int64(10)
	ShadowPartitionPeekMaxBacklogs = int64(100)

	ShadowPartitionRequeueExtendedDuration = 3 * time.Second

	ShadowPartitionLookahead = 2 * PartitionLookahead

	BacklogForceRequeueMaxBackoff = 5 * time.Minute
)

var (
	ErrShadowPartitionAlreadyLeased               = fmt.Errorf("shadow partition already leased")
	ErrShadowPartitionLeaseNotFound               = fmt.Errorf("shadow partition lease not found")
	ErrShadowPartitionNotFound                    = fmt.Errorf("shadow partition not found")
	ErrShadowPartitionPaused                      = fmt.Errorf("shadow partition refill is disabled")
	ErrShadowPartitionBacklogPeekMaxExceedsLimits = fmt.Errorf("shadow partition backlog peek exceeded the maximum limit of %d", ShadowPartitionPeekMaxBacklogs)
	ErrShadowPartitionPeekMaxExceedsLimits        = fmt.Errorf("shadow partition peek exceeded the maximum limit of %d", ShadowPartitionPeekMax)
	ErrShadowPartitionAccountPeekMaxExceedsLimits = fmt.Errorf("account peek with shadow partitions exceeded the maximum limit of %d", ShadowPartitionAccountPeekMax)
)

var (
	durOpGobalShadowPartitionAccountPeek = "global_shadow_partition_account_peek"
	durOpShadowPartitionRequeue          = "shadow_partition_requeue"
)

// shadowWorker runs a blocking process that listens to item being pushed into the
// shadow queue partition channel. This allows us to process an individual shadow partition.
func (q *queue) shadowWorker(ctx context.Context, qspc chan shadowPartitionChanMsg) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-qspc:
			_, err := durationWithTags(
				ctx,
				q.primaryQueueShard.Name,
				"shadow_partition_process_duration",
				q.clock.Now(),
				func(ctx context.Context) (any, error) {
					err := q.processShadowPartition(ctx, msg.sp, msg.continuationCount)
					if errors.Is(err, context.Canceled) {
						return nil, nil
					}
					return nil, err
				},
				map[string]any{
					// 	"partition_id": msg.sp.PartitionID,
				},
			)
			if err != nil {
				q.log.Error("could not scan shadow partition", "error", err, "shadow_part", msg.sp, "continuation_count", msg.continuationCount)
			}
		}
	}
}

func (q *queue) isMigrationLocked(ctx context.Context, shard QueueShard, fnID uuid.UUID) (*time.Time, error) {
	client := shard.RedisClient.Client()
	kg := shard.RedisClient.KeyGenerator()
	cmd := client.B().Get().Key(kg.QueueMigrationLock(fnID)).Build()
	exists, err := client.Do(ctx, cmd).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not check for migration lock: %w", err)
	}

	parsed, err := ulid.Parse(exists)
	if err != nil {
		return nil, fmt.Errorf("invalid lock format: %w", err)
	}

	lockUntil := parsed.Timestamp()
	return &lockUntil, nil
}

func (q *queue) processShadowPartition(ctx context.Context, shadowPart *QueueShadowPartition, continuationCount uint) error {
	shard := q.primaryQueueShard

	metrics.ActiveShadowScannerCount(ctx, 1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name}})
	defer metrics.ActiveShadowScannerCount(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name}})

	// Check if shadow partition cannot be processed (paused/refill disabled, etc.)
	if shadowPart.FunctionID != nil {
		lockedUntil, err := q.isMigrationLocked(ctx, shard, *shadowPart.FunctionID)
		if err != nil {
			return fmt.Errorf("could not check for migration lock: %w", err)
		}

		if lockedUntil != nil {
			q.removeShadowContinue(ctx, shadowPart, false)
			_, err := durationWithTags(ctx, shard.Name, durOpShadowPartitionRequeue, q.clock.Now(), func(ctx context.Context) (any, error) {
				err := q.ShadowPartitionRequeue(ctx, shadowPart, lockedUntil)
				return nil, err
			}, map[string]any{"reason": "migrating"})
			switch err {
			case nil, ErrShadowPartitionNotFound:
				return nil
			default:
				return fmt.Errorf("could not requeue migrating shadow partition: %w", err)
			}
		}

		// Check paused status with a timeout
		dbCtx, dbCtxCancel := context.WithTimeout(ctx, dbReadTimeout)
		info := q.partitionPausedGetter(dbCtx, *shadowPart.FunctionID)
		dbCtxCancel()

		if info.Paused {
			q.removeShadowContinue(ctx, shadowPart, false)

			if !info.Stale {
				forceRequeueAt := q.clock.Now().Add(ShadowPartitionRefillPausedRequeueExtension)
				_, err := durationWithTags(ctx, shard.Name, durOpShadowPartitionRequeue, q.clock.Now(), func(ctx context.Context) (any, error) {
					err := q.ShadowPartitionRequeue(ctx, shadowPart, &forceRequeueAt)
					return nil, err
				}, map[string]any{"reason": "paused"})
				switch err {
				case nil, ErrShadowPartitionNotFound:
					return nil
				default:
					return fmt.Errorf("could not requeue shadow partition: %w", err)
				}
			}
		}
	}

	// acquire lease for shadow partition
	leaseID, err := duration(ctx, shard.Name, "shadow_partition_lease", q.clock.Now(), func(ctx context.Context) (*ulid.ULID, error) {
		leaseID, err := q.ShadowPartitionLease(ctx, shadowPart, ShadowPartitionLeaseDuration)
		return leaseID, err
	})
	if err != nil {
		q.removeShadowContinue(ctx, shadowPart, false)
		if errors.Is(err, ErrShadowPartitionAlreadyLeased) {
			metrics.IncrQueueShadowPartitionLeaseContentionCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": shard.Name,
					// "partition_id": shadowPart.PartitionID,
					"action": "lease",
				},
			})
			return nil
		}
		if errors.Is(err, ErrShadowPartitionNotFound) {
			metrics.IncrQueueShadowPartitionGoneCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": shard.Name,
					// "partition_id": shadowPart.PartitionID,
				},
			})
			return nil
		}
		if errors.Is(err, ErrShadowPartitionPaused) {
			return nil
		}
		return fmt.Errorf("error leasing shadow partition: %w", err)
	}

	if leaseID == nil {
		return fmt.Errorf("missing shadow partition leaseID")
	}

	extendLeaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx, jobCancel := context.WithCancel(context.WithoutCancel(ctx))
	defer jobCancel()

	// keep extending lease while we're processing
	go func() {
		for {
			select {
			case <-extendLeaseCtx.Done():
				return
			case <-time.Tick(ShadowPartitionLeaseDuration / 2):
				if leaseID == nil {
					return
				}

				newLeaseID, err := q.ShadowPartitionExtendLease(ctx, shadowPart, *leaseID, ShadowPartitionLeaseDuration)
				if err != nil {
					jobCancel()

					// lease contention
					if errors.Is(err, ErrShadowPartitionAlreadyLeased) || errors.Is(err, ErrShadowPartitionLeaseNotFound) {
						metrics.IncrQueueShadowPartitionLeaseContentionCounter(ctx, metrics.CounterOpt{
							PkgName: pkgName,
							Tags: map[string]any{
								"queue_shard": shard.Name,
								// "partition_id": shadowPart.PartitionID,
								"action": "extend_lease",
							},
						})
					}

					return
				}

				if newLeaseID == nil {
					jobCancel()
					return
				}

				leaseID = newLeaseID
			}
		}
	}()

	keyQueuesEnabled := shadowPart.keyQueuesEnabled(ctx, q)

	latestConstraints := q.partitionConstraintConfigGetter(ctx, shadowPart.Identifier())

	limit := ShadowPartitionPeekMaxBacklogs

	// Scan a little further into the future
	refillUntil := q.clock.Now().Truncate(time.Millisecond).Add(ShadowPartitionLookahead)
	if !keyQueuesEnabled {
		// If key queues are disabled, peek and refill all
		// items in the entire backlog, not just the next 2 seconds.
		refillUntil = q.clock.Now().Add(time.Hour * 24 * 365)
	}

	// Pick a random backlog offset every time
	sequential := false

	backlogs, totalCount, err := q.ShadowPartitionPeek(ctx, shadowPart, sequential, refillUntil, limit)
	if err != nil {
		return fmt.Errorf("could not peek backlogs for shadow partition: %w", err)
	}
	metrics.GaugeShadowPartitionSize(ctx, int64(totalCount), metrics.GaugeOpt{
		PkgName: pkgName,
		Tags:    map[string]any{
			// "partition_id": shadowPart.PartitionID,
		},
	})

	// Refill backlogs in random order
	fullyProcessedBacklogs := 0 // Number of fully processed backlogs
	var (
		wasConstrained bool // Whether we encountered constraints affecting the shadow partition
		refilledItems  int  // Number of refilled items
	)

	for _, backlog := range shuffleBacklogs(backlogs) {
		// If cancelled, return early
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}

		res, fullyProcessed, err := q.processShadowPartitionBacklog(ctx, shadowPart, backlog, refillUntil, latestConstraints)
		if err != nil {
			return fmt.Errorf("could not process backlog: %w", err)
		}

		if res != nil {
			refilledItems += res.Refilled
		}

		// If we fully refilled, track and continue
		if fullyProcessed {
			fullyProcessedBacklogs++
			continue
		}

		// If we did not refill, continue on to next backlog
		if res == nil {
			continue
		}

		// If we hit a constraint affecting the entire shadow partition, stop processing other backlogs
		// and requeue the partition early, as we cannot refill items from other backlogs right now.
		switch res.Constraint {
		case enums.QueueConstraintNotLimited:
			continue
		case enums.QueueConstraintAccountConcurrency, enums.QueueConstraintFunctionConcurrency:
			// No more backlogs right now, we can continue the scan loop until new items are added
			q.removeShadowContinue(ctx, shadowPart, false)

			forceRequeueShadowPartitionAt := q.clock.Now().Add(PartitionConcurrencyLimitRequeueExtension)

			_, err = durationWithTags(ctx, shard.Name, durOpShadowPartitionRequeue, q.clock.Now(), func(ctx context.Context) (any, error) {
				err := q.ShadowPartitionRequeue(ctx, shadowPart, &forceRequeueShadowPartitionAt)
				return nil, err
			}, map[string]any{"reason": "concurrency_limited", "cause": res.Constraint.String()})
			switch err {
			case nil, ErrShadowPartitionNotFound: // no-op
				return nil
			default:
				return fmt.Errorf("could not requeue shadow partition: %w", err)
			}
		default:
			wasConstrained = true
		}
	}

	metrics.IncrQueueShadowPartitionProcessedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": shard.Name,
		},
	})

	hasMoreBacklogs := totalCount > fullyProcessedBacklogs
	if !hasMoreBacklogs {
		// No more backlogs right now, we can continue the scan loop until new items are added
		q.removeShadowContinue(ctx, shadowPart, false)

		_, err = durationWithTags(ctx, shard.Name, durOpShadowPartitionRequeue, q.clock.Now(), func(ctx context.Context) (any, error) {
			err := q.ShadowPartitionRequeue(ctx, shadowPart, nil)
			return nil, err
		}, map[string]any{"reason": "empty"})
		switch err {
		case nil, ErrShadowPartitionNotFound: // no-op
			return nil
		default:
			return fmt.Errorf("could not requeue shadow partition: %w", err)
		}
	}

	if wasConstrained {
		// Ran into constraint - remove existing continuation
		q.removeShadowContinue(ctx, shadowPart, false)
	} else {
		// Not constrained so we can add a continuation
		q.addShadowContinue(ctx, shadowPart, continuationCount+1)

		if refilledItems > 0 {

			var accountID uuid.UUID
			if shadowPart.AccountID != nil {
				accountID = *shadowPart.AccountID
			}

			// Add an in-memory hint to process the partition immediately if we refilled items
			q.addContinue(ctx, &QueuePartition{
				ID:            shadowPart.PartitionID,
				PartitionType: int(enums.PartitionTypeDefault),
				QueueName:     shadowPart.SystemQueueName,
				FunctionID:    shadowPart.FunctionID,
				EnvID:         shadowPart.EnvID,
				AccountID:     accountID,
				Last:          0, // This is populated during PartitionLease
			}, 1)
		}
	}

	_, err = durationWithTags(ctx, shard.Name, durOpShadowPartitionRequeue, q.clock.Now(), func(ctx context.Context) (any, error) {
		err := q.ShadowPartitionRequeue(ctx, shadowPart, nil)
		return nil, err
	}, map[string]any{"reason": "handled"})
	switch err {
	case nil, ErrShadowPartitionNotFound:
		return nil
	default:
		return fmt.Errorf("could not requeue shadow partition: %w", err)
	}
}

func (q *queue) processShadowPartitionBacklog(
	ctx context.Context,
	shadowPart *QueueShadowPartition,
	backlog *QueueBacklog,
	refillUntil time.Time,
	constraints PartitionConstraintConfig,
) (*BacklogRefillResult, bool, error) {
	enableKeyQueues := shadowPart.SystemQueueName != nil && q.enqueueSystemQueuesToBacklog
	if shadowPart.AccountID != nil {
		enableKeyQueues = q.allowKeyQueues(ctx, *shadowPart.AccountID)
	}

	// May need to normalize - this will not happen for default backlogs
	if reason := backlog.isOutdated(constraints); enableKeyQueues && reason != enums.QueueNormalizeReasonUnchanged {
		l := q.log.With(
			"sp", shadowPart,
			"constraints", constraints,
			"backlog", backlog,
			"reason", reason,
		)

		l.Debug("outdated backlog")

		metrics.IncrQueueOutdatedBacklogCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard": q.primaryQueueShard.Name,
				// "partition_id": shadowPart.PartitionID,
				"reason": reason.String(),
			},
		})

		// ensure exclusive access to backlog
		if _, err := duration(ctx, q.primaryQueueShard.Name, "normalize_lease", q.clock.Now(), func(ctx context.Context) (any, error) {
			err := q.leaseBacklogForNormalization(ctx, backlog)
			return nil, err
		}); err != nil {
			if errors.Is(err, errBacklogAlreadyLeasedForNormalization) {
				return nil, false, nil
			}

			return nil, false, fmt.Errorf("could not lease backlog: %w", err)
		}

		// Prepare normalization, this will just run once as the shadow scanner
		// won't pick it up again after this.
		err := q.BacklogPrepareNormalize(ctx, backlog, shadowPart)
		if err != nil && !errors.Is(err, ErrBacklogGarbageCollected) {
			return nil, false, fmt.Errorf("could not prepare backlog for normalization: %w", err)
		}

		// If backlog was empty and garbage-collected, exit early
		if errors.Is(err, ErrBacklogGarbageCollected) {
			l.Debug("garbage-collected empty backlog")
			return nil, false, nil
		}

		return nil, false, nil
	}

	refillLimit := q.backlogRefillLimit
	if refillLimit > BacklogRefillHardLimit {
		refillLimit = BacklogRefillHardLimit
	}
	if refillLimit <= 0 {
		refillLimit = BacklogRefillHardLimit
	}

	// Peek items (scheduled to run within the next 2s) to be refilled.
	//
	// Peek will delete missing items at the time of peeking.
	//
	// NOTE: We are guaranteed to be the only refilling process for this backlog.
	// There may be concurrent backlog modifications in the data store:
	// - Items may be added to the backlog (earlier or later than peeked items)
	// - Items may be removed from the backlog (dequeued by a cancellation, etc.)
	// - Items may be requeued within the backlog (changing the score to an earlier or later time)
	//
	// Missing items are gracefully handled by BacklogRefill.
	//
	// Items that were added between backlogPeek and BacklogRefill will be considered in the next refill.
	// Items that were moved between backlogPeek and BacklogRefill will still be refilled.
	items, total, err := q.backlogPeek(ctx, backlog, time.Time{}, refillUntil, refillLimit)
	if err != nil {
		return nil, false, fmt.Errorf("could not peek backlog items for refill: %w", err)
	}

	if len(items) == 0 {
		return nil, false, nil
	}

	constraintCheckRes, err := q.backlogRefillConstraintCheck(ctx, shadowPart, backlog, constraints, items)
	if err != nil {
		return nil, false, fmt.Errorf("could not check constraints for backlogRefill: %w", err)
	}

	// If no items can be refilled, exit early
	if len(constraintCheckRes.itemsToRefill) == 0 {
		return &BacklogRefillResult{
			Constraint:        constraintCheckRes.limitingConstraint,
			Refilled:          0,
			Refill:            len(items),
			BacklogCountUntil: total,
			TotalBacklogCount: 0, // Not fetched
			Capacity:          0,
			RefilledItems:     nil,
			RetryAt:           constraintCheckRes.retryAfter,
		}, false, nil
	}

	res, err := durationWithTags(
		ctx,
		q.primaryQueueShard.Name,
		"backlog_process_duration",
		q.clock.Now(),
		func(ctx context.Context) (*BacklogRefillResult, error) {
			return q.BacklogRefill(ctx, backlog, shadowPart, refillUntil, constraintCheckRes.itemsToRefill, constraints)
		},
		map[string]any{
			//	"partition_id": shadowPart.PartitionID,
		},
	)
	if err != nil {
		return nil, false, fmt.Errorf("could not refill backlog: %w", err)
	}

	// Report limiting constraint
	if res.Constraint == enums.QueueConstraintNotLimited && constraintCheckRes.limitingConstraint != enums.QueueConstraintNotLimited {
		res.Constraint = constraintCheckRes.limitingConstraint
	}

	q.log.Trace("processed backlog",
		"backlog", backlog.BacklogID,
		"total", res.TotalBacklogCount,
		"until", res.BacklogCountUntil,
		"constrained", res.Constraint,
		"capacity", res.Capacity,
		"refill", res.Refill,
		"refilled", res.Refilled,
		"constraints", constraints,
		"backlog_throttle", backlog.Throttle,
	)

	if len(res.RefilledItems) > 0 && mrand.Float64() < 0.05 {
		q.log.Debug(
			"refilled items to ready queue",
			"job_id", res.RefilledItems,
			"backlog", backlog.BacklogID,
			"partition", shadowPart.PartitionID,
		)
	}

	// instrumentation
	{
		opts := metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard": q.primaryQueueShard.Name,
				// "partition_id": shadowPart.PartitionID,
			},
		}

		metrics.IncrBacklogProcessedCounter(ctx, opts)
		metrics.IncrQueueBacklogRefilledCounter(ctx, int64(res.Refilled), opts)

		switch res.Constraint {
		case enums.QueueConstraintNotLimited: // no-op
		default:
			// NOTE:
			// we don't want to add an extended amount of time for requeue when there are
			// contraint hits, so we make sure to check more often in order to admit items
			// into processing
			metrics.IncrQueueBacklogRefillConstraintCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.primaryQueueShard.Name,
					// "partition_id": shadowPart.PartitionID,
					"constraint": res.Constraint.String(),
				},
			})

			q.lifecycles.OnBacklogRefillConstraintHit(ctx, shadowPart, backlog, res)
		}

		// NOTE: custom method to instrument result - potentially handling high cardinality data
		q.lifecycles.OnBacklogRefilled(ctx, shadowPart, backlog, res)

		// Invoke previous constraint lifecycles to update UI
		switch res.Constraint {
		case enums.QueueConstraintAccountConcurrency:
			if shadowPart.AccountID != nil {
				q.lifecycles.OnAccountConcurrencyLimitReached(context.WithoutCancel(ctx), *shadowPart.AccountID, shadowPart.EnvID)
			}
		case enums.QueueConstraintFunctionConcurrency:
			if shadowPart.FunctionID != nil {
				q.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *shadowPart.FunctionID)
			}
		case enums.QueueConstraintCustomConcurrencyKey1:
			if shadowPart.FunctionID != nil {
				q.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *shadowPart.FunctionID)
			}
			if len(backlog.ConcurrencyKeys) > 0 {
				q.lifecycles.OnCustomKeyConcurrencyLimitReached(context.WithoutCancel(ctx), backlog.ConcurrencyKeys[0].CanonicalKeyID)
			}
		case enums.QueueConstraintCustomConcurrencyKey2:
			if shadowPart.FunctionID != nil {
				q.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *shadowPart.FunctionID)
			}
			if len(backlog.ConcurrencyKeys) > 1 {
				q.lifecycles.OnCustomKeyConcurrencyLimitReached(context.WithoutCancel(ctx), backlog.ConcurrencyKeys[1].CanonicalKeyID)
			}
		default:
		}
	}

	forceRequeueBacklogAt := res.RetryAt
	switch res.Constraint {
	// If backlog is concurrency limited by custom key, requeue just this backlog in the future
	case enums.QueueConstraintCustomConcurrencyKey1, enums.QueueConstraintCustomConcurrencyKey2:
		forceRequeueBacklogAt = backlog.requeueBackOff(q.clock.Now(), res.Constraint)
	}

	if !forceRequeueBacklogAt.IsZero() {
		// Cap maximum backoff to ensure constraint updates (changing throttle period, etc.) are reflected reasonably quickly
		if forceRequeueBacklogAt.Sub(q.clock.Now()) > BacklogForceRequeueMaxBackoff {
			forceRequeueBacklogAt = q.clock.Now().Add(BacklogForceRequeueMaxBackoff)
		}

		if err := q.BacklogRequeue(ctx, backlog, shadowPart, forceRequeueBacklogAt); err != nil && !errors.Is(err, ErrBacklogNotFound) {
			return nil, false, fmt.Errorf("could not requeue backlog: %w", err)
		}
	}

	remainingItems := res.TotalBacklogCount - res.Refilled
	fullyProcessedBacklog := remainingItems == 0
	return res, fullyProcessedBacklog, nil
}

type shadowPartitionChanMsg struct {
	sp                *QueueShadowPartition
	continuationCount uint
}

func (q *queue) scanShadowContinuations(ctx context.Context) error {
	if !q.runMode.ShadowContinuations {
		return nil
	}

	if mrand.Float64() <= q.runMode.ShadowContinuationSkipProbability {
		return nil
	}

	eg := errgroup.Group{}
	q.shadowContinuesLock.Lock()
	for _, c := range q.shadowContinues {
		cont := c
		eg.Go(func() error {
			sp := cont.shadowPart

			_, err := durationWithTags(
				ctx,
				q.primaryQueueShard.Name,
				"shadow_partition_process_duration",
				q.clock.Now(),
				func(ctx context.Context) (any, error) {
					err := q.processShadowPartition(ctx, sp, cont.count)
					if errors.Is(err, context.Canceled) {
						return nil, nil
					}
					return nil, err
				},
				map[string]any{
					// "partition_id": sp.PartitionID,
				},
			)
			if err != nil {
				if err == ErrShadowPartitionLeaseNotFound {
					return nil
				}
				if !errors.Is(err, context.Canceled) {
					q.log.Error("error processing shadow partition", "error", err, "continuation", true, "continuation_count", cont.count)
				}
				return err
			}

			return nil
		})
	}
	q.shadowContinuesLock.Unlock()
	return eg.Wait()
}

func (q *queue) scanShadowPartitions(ctx context.Context, until time.Time, qspc chan shadowPartitionChanMsg) error {
	// check whether continuations are enabled and apply chance of skipping continuations in this iteration
	if err := q.scanShadowContinuations(ctx); err != nil {
		return fmt.Errorf("error scanning shadow continuations: %w", err)
	}

	shouldScanAccount := q.runMode.AccountShadowPartition && mrand.Intn(100) <= q.runMode.AccountShadowPartitionWeight
	if len(q.runMode.ExclusiveAccounts) > 0 {
		shouldScanAccount = true
	}

	if shouldScanAccount {
		sequential := false

		var peekedAccounts []uuid.UUID
		if len(q.runMode.ExclusiveAccounts) > 0 {
			peekedAccounts = q.runMode.ExclusiveAccounts
		} else {
			peeked, err := duration(ctx, q.primaryQueueShard.Name, durOpGobalShadowPartitionAccountPeek, q.clock.Now(), func(ctx context.Context) ([]uuid.UUID, error) {
				return q.peekGlobalShadowPartitionAccounts(ctx, sequential, until, ShadowPartitionAccountPeekMax)
			})
			if err != nil {
				return fmt.Errorf("could not peek global shadow partition accounts: %w", err)
			}
			peekedAccounts = peeked
		}

		if len(peekedAccounts) == 0 {
			return nil
		}

		// Reduce number of peeked partitions as we're processing multiple accounts in parallel
		// Note: This is not optimal as some accounts may have fewer partitions than others and
		// we're leaving capacity on the table. We'll need to find a better way to determine the
		// optimal peek size in this case.
		accountPartitionPeekMax := int64(math.Round(float64(ShadowPartitionPeekMax / int64(len(peekedAccounts)))))

		// Scan and process account partitions in parallel
		wg := sync.WaitGroup{}
		for _, account := range peekedAccounts {
			account := account

			wg.Add(1)
			go func(account uuid.UUID) {
				defer wg.Done()
				partitionKey := q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(account)

				parts, err := q.peekShadowPartitions(ctx, partitionKey, sequential, accountPartitionPeekMax, until)
				if err != nil && !errors.Is(err, context.Canceled) {
					q.log.ReportError(err, "error peeking account partition",
						logger.WithErrorReportTags(map[string]string{
							"account_id": account.String(),
						}),
					)
					return
				}

				for _, part := range parts {
					qspc <- shadowPartitionChanMsg{
						sp:                part,
						continuationCount: 0,
					}
				}
			}(account)
		}

		wg.Wait()

		return nil
	}

	kg := q.primaryQueueShard.RedisClient.kg
	sequential := false
	parts, err := q.peekShadowPartitions(ctx, kg.GlobalShadowPartitionSet(), sequential, ShadowPartitionPeekMax, until)
	if err != nil {
		return fmt.Errorf("could not peek global shadow partitions: %w", err)
	}

	for _, part := range parts {
		qspc <- shadowPartitionChanMsg{
			sp:                part,
			continuationCount: 0,
		}
	}

	return nil
}

// shadowScan iterates through the shadow partitions and attempt to add queue items
// to the function partition for processing
func (q *queue) shadowScan(ctx context.Context) error {
	l := q.log.With("method", "shadowScan")
	qspc := make(chan shadowPartitionChanMsg)

	for i := int32(0); i < q.numShadowWorkers; i++ {
		go q.shadowWorker(ctx, qspc)
	}

	tick := q.clock.NewTicker(q.shadowPollTick)
	l.Debug("starting shadow scanner", "poll", q.shadowPollTick.String())

	backoff := 200 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return nil

		case <-tick.Chan():
			// Scan a little further into the future
			now := q.clock.Now()
			scanUntil := now.Truncate(time.Second).Add(ShadowPartitionLookahead)
			if err := q.scanShadowPartitions(ctx, scanUntil, qspc); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					l.Warn("deadline exceeded scanning shadow partitions")
					<-time.After(backoff)

					// Backoff doubles up to 5 seconds
					backoff = time.Duration(math.Min(float64(backoff*2), float64(5*time.Second)))
					continue
				}

				if !errors.Is(err, context.Canceled) {
					l.ReportError(err, "error scanning shadow partitions")
				}
				return fmt.Errorf("error scanning shadow partitions: %w", err)
			}

			backoff = 200 * time.Millisecond
		}
	}
}

// peekShadowPartitions returns pending shadow partitions within the global shadow partition pointer _or_ account shadow partition pointer ZSET.
func (q *queue) peekShadowPartitions(ctx context.Context, partitionIndexKey string, sequential bool, peekLimit int64, until time.Time) ([]*QueueShadowPartition, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for peekShadowPartitions: %s", q.primaryQueueShard.Kind)
	}

	p := peeker[QueueShadowPartition]{
		q:               q,
		opName:          "peekShadowPartitions",
		keyMetadataHash: q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		max:             ShadowPartitionPeekMax,
		maker: func() *QueueShadowPartition {
			return &QueueShadowPartition{}
		},
		handleMissingItems: func(pointers []string) error {
			return nil
		},
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, partitionIndexKey, sequential, until, peekLimit)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, ErrShadowPartitionPeekMaxExceedsLimits
		}
		return nil, fmt.Errorf("could not peek shadow partitions: %w", err)
	}

	if res.TotalCount > 0 {
		for _, p := range res.Items {
			q.log.Trace("peeked shadow partition", "partition_id", p.PartitionID, "until", until.Format(time.StampMilli))
		}
	}

	return res.Items, nil
}

// addShadowContinue is the equivalent of addContinue for shadow partitions
func (q *queue) addShadowContinue(ctx context.Context, p *QueueShadowPartition, ctr uint) {
	if !q.runMode.ShadowContinuations {
		// shadow continuations are not enabled.
		return
	}

	if ctr >= q.shadowContinuationLimit {
		q.removeShadowContinue(ctx, p, true)
		return
	}

	q.shadowContinuesLock.Lock()
	defer q.shadowContinuesLock.Unlock()

	// If this is the first shadow continuation, check if we're on a cooldown, or if we're
	// beyond capacity.
	if ctr == 1 {
		if len(q.shadowContinues) > consts.QueueShadowContinuationMaxPartitions {
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name, "op": "max_capacity"}})
			return
		}
		if t, ok := q.shadowContinueCooldown[p.PartitionID]; ok && t.After(time.Now()) {
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name, "op": "cooldown"}})
			return
		}

		// Remove the shadow continuation cooldown.
		delete(q.shadowContinueCooldown, p.PartitionID)
	}

	c, ok := q.shadowContinues[p.PartitionID]
	if !ok || c.count < ctr {
		// Update the continue count if it doesn't exist, or the current counter
		// is higher.  This ensures that we always have the highest continuation
		// count stored for queue processing.
		q.shadowContinues[p.PartitionID] = shadowContinuation{shadowPart: p, count: ctr}
		metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name, "op": "added"}})
	}
}

func (q *queue) removeShadowContinue(ctx context.Context, p *QueueShadowPartition, cooldown bool) {
	if !q.runMode.ShadowContinuations {
		// shadow continuations are not enabled.
		return
	}

	// This is over the limit for continuing the shadow partition, so force it to be
	// removed in every case.
	q.shadowContinuesLock.Lock()
	defer q.shadowContinuesLock.Unlock()

	metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name, "op": "removed"}})

	delete(q.shadowContinues, p.PartitionID)

	if cooldown {
		// Add a cooldown, preventing this partition from being added as a continuation
		// for a given period of time.
		//
		// Note that this isn't shared across replicas;  cooldowns
		// only exist in the current replica.
		q.shadowContinueCooldown[p.PartitionID] = time.Now().Add(
			consts.QueueShadowContinuationCooldownPeriod,
		)
	}
}

func (q *queue) ShadowPartitionPeek(ctx context.Context, sp *QueueShadowPartition, sequential bool, until time.Time, limit int64, opts ...PeekOpt) ([]*QueueBacklog, int, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, 0, fmt.Errorf("unsupported queue shard kind for ShadowPartitionPeek: %s", q.primaryQueueShard.Kind)
	}

	opt := peekOption{}
	for _, apply := range opts {
		apply(&opt)
	}

	rc := q.primaryQueueShard.RedisClient
	if opt.Shard != nil {
		rc = opt.Shard.RedisClient
	}

	shadowPartitionSet := rc.kg.ShadowPartitionSet(sp.PartitionID)

	p := peeker[QueueBacklog]{
		q:               q,
		opName:          "ShadowPartitionPeek",
		keyMetadataHash: rc.kg.BacklogMeta(),
		max:             ShadowPartitionPeekMaxBacklogs,
		maker: func() *QueueBacklog {
			return &QueueBacklog{}
		},
		handleMissingItems:     CleanupMissingPointers(ctx, shadowPartitionSet, rc.Client(), q.log.With("sp", sp)),
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, shadowPartitionSet, sequential, until, limit, opts...)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, 0, ErrShadowPartitionBacklogPeekMaxExceedsLimits
		}
		return nil, 0, fmt.Errorf("could not peek shadow partition backlogs: %w", err)
	}

	return res.Items, res.TotalCount, nil
}

func (q *queue) ShadowPartitionExtendLease(ctx context.Context, sp *QueueShadowPartition, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionExtendLease"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ShadowPartitionExtendLease: %s", q.primaryQueueShard.Kind)
	}

	now := q.clock.Now()
	leaseExpiry := now.Add(duration)
	newLeaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("could not generate new leaseID: %w", err)
	}

	sp.LeaseID = &newLeaseID

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		q.primaryQueueShard.RedisClient.kg.GlobalShadowPartitionSet(),
		q.primaryQueueShard.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(accountID),
	}
	args, err := StrSlice([]any{
		sp.PartitionID,
		accountID,
		leaseID,
		newLeaseID,
		now.UnixMilli(),
		leaseExpiry.UnixMilli(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/shadowPartitionExtendLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "shadowPartitionExtendLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error extending shadow partition lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case -1:
		return nil, ErrShadowPartitionNotFound
	case -2:
		return nil, ErrShadowPartitionLeaseNotFound
	case -3:
		return nil, ErrShadowPartitionAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response extending shadow partition lease: %v (%T)", status, status)
	}
}

func (q *queue) ShadowPartitionRequeue(ctx context.Context, sp *QueueShadowPartition, requeueAt *time.Time) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionRequeue"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for ShadowPartitionRequeue: %s", q.primaryQueueShard.Kind)
	}

	sp.LeaseID = nil

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	var requeueAtMS int64
	var requeueAtStr string
	if requeueAt != nil {
		requeueAtMS = requeueAt.UnixMilli()
		requeueAtStr = requeueAt.Format(time.StampMilli)
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		q.primaryQueueShard.RedisClient.kg.GlobalShadowPartitionSet(),
		q.primaryQueueShard.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(accountID),
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionSet(sp.PartitionID),
	}
	args, err := StrSlice([]any{
		sp.PartitionID,
		accountID,
		q.clock.Now().UnixMilli(),
		requeueAtMS,
	})
	if err != nil {
		return fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/shadowPartitionRequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "shadowPartitionRequeue"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error returning shadow partition lease: %w", err)
	}

	q.log.Trace("requeued shadow partition",
		"id", sp.PartitionID,
		"time", requeueAtStr,
		"status", status,
	)

	switch status {
	case 0:
		return nil
	case -1:
		metrics.IncrQueueShadowPartitionLeaseContentionCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard": q.primaryQueueShard.Name,
				// "partition_id": sp.PartitionID,
				"action": "not_found",
			},
		})

		return ErrShadowPartitionNotFound
	default:
		return fmt.Errorf("unknown response returning shadow partition lease: %v (%T)", status, status)
	}
}

func (q *queue) ShadowPartitionLease(ctx context.Context, sp *QueueShadowPartition, duration time.Duration) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionLease"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ShadowPartitionLease: %s", q.primaryQueueShard.Kind)
	}

	now := q.clock.Now()
	leaseExpiry := now.Add(duration)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("could not generate leaseID: %w", err)
	}

	sp.LeaseID = &leaseID

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		q.primaryQueueShard.RedisClient.kg.GlobalShadowPartitionSet(),
		q.primaryQueueShard.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(accountID),
	}
	args, err := StrSlice([]any{
		sp.PartitionID,
		accountID,
		leaseID,
		now.UnixMilli(),
		leaseExpiry.UnixMilli(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/shadowPartitionLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "shadowPartitionLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error leasing shadow partition: %w", err)
	}
	switch status {
	case 0:
		return &leaseID, nil
	case -1:
		return nil, ErrShadowPartitionNotFound
	case -2:
		return nil, ErrShadowPartitionAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response leasing shadow partition: %v (%T)", status, status)
	}
}

func (q *queue) peekGlobalShadowPartitionAccounts(ctx context.Context, sequential bool, until time.Time, limit int64) ([]uuid.UUID, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for peekGlobalShadowPartitionAccounts: %s", q.primaryQueueShard.Kind)
	}

	rc := q.primaryQueueShard.RedisClient

	p := peeker[QueueBacklog]{
		q:                      q,
		opName:                 "peekGlobalShadowPartitionAccounts",
		max:                    ShadowPartitionAccountPeekMax,
		isMillisecondPrecision: true,
	}

	return p.peekUUIDPointer(ctx, rc.kg.GlobalAccountShadowPartitions(), sequential, until, limit)
}
