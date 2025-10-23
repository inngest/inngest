package redis_state

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
)

type backlogRefillConstraintCheckResult struct {
	itemsToRefill        []string
	itemCapacityLeases   map[string]ulid.ULID
	limitingConstraint   enums.QueueConstraint
	skipConstraintChecks bool

	fallbackIdempotencyKey string
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
	idempotencyKey := fmt.Sprint("%s-%d", backlog.BacklogID, now.UnixMilli())

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

	if len(res.Leases) < len(items) {
		// TODO: Report missing capacity properly
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.LimitingConstraints) > 0 {
		// TODO: Resolve constraint
	}

	if len(res.Leases) == 0 {
		// TODO Handle no capacity

		return &backlogRefillConstraintCheckResult{
			itemsToRefill:      nil,
			limitingConstraint: constraint,
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
