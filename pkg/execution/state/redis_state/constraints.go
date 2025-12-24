package redis_state

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

func (q *queue) backlogRefillConstraintCheck(
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

	useAPI, fallback := q.UseConstraintAPI(ctx, *shadowPart.AccountID, *shadowPart.EnvID, *shadowPart.FunctionID)
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
		Configuration:        constraintConfigFromConstraints(constraints),
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
		Migration: constraintapi.MigrationIdentifier{
			IsRateLimit: false,
			QueueShard:  q.PrimaryQueueShard.Name(),
		},
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("acquiring capacity lease failed", "err", err, "method", "backlogRefillConstraintCheck", "functionID", *shadowPart.FunctionID)

		if !fallback {
			return nil, fmt.Errorf("could not enforce constraints and acquire lease: %w", err)
		}

		// Attempt to fall back to BacklogRefill -- ignore GCRA with constraint check idempotency
		metrics.IncrBacklogRefillConstraintCheckFallbackCounter(ctx, enums.BacklogRefillConstraintCheckFallbackReasonConstraintAPIError.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return &osqueue.BacklogRefillConstraintCheckResult{
			ItemsToRefill: itemIDs,
		}, nil
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.LimitingConstraints) > 0 {
		constraint = osqueue.ConvertLimitingConstraint(constraints, res.LimitingConstraints)
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
func (q *queue) itemLeaseConstraintCheck(
	ctx context.Context,
	shadowPart *osqueue.QueueShadowPartition,
	backlog *osqueue.QueueBacklog,
	constraints osqueue.PartitionConstraintConfig,
	item *osqueue.QueueItem,
	now time.Time,
	kg QueueKeyGenerator,
) (osqueue.ItemLeaseConstraintCheckResult, error) {
	l := logger.StdlibLogger(ctx)

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

	useAPI, fallback := q.UseConstraintAPI(ctx, *shadowPart.AccountID, *shadowPart.EnvID, *shadowPart.FunctionID)
	if !useAPI {
		metrics.IncrQueueItemConstraintCheckFallbackCounter(ctx, enums.QueueItemConstraintFallbackReasonFeatureFlagDisabled.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return osqueue.ItemLeaseConstraintCheckResult{}, nil
	}

	// If capacity lease is still valid for the forseeable future, use it
	hasValidCapacityLease := item.CapacityLease != nil && item.CapacityLease.LeaseID.Timestamp().After(now.Add(2*time.Second))
	if hasValidCapacityLease {
		return osqueue.ItemLeaseConstraintCheckResult{
			CapacityLease: item.CapacityLease,
			// Skip any constraint checks and subsequent updates,
			// as constraint state is maintained in the Constraint API.
			SkipConstraintChecks: true,
		}, nil
	}

	idempotencyKey := item.ID

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
		Configuration:   constraintConfigFromConstraints(constraints),
		Constraints:     constraintItemsFromBacklog(shadowPart, backlog, kg),
		Amount:          1,
		MaximumLifetime: consts.MaxFunctionTimeout + 30*time.Minute,
		Source: constraintapi.LeaseSource{
			Service:           constraintapi.ServiceExecutor,
			Location:          constraintapi.CallerLocationItemLease,
			RunProcessingMode: constraintapi.RunProcessingModeBackground,
		},
		Migration: constraintapi.MigrationIdentifier{
			IsRateLimit: false,
			QueueShard:  q.PrimaryQueueShard.Name(),
		},
	})
	if err != nil {
		l.Error("acquiring capacity lease failed", "err", err, "method", "itemLeaseConstraintCheck", "constraints", constraints, "item", item)

		if !fallback {
			return osqueue.ItemLeaseConstraintCheckResult{}, fmt.Errorf("could not enforce constraints and acquire lease: %w", err)
		}

		// Fallback to Lease (with idempotency)
		metrics.IncrQueueItemConstraintCheckFallbackCounter(ctx, enums.QueueItemConstraintFallbackReasonConstraintAPIError.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return osqueue.ItemLeaseConstraintCheckResult{}, nil
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.LimitingConstraints) > 0 {
		constraint = osqueue.ConvertLimitingConstraint(constraints, res.LimitingConstraints)
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

func constraintConfigFromConstraints(
	constraints osqueue.PartitionConstraintConfig,
) constraintapi.ConstraintConfig {
	config := constraintapi.ConstraintConfig{
		FunctionVersion: constraints.FunctionVersion,
		Concurrency: constraintapi.ConcurrencyConfig{
			AccountConcurrency:     constraints.Concurrency.AccountConcurrency,
			FunctionConcurrency:    constraints.Concurrency.FunctionConcurrency,
			AccountRunConcurrency:  constraints.Concurrency.AccountRunConcurrency,
			FunctionRunConcurrency: constraints.Concurrency.FunctionRunConcurrency,
		},
	}

	if len(constraints.Concurrency.CustomConcurrencyKeys) > 0 {
		config.Concurrency.CustomConcurrencyKeys = make([]constraintapi.CustomConcurrencyLimit, len(constraints.Concurrency.CustomConcurrencyKeys))

		for i, ccl := range constraints.Concurrency.CustomConcurrencyKeys {
			config.Concurrency.CustomConcurrencyKeys[i] = constraintapi.CustomConcurrencyLimit{
				Mode:              ccl.Mode,
				Scope:             ccl.Scope,
				Limit:             ccl.Limit,
				KeyExpressionHash: ccl.HashedKeyExpression,
			}
		}
	}

	if constraints.Throttle != nil {
		config.Throttle = append(config.Throttle, constraintapi.ThrottleConfig{
			Limit:             constraints.Throttle.Limit,
			Burst:             constraints.Throttle.Burst,
			Period:            constraints.Throttle.Period,
			KeyExpressionHash: constraints.Throttle.ThrottleKeyExpressionHash,
		})
	}

	return config
}

func constraintItemsFromBacklog(sp *osqueue.QueueShadowPartition, backlog *osqueue.QueueBacklog, kg QueueKeyGenerator) []constraintapi.ConstraintItem {
	constraints := []constraintapi.ConstraintItem{
		// Account concurrency (always set)
		{
			Kind: constraintapi.ConstraintKindConcurrency,
			Concurrency: &constraintapi.ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeAccount,
				InProgressItemKey: shadowPartitionAccountInProgressKey(*sp, kg),
			},
		},
		// Function concurrency (always set - falls back to account concurrency)
		{
			Kind: constraintapi.ConstraintKindConcurrency,
			Concurrency: &constraintapi.ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				InProgressItemKey: shadowPartitionInProgressKey(*sp, kg),
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
					InProgressItemKey: backlogConcurrencyKey(bck, kg),
				},
			})
		}
	}

	return constraints
}

func (q *queue) keyConstraintCheckIdempotency(accountID *uuid.UUID, itemID string) string {
	kg := q.RedisClient.kg

	if accountID == nil || *accountID == uuid.Nil {
		return fmt.Sprintf("%s:-", kg.QueuePrefix())
	}

	if q.CapacityManager == nil {
		return fmt.Sprintf("%s:-", kg.QueuePrefix())
	}

	if itemID == "" {
		return fmt.Sprintf("%s:-", kg.QueuePrefix())
	}

	return q.CapacityManager.KeyConstraintCheckIdempotency(constraintapi.MigrationIdentifier{
		IsRateLimit: false,
		QueueShard:  q.PrimaryQueueShard.Name(),
	}, *accountID, itemID)
}
