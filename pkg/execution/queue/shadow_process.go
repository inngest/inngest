package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

func (q *queueProcessor) processShadowPartition(ctx context.Context, shadowPart *osqueue.QueueShadowPartition, continuationCount uint) error {
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

	// Always shuffle backlogs while prioritizing non-start backlogs.
	// This is necessary to ensure we refill items to finish existing runs before
	// refilling run starts.
	backlogs = shuffleBacklogs(backlogs)

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
		multiplier := backlogRefillMultiplier(backlogs, backlog, latestConstraints)
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

		// Hint to the executor
		if refilledItems > 0 {
			l.Trace("hinting to executor after refilling items")

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
