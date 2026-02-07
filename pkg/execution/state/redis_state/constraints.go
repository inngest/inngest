package redis_state

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

func (q *queue) BacklogRefillConstraintCheck(
	ctx context.Context,
	shadowPart *osqueue.QueueShadowPartition,
	backlog *osqueue.QueueBacklog,
	constraints osqueue.PartitionConstraintConfig,
	items []*osqueue.QueueItem,
	operationIdempotencyKey string,
	now time.Time,
) (*osqueue.BacklogRefillConstraintCheckResult, error) {
	kg := q.RedisClient.kg

	itemIDs := make([]string, len(items))
	itemRunIDs := make(map[string]ulid.ULID)
	for i, item := range items {
		itemIDs[i] = item.ID
		itemRunIDs[item.ID] = item.Data.Identifier.RunID
	}

	if q.CapacityManager == nil || q.UseConstraintAPI == nil {
		metrics.IncrBacklogRefillConstraintCheckFallbackCounter(ctx, enums.BacklogRefillConstraintCheckFallbackReasonConstraintAPIUninitialized.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return &osqueue.BacklogRefillConstraintCheckResult{
			ItemsToRefill: itemIDs,
		}, nil
	}

	if shadowPart.AccountID == nil || shadowPart.EnvID == nil || shadowPart.FunctionID == nil {
		metrics.IncrBacklogRefillConstraintCheckFallbackCounter(ctx, enums.BacklogRefillConstraintCheckFallbackReasonIDNil.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return &osqueue.BacklogRefillConstraintCheckResult{
			ItemsToRefill: itemIDs,
		}, nil
	}

	useAPI := q.UseConstraintAPI(ctx, *shadowPart.AccountID, *shadowPart.EnvID, *shadowPart.FunctionID)
	if !useAPI {
		metrics.IncrBacklogRefillConstraintCheckFallbackCounter(ctx, enums.BacklogRefillConstraintCheckFallbackReasonFeatureFlagDisabled.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return &osqueue.BacklogRefillConstraintCheckResult{
			ItemsToRefill: itemIDs,
		}, nil
	}

	res, err := q.CapacityManager.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID:            *shadowPart.AccountID,
		EnvID:                *shadowPart.EnvID,
		IdempotencyKey:       operationIdempotencyKey,
		FunctionID:           *shadowPart.FunctionID,
		CurrentTime:          now,
		Duration:             osqueue.QueueLeaseDuration,
		Configuration:        osqueue.ConstraintConfigFromConstraints(constraints),
		Constraints:          constraintItemsFromBacklog(shadowPart, backlog, kg),
		Amount:               len(items),
		LeaseIdempotencyKeys: itemIDs,
		LeaseRunIDs:          itemRunIDs,
		MaximumLifetime:      consts.MaxFunctionTimeout + 30*time.Minute,
		Source: constraintapi.LeaseSource{
			Service:           constraintapi.ServiceExecutor,
			Location:          constraintapi.CallerLocationBacklogRefill,
			RunProcessingMode: constraintapi.RunProcessingModeBackground,
		},
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("acquiring capacity lease failed", "err", err, "method", "backlogRefillConstraintCheck", "functionID", *shadowPart.FunctionID)
		metrics.IncrBacklogRefillConstraintCheckFallbackCounter(ctx, enums.BacklogRefillConstraintCheckFallbackReasonConstraintAPIError.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return nil, fmt.Errorf("could not enforce constraints and acquire lease: %w", err)
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.ExhaustedConstraints) > 0 {
		constraint = osqueue.ConvertLimitingConstraint(constraints, res.ExhaustedConstraints)
	}

	if len(res.Leases) == 0 {
		return &osqueue.BacklogRefillConstraintCheckResult{
			ItemsToRefill:      nil,
			LimitingConstraint: constraint,
			RetryAfter:         res.RetryAfter,
		}, nil
	}

	itemsToRefill := make([]string, len(res.Leases))
	itemCapacityLeases := make([]osqueue.CapacityLease, len(res.Leases))
	for i, l := range res.Leases {
		// NOTE: This works because idempotency key == queue item ID
		itemsToRefill[i] = l.IdempotencyKey
		itemCapacityLeases[i] = osqueue.CapacityLease{
			LeaseID: l.LeaseID,
		}
	}

	return &osqueue.BacklogRefillConstraintCheckResult{
		ItemsToRefill:      itemsToRefill,
		ItemCapacityLeases: itemCapacityLeases,
		LimitingConstraint: constraint,
		// NOTE: We've enforced constraints, so BacklogRefill can skip GCRA, etc.
		SkipConstraintChecks: true,
	}, nil
}

// itemLeaseConstraintCheck determines whether the given queue item
// can start processing with or without checking and updating constraint state.
//
// If enrolled to the Constraint API, constraint checks will be moved to the new API.
// In case of failing API requests and enabled fallback, we will attempt to check
// constraints in the queue during regular Lease operations (while handling idempotency
// in case the Constraint API call succeeded internally).
//
// In the case of using the Constraint API, the item may also receive a capacity lease to be
// extended for the duration of processing.
func (q *queue) ItemLeaseConstraintCheck(
	ctx context.Context,
	shadowPart *osqueue.QueueShadowPartition,
	backlog *osqueue.QueueBacklog,
	constraints osqueue.PartitionConstraintConfig,
	item *osqueue.QueueItem,
	now time.Time,
) (osqueue.ItemLeaseConstraintCheckResult, error) {
	l := logger.StdlibLogger(ctx)

	kg := q.RedisClient.kg

	// Disable lease checks for system queues
	// NOTE: This also disables constraint updates during processing, for consistency.
	if shadowPart.SystemQueueName != nil {
		return osqueue.ItemLeaseConstraintCheckResult{
			SkipConstraintChecks: true,
		}, nil
	}

	if shadowPart.AccountID == nil ||
		shadowPart.EnvID == nil ||
		shadowPart.FunctionID == nil {
		metrics.IncrQueueItemConstraintCheckFallbackCounter(ctx, enums.QueueItemConstraintFallbackReasonIdNil.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return osqueue.ItemLeaseConstraintCheckResult{}, nil
	}

	if q.CapacityManager == nil ||
		q.UseConstraintAPI == nil {
		metrics.IncrQueueItemConstraintCheckFallbackCounter(ctx, enums.QueueItemConstraintFallbackReasonConstraintAPIUninitialized.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return osqueue.ItemLeaseConstraintCheckResult{}, nil
	}

	useAPI := q.UseConstraintAPI(ctx, *shadowPart.AccountID, *shadowPart.EnvID, *shadowPart.FunctionID)
	if !useAPI {
		metrics.IncrQueueItemConstraintCheckFallbackCounter(ctx, enums.QueueItemConstraintFallbackReasonFeatureFlagDisabled.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return osqueue.ItemLeaseConstraintCheckResult{}, nil
	}

	idempotencyKey := item.ID

	// If capacity lease is still valid for the forseeable future, use it
	if item.CapacityLease != nil {
		expiry := item.CapacityLease.LeaseID.Timestamp()
		hasValidLease := expiry.After(now.Add(2 * time.Second))

		ttl := expiry.Sub(now)
		expired := ttl <= 0

		metrics.HistogramConstraintAPIQueueItemLeaseTTL(ctx, ttl, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"expired": expired,
				"kq":      item.RefilledAt != 0,
			},
		})

		// Lease is still valid, return immediately
		if hasValidLease {
			return osqueue.ItemLeaseConstraintCheckResult{
				CapacityLease: item.CapacityLease,
				// Skip any constraint checks and subsequent updates,
				// as constraint state is maintained in the Constraint API.
				SkipConstraintChecks: true,
			}, nil
		}

		// Lease is invalid or not valid long enough, optimistically return capacity
		// without blocking critical path operations
		service.Go(func() {
			_, err := q.CapacityManager.Release(context.Background(), &constraintapi.CapacityReleaseRequest{
				AccountID:      *shadowPart.AccountID,
				IdempotencyKey: idempotencyKey,
				LeaseID:        item.CapacityLease.LeaseID,
				Source: constraintapi.LeaseSource{
					Location:          constraintapi.CallerLocationItemLease,
					Service:           constraintapi.ServiceExecutor,
					RunProcessingMode: constraintapi.RunProcessingModeBackground,
				},
			})
			if err != nil {
				l.ReportError(err, "failed to release expired capacity", logger.WithErrorReportTags(map[string]string{
					"account_id":  shadowPart.AccountID.String(),
					"lease_id":    item.CapacityLease.LeaseID.String(),
					"function_id": shadowPart.FunctionID.String(),
				}))
			}
		})
	}

	res, err := q.CapacityManager.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID: *shadowPart.AccountID,
		EnvID:     *shadowPart.EnvID,
		// TODO: Double check if the item ID works for idempotency:
		// - Consistent across the same attempt
		// - Do we need to re-evaluate per retry?
		IdempotencyKey:       idempotencyKey,
		LeaseIdempotencyKeys: []string{idempotencyKey},
		LeaseRunIDs: map[string]ulid.ULID{
			idempotencyKey: item.Data.Identifier.RunID,
		},
		FunctionID:      *shadowPart.FunctionID,
		CurrentTime:     now,
		Duration:        osqueue.QueueLeaseDuration,
		Configuration:   osqueue.ConstraintConfigFromConstraints(constraints),
		Constraints:     constraintItemsFromBacklog(shadowPart, backlog, kg),
		Amount:          1,
		MaximumLifetime: consts.MaxFunctionTimeout + 30*time.Minute,
		Source: constraintapi.LeaseSource{
			Service:           constraintapi.ServiceExecutor,
			Location:          constraintapi.CallerLocationItemLease,
			RunProcessingMode: constraintapi.RunProcessingModeBackground,
		},
	})
	if err != nil {
		l.Error("acquiring capacity lease failed", "err", err, "method", "itemLeaseConstraintCheck", "constraints", constraints, "item", item, "function_id", *shadowPart.FunctionID)
		metrics.IncrQueueItemConstraintCheckFallbackCounter(ctx, enums.QueueItemConstraintFallbackReasonConstraintAPIError.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return osqueue.ItemLeaseConstraintCheckResult{}, fmt.Errorf("could not enforce constraints and acquire lease: %w", err)
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.ExhaustedConstraints) > 0 {
		constraint = osqueue.ConvertLimitingConstraint(constraints, res.ExhaustedConstraints)
	}

	if len(res.Leases) == 0 {
		return osqueue.ItemLeaseConstraintCheckResult{
			LimitingConstraint: constraint,
			RetryAfter:         res.RetryAfter,
		}, nil
	}

	capacityLeaseID := res.Leases[0].LeaseID

	return osqueue.ItemLeaseConstraintCheckResult{
		CapacityLease: &osqueue.CapacityLease{
			LeaseID: capacityLeaseID,
		},
		// Skip any constraint checks and subsequent updates,
		// as constraint state is maintained in the Constraint API.
		SkipConstraintChecks: true,
	}, nil
}

func constraintItemsFromBacklog(sp *osqueue.QueueShadowPartition, backlog *osqueue.QueueBacklog, kg QueueKeyGenerator) []constraintapi.ConstraintItem {
	constraints := []constraintapi.ConstraintItem{
		// Account concurrency (always set)
		{
			Kind: constraintapi.ConstraintKindConcurrency,
			Concurrency: &constraintapi.ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeAccount,
			},
		},
		// Function concurrency (always set - falls back to account concurrency)
		{
			Kind: constraintapi.ConstraintKindConcurrency,
			Concurrency: &constraintapi.ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
		},
	}

	if backlog.Throttle != nil {
		constraints = append(constraints, constraintapi.ConstraintItem{
			Kind: constraintapi.ConstraintKindThrottle,
			Throttle: &constraintapi.ThrottleConstraint{
				KeyExpressionHash: backlog.Throttle.ThrottleKeyExpressionHash,
				EvaluatedKeyHash:  backlog.Throttle.ThrottleKey,
			},
		})
	}

	if len(backlog.ConcurrencyKeys) > 0 {
		for _, bck := range backlog.ConcurrencyKeys {
			constraints = append(constraints, constraintapi.ConstraintItem{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Mode:              bck.ConcurrencyMode,
					Scope:             bck.Scope,
					KeyExpressionHash: bck.HashedKeyExpression,
					EvaluatedKeyHash:  bck.HashedValue,
				},
			})
		}
	}

	return constraints
}
