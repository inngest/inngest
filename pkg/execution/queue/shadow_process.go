package queue

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

var (
	durOpGobalShadowPartitionAccountPeek = "global_shadow_partition_account_peek"
	durOpShadowPartitionRequeue          = "shadow_partition_requeue"
)

func (q *queueProcessor) processShadowPartition(ctx context.Context, shadowPart *QueueShadowPartition, continuationCount uint) error {
	l := logger.StdlibLogger(ctx).With(
		"partition_id", shadowPart.PartitionID,
		"account_id", shadowPart.AccountID,
	)
	shard := q.primaryQueueShard

	metrics.ActiveShadowScannerCount(ctx, 1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name}})
	defer metrics.ActiveShadowScannerCount(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name}})

	// Check if shadow partition cannot be processed (paused/refill disabled, etc.)
	if shadowPart.FunctionID != nil {
		lockedUntil, err := q.primaryQueueShard.IsMigrationLocked(ctx, *shadowPart.FunctionID)
		if err != nil {
			return fmt.Errorf("could not check for migration lock: %w", err)
		}

		if lockedUntil != nil {
			q.removeShadowContinue(ctx, shadowPart, false)
			_, err := DurationWithTags(ctx, q.primaryQueueShard.Name(), durOpShadowPartitionRequeue, q.Clock().Now(), func(ctx context.Context) (any, error) {
				err := q.primaryQueueShard.ShadowPartitionRequeue(ctx, shadowPart, lockedUntil)
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
		dbCtx, dbCtxCancel := context.WithTimeout(ctx, DatabaseReadTimeout)
		info := q.PartitionPausedGetter(dbCtx, *shadowPart.FunctionID)

		if dbCtx.Err() == context.DeadlineExceeded {
			metrics.IncrQueueDatabaseContextTimeoutCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"operation": "shadow_partition_paused_getter",
				},
			})
		}

		dbCtxCancel()

		if info.Paused {
			q.removeShadowContinue(ctx, shadowPart, false)

			if !info.Stale {
				forceRequeueAt := q.Clock().Now().Add(ShadowPartitionRefillPausedRequeueExtension)
				_, err := DurationWithTags(ctx, shard.Name(), durOpShadowPartitionRequeue, q.Clock().Now(), func(ctx context.Context) (any, error) {
					err := q.primaryQueueShard.ShadowPartitionRequeue(ctx, shadowPart, &forceRequeueAt)
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
	leaseID, err := Duration(ctx, shard.Name(), "shadow_partition_lease", q.Clock().Now(), func(ctx context.Context) (*ulid.ULID, error) {
		leaseID, err := q.primaryQueueShard.ShadowPartitionLease(ctx, shadowPart, ShadowPartitionLeaseDuration)
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

				newLeaseID, err := q.primaryQueueShard.ShadowPartitionExtendLease(ctx, shadowPart, *leaseID, ShadowPartitionLeaseDuration)
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

	keyQueuesEnabled := shadowPart.KeyQueuesEnabled(ctx, q.QueueOptions)

	latestConstraints := q.PartitionConstraintConfigGetter(ctx, shadowPart.Identifier())

	limit := ShadowPartitionPeekMaxBacklogs

	// Scan a little further into the future
	refillUntil := q.Clock().Now().Truncate(time.Millisecond).Add(ShadowPartitionLookahead)
	if !keyQueuesEnabled {
		// If key queues are disabled, peek and refill all
		// items in the entire backlog, not just the next 2 seconds.
		refillUntil = q.Clock().Now().Add(time.Hour * 24 * 365)
	}

	// Pick a random backlog offset every time
	sequential := false

	backlogs, totalCount, err := q.primaryQueueShard.ShadowPartitionPeek(ctx, shadowPart, sequential, refillUntil, limit)
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

	// Always shuffle backlogs while prioritizing non-start backlogs.
	// This is necessary to ensure we refill items to finish existing runs before
	// refilling run starts.
	backlogs = ShuffleBacklogs(backlogs)

	// If throttle is configured without custom concurrency keys, we have a mismatch:
	// - Each start queue item is added to a dedicated backlog per key
	// - Non-start queue items are added to the default backlog
	//
	// In this case, we always want to refill the default backlog first to ensure existing
	// runs can finish before new runs are started.
	if latestConstraints.Throttle != nil && len(latestConstraints.Concurrency.CustomConcurrencyKeys) == 0 {
		// Create non-start function backlog
		fnBacklog := shadowPart.DefaultBacklog(latestConstraints, false)
		if fnBacklog != nil {
			l.Trace("refilling from fn backlog for fairness", "backlog_id", fnBacklog.BacklogID)

			// Start with non-start function backlog
			backlogs = append([]*QueueBacklog{fnBacklog}, backlogs...)
		}
	}

	for _, backlog := range backlogs {
		// Apply a refill multiplier: Some backlogs should receive more refill capacity.
		// While the global refill limit controls how many items should be refilled per backlog, the multiplier
		// applies a backlog and constraint-specific policy to determine if the backlog should be prioritized.
		multiplier := BacklogRefillMultiplier(backlogs, backlog, latestConstraints)
		for i := range multiplier {
			l := l.With(
				"multiplier", multiplier,
				"multiplier_index", i,
			)

			// If cancelled, return early
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}

			res, fullyProcessed, err := q.processShadowPartitionBacklog(logger.WithStdlib(ctx, l), shadowPart, backlog, refillUntil, latestConstraints)
			if err != nil {
				return fmt.Errorf("could not process backlog: %w", err)
			}

			if res != nil {
				refilledItems += res.Refilled
			}

			// If we fully refilled, track and continue
			if fullyProcessed {
				fullyProcessedBacklogs++
				break // continue with next backlog
			}

			// If we did not refill, continue on to next backlog
			if res == nil {
				break // continue with next backlog
			}

			// If we hit a constraint affecting the entire shadow partition, stop processing other backlogs
			// and requeue the partition early, as we cannot refill items from other backlogs right now.
			switch res.Constraint {
			case enums.QueueConstraintNotLimited:
				// continue with next backlog
			case enums.QueueConstraintAccountConcurrency, enums.QueueConstraintFunctionConcurrency:
				l.Trace("limited by concurrency, requeueing shadow partition in the future",
					"scope", res.Constraint,
				)

				// No more backlogs right now, we can continue the scan loop until new items are added
				q.removeShadowContinue(ctx, shadowPart, false)

				forceRequeueShadowPartitionAt := q.Clock().Now().Add(PartitionConcurrencyLimitRequeueExtension)

				_, err = DurationWithTags(ctx, shard.Name(), durOpShadowPartitionRequeue, q.Clock().Now(), func(ctx context.Context) (any, error) {
					err := q.primaryQueueShard.ShadowPartitionRequeue(ctx, shadowPart, &forceRequeueShadowPartitionAt)
					return nil, err
				}, map[string]any{"reason": "concurrency_limited", "cause": res.Constraint.String()})
				switch err {
				case nil, ErrShadowPartitionNotFound: // no-op
					return nil
				default:
					return fmt.Errorf("could not requeue shadow partition: %w", err)
				}
			default:
				// backlog was constrained, continue with others until the shadow partition is constrained
				wasConstrained = true
			}
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
		l.Trace("no more backlogs in shadow partition")

		// No more backlogs right now, we can continue the scan loop until new items are added
		q.removeShadowContinue(ctx, shadowPart, false)

		_, err = DurationWithTags(ctx, shard.Name(), durOpShadowPartitionRequeue, q.Clock().Now(), func(ctx context.Context) (any, error) {
			err := q.primaryQueueShard.ShadowPartitionRequeue(ctx, shadowPart, nil)
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

		// Hint to the executor
		if refilledItems > 0 {
			l.Trace("hinting to executor after refilling items")

			var accountID uuid.UUID
			if shadowPart.AccountID != nil {
				accountID = *shadowPart.AccountID
			}

			// Add an in-memory hint to process the partition immediately if we refilled items
			q.addContinue(ctx, &QueuePartition{
				ID:         shadowPart.PartitionID,
				QueueName:  shadowPart.SystemQueueName,
				FunctionID: shadowPart.FunctionID,
				EnvID:      shadowPart.EnvID,
				AccountID:  accountID,
				Last:       0, // This is populated during PartitionLease
			}, 1)
		}
	}

	_, err = DurationWithTags(ctx, shard.Name(), durOpShadowPartitionRequeue, q.Clock().Now(), func(ctx context.Context) (any, error) {
		err := q.primaryQueueShard.ShadowPartitionRequeue(ctx, shadowPart, nil)
		return nil, err
	}, map[string]any{"reason": "handled"})
	switch err {
	case nil, ErrShadowPartitionNotFound:
		return nil
	default:
		return fmt.Errorf("could not requeue shadow partition: %w", err)
	}
}

func (q *queueProcessor) processShadowPartitionBacklog(
	ctx context.Context,
	shadowPart *QueueShadowPartition,
	backlog *QueueBacklog,
	refillUntil time.Time,
	constraints PartitionConstraintConfig,
) (*BacklogRefillResult, bool, error) {
	l := logger.StdlibLogger(ctx).With(
		"backlog_id", backlog.BacklogID,
	)

	var enableKeyQueues bool
	if shadowPart.AccountID != nil && shadowPart.FunctionID != nil {
		enableKeyQueues = q.AllowKeyQueues(ctx, *shadowPart.AccountID, *shadowPart.FunctionID)
	}

	// May need to normalize - this will not happen for default backlogs
	if reason := backlog.IsOutdated(constraints); enableKeyQueues && reason != enums.QueueNormalizeReasonUnchanged {
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
		if _, err := Duration(ctx, q.primaryQueueShard.Name(), "normalize_lease", q.Clock().Now(), func(ctx context.Context) (any, error) {
			err := q.primaryQueueShard.LeaseBacklogForNormalization(ctx, backlog)
			return nil, err
		}); err != nil {
			if errors.Is(err, ErrBacklogAlreadyLeasedForNormalization) {
				return nil, false, nil
			}

			return nil, false, fmt.Errorf("could not lease backlog: %w", err)
		}

		// Prepare normalization, this will just run once as the shadow scanner
		// won't pick it up again after this.
		err := q.primaryQueueShard.BacklogPrepareNormalize(ctx, backlog, shadowPart)
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
	items, total, err := q.primaryQueueShard.BacklogPeek(ctx, backlog, time.Time{}, refillUntil, refillLimit)
	if err != nil {
		return nil, false, fmt.Errorf("could not peek backlog items for refill: %w", err)
	}

	if len(items) == 0 {
		return nil, false, nil
	}

	// NOTE: This idempotency key is simply used for retrying Acquire
	// We do not use the same key for multiple processShadowPartitionBacklog attempts
	now := q.Clock().Now()
	operationIdempotencyKey := fmt.Sprintf("%s-%d", backlog.BacklogID, now.UnixMilli())

	constraintCheckRes, err := q.primaryQueueShard.BacklogRefillConstraintCheck(ctx, shadowPart, backlog, constraints, items, operationIdempotencyKey, now)
	if err != nil {
		return nil, false, fmt.Errorf("could not check constraints for backlogRefill: %w", err)
	}

	// In case the Constraint API determines no work can happen right now, we will report the limit
	// and respect the retryAfter value
	res := &BacklogRefillResult{
		Constraint:        constraintCheckRes.LimitingConstraint,
		RetryAt:           constraintCheckRes.RetryAfter,
		BacklogCountUntil: total,
	}

	// If no items can be refilled, exit early
	if len(constraintCheckRes.ItemsToRefill) > 0 {
		res, err = DurationWithTags(
			ctx,
			q.primaryQueueShard.Name(),
			"backlog_process_duration",
			q.Clock().Now(),
			func(ctx context.Context) (*BacklogRefillResult, error) {
				return q.primaryQueueShard.BacklogRefill(
					ctx,
					backlog,
					shadowPart,
					refillUntil,
					constraintCheckRes.ItemsToRefill,
					constraints,
					WithBacklogRefillConstraintCheckIdempotencyKey(operationIdempotencyKey),
					WithBacklogRefillDisableConstraintChecks(constraintCheckRes.SkipConstraintChecks),
					WithBacklogRefillItemCapacityLeases(constraintCheckRes.ItemCapacityLeases),
				)
			},
			map[string]any{
				//	"partition_id": shadowPart.PartitionID,
			},
		)
		if err != nil {
			return nil, false, fmt.Errorf("could not refill backlog: %w", err)
		}
	} else {
		l.Trace("no items to refill after capacity check", "limiting", res.Constraint)
	}

	// Report limiting constraint
	if res.Constraint == enums.QueueConstraintNotLimited && constraintCheckRes.LimitingConstraint != enums.QueueConstraintNotLimited {
		res.Constraint = constraintCheckRes.LimitingConstraint
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

	if len(res.RefilledItems) > 0 && rand.Float64() < 0.05 {
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
				"queue_shard":    q.primaryQueueShard.Name,
				"constraint_api": constraintCheckRes.SkipConstraintChecks,
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
		forceRequeueBacklogAt = backlog.requeueBackOff(q.Clock().Now(), res.Constraint)
	}

	if !forceRequeueBacklogAt.IsZero() {
		// Cap maximum backoff to ensure constraint updates (changing throttle period, etc.) are reflected reasonably quickly
		if forceRequeueBacklogAt.Sub(q.Clock().Now()) > BacklogForceRequeueMaxBackoff {
			forceRequeueBacklogAt = q.Clock().Now().Add(BacklogForceRequeueMaxBackoff)
		}

		if err := q.primaryQueueShard.BacklogRequeue(ctx, backlog, shadowPart, forceRequeueBacklogAt); err != nil && !errors.Is(err, ErrBacklogNotFound) {
			return nil, false, fmt.Errorf("could not requeue backlog: %w", err)
		}
	}

	remainingItems := res.TotalBacklogCount - res.Refilled
	fullyProcessedBacklog := remainingItems == 0
	return res, fullyProcessedBacklog, nil
}

func (q *queueProcessor) scanShadowPartitions(ctx context.Context, until time.Time, qspc chan shadowPartitionChanMsg) error {
	// check whether continuations are enabled and apply chance of skipping continuations in this iteration
	if err := q.scanShadowContinuations(ctx); err != nil {
		return fmt.Errorf("error scanning shadow continuations: %w", err)
	}

	shouldScanAccount := q.runMode.AccountShadowPartition && rand.Intn(100) <= q.runMode.AccountShadowPartitionWeight
	if len(q.runMode.ExclusiveAccounts) > 0 {
		shouldScanAccount = true
	}

	if shouldScanAccount {
		sequential := false

		var peekedAccounts []uuid.UUID
		if len(q.runMode.ExclusiveAccounts) > 0 {
			peekedAccounts = q.runMode.ExclusiveAccounts
		} else {
			peeked, err := Duration(ctx, q.primaryQueueShard.Name(), durOpGobalShadowPartitionAccountPeek, q.Clock().Now(), func(ctx context.Context) ([]uuid.UUID, error) {
				return q.primaryQueueShard.PeekGlobalShadowPartitionAccounts(ctx, sequential, until, ShadowPartitionAccountPeekMax)
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

				parts, err := q.primaryQueueShard.PeekShadowPartitions(ctx, &account, sequential, accountPartitionPeekMax, until)
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

	sequential := false
	parts, err := q.primaryQueueShard.PeekShadowPartitions(ctx, nil, sequential, ShadowPartitionPeekMax, until)
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
