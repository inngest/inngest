package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

type PartitionConstraintConfig struct {
	FunctionVersion int `json:"fv,omitempty,omitzero"`

	Concurrency PartitionConcurrency `json:"c,omitempty,omitzero"`

	// Throttle configuration, optionally specifying key. If no key is set, the throttle value will be the function ID.
	Throttle *PartitionThrottle `json:"t,omitempty,omitzero"`
}

type CustomConcurrencyLimit struct {
	Mode                enums.ConcurrencyMode  `json:"m"`
	Scope               enums.ConcurrencyScope `json:"s"`
	HashedKeyExpression string                 `json:"k"`
	Limit               int                    `json:"l"`
}

type PartitionThrottle struct {
	// ThrottleKeyExpressionHash is the hashed throttle key expression, if set.
	ThrottleKeyExpressionHash string `json:"tkh,omitempty"`

	// Limit is the actual rate limit
	Limit int `json:"l"`
	// Burst is the busrsable capacity of the rate limit
	Burst int `json:"b"`
	// Period is the rate limit period, in seconds
	Period int `json:"p"`
}

type PartitionConcurrency struct {
	// SystemConcurrency represents the concurrency limit to apply to system queues. Unset on regular function partitions.
	SystemConcurrency int `json:"sc,omitempty"`

	// AccountConcurrency represents the global account concurrency limit. This is unset on system queues.
	AccountConcurrency int `json:"ac,omitempty"`

	// FunctionConcurrency represents the function concurrency limit.
	FunctionConcurrency int `json:"fc,omitempty"`

	// AccountRunConcurrency represents the global account run concurrency limit (how many active runs per account). This is unset on system queues.
	AccountRunConcurrency int `json:"arc,omitempty"`

	// FunctionRunConcurrency represents the function run concurrency limit (how many active runs allowed per function).
	FunctionRunConcurrency int `json:"frc,omitempty"`

	// Up to two custom concurrency keys on user-defined scopes, optionally specifying a key. The key is required
	// on env or account level scopes.
	CustomConcurrencyKeys []CustomConcurrencyLimit `json:"cck,omitempty"`
}

type BacklogRefillConstraintCheckResult struct {
	ItemsToRefill        []string
	ItemCapacityLeases   []CapacityLease
	LimitingConstraint   enums.QueueConstraint
	SkipConstraintChecks bool

	RetryAfter time.Time
}

func ConvertLimitingConstraint(
	constraints PartitionConstraintConfig,
	limitingConstraints []constraintapi.ConstraintItem,
) enums.QueueConstraint {
	constraint := enums.QueueConstraintNotLimited

	for _, c := range limitingConstraints {
		switch {
		// Account concurrency
		case
			c.Kind == constraintapi.ConstraintKindConcurrency &&
				c.Concurrency.Scope == enums.ConcurrencyScopeAccount &&
				c.Concurrency.KeyExpressionHash == "":
			constraint = enums.QueueConstraintAccountConcurrency

		// Function concurrency
		case
			c.Kind == constraintapi.ConstraintKindConcurrency &&
				c.Concurrency.Scope == enums.ConcurrencyScopeFn &&
				c.Concurrency.KeyExpressionHash == "":
			constraint = enums.QueueConstraintFunctionConcurrency

		// Custom concurrency key 1
		case
			len(constraints.Concurrency.CustomConcurrencyKeys) > 0 &&
				c.Kind == constraintapi.ConstraintKindConcurrency &&
				c.Concurrency.Mode == constraints.Concurrency.CustomConcurrencyKeys[0].Mode &&
				c.Concurrency.Scope == constraints.Concurrency.CustomConcurrencyKeys[0].Scope &&
				c.Concurrency.KeyExpressionHash == constraints.Concurrency.CustomConcurrencyKeys[0].HashedKeyExpression:
			constraint = enums.QueueConstraintCustomConcurrencyKey1

		// Custom concurrency key 2
		case
			len(constraints.Concurrency.CustomConcurrencyKeys) > 1 &&
				c.Kind == constraintapi.ConstraintKindConcurrency &&
				c.Concurrency.Mode == constraints.Concurrency.CustomConcurrencyKeys[1].Mode &&
				c.Concurrency.Scope == constraints.Concurrency.CustomConcurrencyKeys[1].Scope &&
				c.Concurrency.KeyExpressionHash == constraints.Concurrency.CustomConcurrencyKeys[1].HashedKeyExpression:
			constraint = enums.QueueConstraintCustomConcurrencyKey2

		// Throttle
		case
			c.Kind == constraintapi.ConstraintKindThrottle:
			constraint = enums.QueueConstraintThrottle
		}
	}

	return constraint
}

type ItemLeaseConstraintCheckResult struct {
	// capacityLease optionally returns a capacity lease ID which
	// must be passed to the processing function to be extended
	// while processing the item.
	CapacityLease *CapacityLease

	// limitingConstraint returns the most limiting constraint in case
	// no capacity was available.
	LimitingConstraint enums.QueueConstraint

	// skipConstraintChecks determines whether subsequent operations
	// should check and enforce constraints, and whether constraint state
	// should be updated while processing a queue item.
	//
	// When enrolled to the Constraint API and holding a valid capacity lease,
	// constraint checks _and_ updates may be skipped, as state is maintained within
	// the Constraint API.
	SkipConstraintChecks bool

	RetryAfter time.Time
}

func ConstraintConfigFromConstraints(
	constraints PartitionConstraintConfig,
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

func (q *queueProcessor) BacklogRefillConstraintCheck(
	ctx context.Context,
	shadowPart *QueueShadowPartition,
	backlog *QueueBacklog,
	constraints PartitionConstraintConfig,
	items []*QueueItem,
	operationIdempotencyKey string,
	now time.Time,
) (*BacklogRefillConstraintCheckResult, error) {
	itemIDs := make([]string, len(items))
	itemRunIDs := make(map[string]ulid.ULID)
	for i, item := range items {
		itemIDs[i] = item.ID
		itemRunIDs[item.ID] = item.Data.Identifier.RunID
	}

	if q.CapacityManager == nil || q.UseConstraintAPI == nil {
		metrics.IncrBacklogRefillConstraintCheckCounter(ctx, enums.BacklogRefillConstraintCheckReasonConstraintAPIUninitialized.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return &BacklogRefillConstraintCheckResult{
			ItemsToRefill: itemIDs,
		}, nil
	}

	if shadowPart.AccountID == nil || shadowPart.EnvID == nil || shadowPart.FunctionID == nil {
		metrics.IncrBacklogRefillConstraintCheckCounter(ctx, enums.BacklogRefillConstraintCheckReasonIDNil.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return &BacklogRefillConstraintCheckResult{
			ItemsToRefill: itemIDs,
		}, nil
	}

	useAPI := q.UseConstraintAPI(ctx, *shadowPart.AccountID, *shadowPart.EnvID, *shadowPart.FunctionID)
	if !useAPI {
		metrics.IncrBacklogRefillConstraintCheckCounter(ctx, enums.BacklogRefillConstraintCheckReasonFeatureFlagDisabled.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return &BacklogRefillConstraintCheckResult{
			ItemsToRefill: itemIDs,
		}, nil
	}

	res, err := q.CapacityManager.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID:            *shadowPart.AccountID,
		EnvID:                *shadowPart.EnvID,
		IdempotencyKey:       operationIdempotencyKey,
		FunctionID:           *shadowPart.FunctionID,
		CurrentTime:          now,
		Duration:             QueueLeaseDuration,
		Configuration:        ConstraintConfigFromConstraints(constraints),
		Constraints:          constraintItemsFromBacklog(shadowPart, backlog),
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
		metrics.IncrBacklogRefillConstraintCheckCounter(ctx, enums.BacklogRefillConstraintCheckReasonConstraintAPIError.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return nil, fmt.Errorf("could not enforce constraints and acquire lease: %w", err)
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.ExhaustedConstraints) > 0 {
		constraint = ConvertLimitingConstraint(constraints, res.ExhaustedConstraints)
	}

	if len(res.Leases) == 0 {
		return &BacklogRefillConstraintCheckResult{
			ItemsToRefill:      nil,
			LimitingConstraint: constraint,
			RetryAfter:         res.RetryAfter,
		}, nil
	}

	itemsToRefill := make([]string, len(res.Leases))
	itemCapacityLeases := make([]CapacityLease, len(res.Leases))
	for i, l := range res.Leases {
		// NOTE: This works because idempotency key == queue item ID
		itemsToRefill[i] = l.IdempotencyKey
		itemCapacityLeases[i] = CapacityLease{
			LeaseID: l.LeaseID,
		}
	}

	return &BacklogRefillConstraintCheckResult{
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
func (q *queueProcessor) ItemLeaseConstraintCheck(
	ctx context.Context,
	shadowPart *QueueShadowPartition,
	backlog *QueueBacklog,
	constraints PartitionConstraintConfig,
	item *QueueItem,
	now time.Time,
) (ItemLeaseConstraintCheckResult, error) {
	l := logger.StdlibLogger(ctx)

	// Disable lease checks for system queues
	// NOTE: This also disables constraint updates during processing, for consistency.
	if shadowPart.SystemQueueName != nil {
		return ItemLeaseConstraintCheckResult{
			SkipConstraintChecks: true,
		}, nil
	}

	if shadowPart.AccountID == nil ||
		shadowPart.EnvID == nil ||
		shadowPart.FunctionID == nil {
		metrics.IncrQueueItemConstraintCheckCounter(ctx, enums.QueueItemConstraintReasonIdNil.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return ItemLeaseConstraintCheckResult{}, nil
	}

	if q.CapacityManager == nil ||
		q.UseConstraintAPI == nil {
		metrics.IncrQueueItemConstraintCheckCounter(ctx, enums.QueueItemConstraintReasonConstraintAPIUninitialized.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return ItemLeaseConstraintCheckResult{}, nil
	}

	useAPI := q.UseConstraintAPI(ctx, *shadowPart.AccountID, *shadowPart.EnvID, *shadowPart.FunctionID)
	if !useAPI {
		metrics.IncrQueueItemConstraintCheckCounter(ctx, enums.QueueItemConstraintReasonFeatureFlagDisabled.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return ItemLeaseConstraintCheckResult{}, nil
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
			return ItemLeaseConstraintCheckResult{
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
		Duration:        QueueLeaseDuration,
		Configuration:   ConstraintConfigFromConstraints(constraints),
		Constraints:     constraintItemsFromBacklog(shadowPart, backlog),
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
		metrics.IncrQueueItemConstraintCheckCounter(ctx, enums.QueueItemConstraintReasonConstraintAPIError.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return ItemLeaseConstraintCheckResult{}, fmt.Errorf("could not enforce constraints and acquire lease: %w", err)
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.ExhaustedConstraints) > 0 {
		constraint = ConvertLimitingConstraint(constraints, res.ExhaustedConstraints)
	}

	if len(res.Leases) == 0 {
		return ItemLeaseConstraintCheckResult{
			LimitingConstraint: constraint,
			RetryAfter:         res.RetryAfter,
		}, nil
	}

	capacityLeaseID := res.Leases[0].LeaseID

	return ItemLeaseConstraintCheckResult{
		CapacityLease: &CapacityLease{
			LeaseID: capacityLeaseID,
		},
		// Skip any constraint checks and subsequent updates,
		// as constraint state is maintained in the Constraint API.
		SkipConstraintChecks: true,
	}, nil
}

func constraintItemsFromBacklog(sp *QueueShadowPartition, backlog *QueueBacklog) []constraintapi.ConstraintItem {
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
