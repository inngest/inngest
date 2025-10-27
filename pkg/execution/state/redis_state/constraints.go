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
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

type backlogRefillConstraintCheckResult struct {
	itemsToRefill        []string
	itemCapacityLeases   map[string]ulid.ULID
	limitingConstraint   enums.QueueConstraint
	skipConstraintChecks bool

	fallbackIdempotencyKey string
	retryAfter             time.Time
}

func convertLimitingConstraint(
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
				c.Concurrency.KeyExpressionHash == util.XXHash(""):
			constraint = enums.QueueConstraintAccountConcurrency

		// Function concurrency
		case
			c.Kind == constraintapi.ConstraintKindConcurrency &&
				c.Concurrency.Scope == enums.ConcurrencyScopeFn &&
				c.Concurrency.KeyExpressionHash == util.XXHash(""):
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

func (q *queue) backlogRefillConstraintCheck(
	ctx context.Context,
	shadowPart *QueueShadowPartition,
	backlog *QueueBacklog,
	constraints PartitionConstraintConfig,
	items []*osqueue.QueueItem,
) (*backlogRefillConstraintCheckResult, error) {
	itemIDs := make([]string, len(items))
	for i, item := range items {
		itemIDs[i] = item.ID
	}

	if q.capacityManager == nil || q.useConstraintAPI == nil {
		return &backlogRefillConstraintCheckResult{
			itemsToRefill: itemIDs,
		}, nil
	}

	if shadowPart.AccountID == nil || shadowPart.EnvID == nil || shadowPart.FunctionID == nil {
		return &backlogRefillConstraintCheckResult{
			itemsToRefill: itemIDs,
		}, nil
	}

	// TODO: Extract this
	useAPI, fallback := q.useConstraintAPI(ctx, *shadowPart.AccountID)
	if !useAPI {
		return &backlogRefillConstraintCheckResult{
			itemsToRefill: itemIDs,
		}, nil
	}

	now := q.clock.Now()

	// NOTE: This idempotency key is simply used for retrying Acquire
	// We do not use the same key for multiple processShadowPartitionBacklog attempts
	idempotencyKey := fmt.Sprintf("%s-%d", backlog.BacklogID, now.UnixMilli())

	res, err := q.capacityManager.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID:      *shadowPart.AccountID,
		EnvID:          *shadowPart.EnvID,
		IdempotencyKey: idempotencyKey,
		FunctionID:     *shadowPart.FunctionID,
		CurrentTime:    now,
		Duration:       QueueLeaseDuration,
		// TODO: Build config
		Configuration: constraintapi.ConstraintConfig{},
		// TODO: Supply capacity
		Constraints:          []constraintapi.ConstraintItem{},
		Amount:               len(items),
		LeaseIdempotencyKeys: itemIDs,
		MaximumLifetime:      consts.MaxFunctionTimeout + 30*time.Minute,
		Source: constraintapi.LeaseSource{
			Service:           constraintapi.ServiceExecutor,
			Location:          constraintapi.LeaseLocationItemLease,
			RunProcessingMode: constraintapi.RunProcessingModeBackground,
		},
	})
	if err != nil {
		if !fallback {
			return nil, fmt.Errorf("could not enforce constraints and acquire lease: %w", err)
		}

		// Attempt to fall back to BacklogRefill -- ignore GCRA with fallbackIdempofallbackIdempotencyKey
		return &backlogRefillConstraintCheckResult{
			itemsToRefill:          itemIDs,
			fallbackIdempotencyKey: idempotencyKey,
		}, nil
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.LimitingConstraints) > 0 {
		constraint = convertLimitingConstraint(constraints, res.LimitingConstraints)
	}

	if len(res.Leases) == 0 {
		// TODO Handle no capacity
		return &backlogRefillConstraintCheckResult{
			itemsToRefill:      nil,
			limitingConstraint: constraint,
			retryAfter:         res.RetryAfter,
		}, nil
	}

	itemsToRefill := make([]string, len(res.Leases))
	itemCapacityLeases := make(map[string]ulid.ULID, len(res.Leases))
	for i, l := range res.Leases {
		// NOTE: This works because idempotency key == queue item ID
		itemsToRefill[i] = l.IdempotencyKey
		itemCapacityLeases[l.IdempotencyKey] = l.LeaseID
	}

	return &backlogRefillConstraintCheckResult{
		itemsToRefill:      itemsToRefill,
		itemCapacityLeases: itemCapacityLeases,
		limitingConstraint: constraint,
		// NOTE: We've enforced constraints, so BacklogRefill can skip GCRA, etc.
		skipConstraintChecks: true,
	}, nil
}

type itemLeaseConstraintCheckResult struct {
	leaseID              *ulid.ULID
	limitingConstraint   enums.QueueConstraint
	skipConstraintChecks bool

	fallbackIdempotencyKey string
	retryAfter             time.Time
}

func (q *queue) itemLeaseConstraintCheck(
	ctx context.Context,
	partition QueuePartition,
	constraints PartitionConstraintConfig,
	item *osqueue.QueueItem,
	now time.Time,
) (*itemLeaseConstraintCheckResult, error) {
	if partition.AccountID == uuid.Nil ||
		partition.EnvID == nil ||
		partition.FunctionID == nil {
		return &itemLeaseConstraintCheckResult{}, nil
	}

	if q.capacityManager == nil ||
		q.useConstraintAPI == nil {
		return &itemLeaseConstraintCheckResult{}, nil
	}

	useAPI, fallback := q.useConstraintAPI(ctx, partition.AccountID)
	if !useAPI {
		return &itemLeaseConstraintCheckResult{}, nil
	}

	// If capacity lease is still valid for the forseeable future, use it
	hasValidCapacityLease := item.CapacityLeaseID != nil && item.CapacityLeaseID.Timestamp().Before(now.Add(5*time.Second))
	if hasValidCapacityLease {
		return &itemLeaseConstraintCheckResult{
			leaseID:              item.CapacityLeaseID,
			skipConstraintChecks: true,
		}, nil
	}

	idempotencyKey := item.ID

	res, err := q.capacityManager.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID: partition.AccountID,
		EnvID:     *partition.EnvID,
		// TODO: Double check if the item ID works for idempotency:
		// - Consistent across the same attempt
		// - Do we need to re-evaluate per retry?
		IdempotencyKey: idempotencyKey,
		FunctionID:     *partition.FunctionID,
		CurrentTime:    now,
		Duration:       QueueLeaseDuration,
		// TODO: Build config
		Configuration: constraintapi.ConstraintConfig{},
		// TODO: Supply constraints
		Constraints:     []constraintapi.ConstraintItem{},
		Amount:          1,
		MaximumLifetime: consts.MaxFunctionTimeout + 30*time.Minute,
		Source: constraintapi.LeaseSource{
			Service:           constraintapi.ServiceExecutor,
			Location:          constraintapi.LeaseLocationItemLease,
			RunProcessingMode: constraintapi.RunProcessingModeBackground,
		},
	})
	if err != nil {
		if !fallback {
			return nil, fmt.Errorf("could not enforce constraints and acquire lease: %w", err)
		}

		// Fallback to Lease (with idempotency)
		return &itemLeaseConstraintCheckResult{
			fallbackIdempotencyKey: idempotencyKey,
		}, nil
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.LimitingConstraints) > 0 {
		constraint = convertLimitingConstraint(constraints, res.LimitingConstraints)
	}

	if len(res.Leases) == 0 {
		return &itemLeaseConstraintCheckResult{
			limitingConstraint: constraint,
			retryAfter:         res.RetryAfter,
		}, nil
	}

	capacityLeaseID := res.Leases[0].LeaseID

	return &itemLeaseConstraintCheckResult{
		leaseID:              &capacityLeaseID,
		skipConstraintChecks: true,
	}, nil
}
