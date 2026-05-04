package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
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
	ItemsToRefill      []string
	ItemCapacityLeases []CapacityLease
	LimitingConstraint enums.QueueConstraint

	RetryAfter time.Time
}

// enqueuedAtTime returns the wall-clock enqueue time for a queue item, or the
// zero value when the item predates the EnqueuedAt schema field. Returning
// zero is important: the constraint cache treats a zero RequestTime as
// unanchored and falls back to its default behavior, so legacy items behave
// the same as before.
func enqueuedAtTime(item *QueueItem) time.Time {
	if item == nil || item.EnqueuedAt == 0 {
		return time.Time{}
	}
	return time.UnixMilli(item.EnqueuedAt)
}

// earliestEnqueuedAt returns the oldest EnqueuedAt across the items, used as
// RequestTime on batched Acquire calls. Items already waiting in the queue
// when the constraint cache populated an "exhausted" entry must bypass that
// stale entry once capacity is freed; the cache compares cache addedAt
// against RequestTime and only bypasses when RequestTime is older.
func earliestEnqueuedAt(items []*QueueItem) time.Time {
	var earliest time.Time
	for _, item := range items {
		t := enqueuedAtTime(item)
		if t.IsZero() {
			continue
		}
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}
	return earliest
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

		// Semaphore
		case
			c.Kind == constraintapi.ConstraintKindSemaphore:
			constraint = enums.QueueConstraintSemaphore
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

	if q.CapacityManager == nil {
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

	if !q.AcquireCapacityLeaseOnBacklogRefill {
		metrics.IncrBacklogRefillConstraintCheckCounter(ctx, enums.BacklogRefillConstraintCheckReasonAcquireOnRefillDisabled.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return &BacklogRefillConstraintCheckResult{
			ItemsToRefill: itemIDs,
		}, nil
	}

	// Build constraint items and config.
	// NOTE: Semaphores are NOT included in the batch Acquire... they come directly
	// from queue items.
	constraintsToCheck := constraintItemsFromBacklog(backlog, constraints)
	if len(constraintsToCheck) == 0 {
		return &BacklogRefillConstraintCheckResult{
			ItemsToRefill: itemIDs,
		}, nil
	}

	var appID uuid.UUID
	if len(items) > 0 {
		appID = items[0].Data.Identifier.AppID
	}

	res, err := q.CapacityManager.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID: *shadowPart.AccountID,
		EnvID:     *shadowPart.EnvID,
		// TODO: Make appID available to backlog
		AppID:                appID,
		IdempotencyKey:       operationIdempotencyKey,
		FunctionID:           *shadowPart.FunctionID,
		CurrentTime:          now,
		RequestTime:          earliestEnqueuedAt(items),
		Duration:             QueueLeaseDuration,
		Constraints:          constraintsToCheck,
		Configuration:        ConstraintConfigFromConstraints(constraints),
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
			LeaseID:    l.LeaseID,
			IssuedAtMS: now.UnixMilli(),
		}
	}

	return &BacklogRefillConstraintCheckResult{
		ItemsToRefill:      itemsToRefill,
		ItemCapacityLeases: itemCapacityLeases,
		LimitingConstraint: constraint,
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
		return ItemLeaseConstraintCheckResult{}, nil
	}

	if shadowPart.AccountID == nil ||
		shadowPart.EnvID == nil ||
		shadowPart.FunctionID == nil {
		metrics.IncrQueueItemConstraintCheckCounter(ctx, enums.QueueItemConstraintReasonIdNil.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return ItemLeaseConstraintCheckResult{}, nil
	}

	if q.CapacityManager == nil {
		metrics.IncrQueueItemConstraintCheckCounter(ctx, enums.QueueItemConstraintReasonConstraintAPIUninitialized.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return ItemLeaseConstraintCheckResult{}, nil
	}

	ctx, span := q.Options().ConditionalTracer.NewSpan(ctx, "queue.ItemLeaseConstraintCheck", *shadowPart.AccountID, *shadowPart.EnvID, *shadowPart.FunctionID)
	defer span.End()

	idempotencyKey := item.ID

	var (
		// a container for constraints that we'll check
		constraintItems []constraintapi.ConstraintItem
		config          = constraintapi.ConstraintConfig{
			FunctionVersion: constraints.FunctionVersion,
		}
	)

	switch hasValidCapacityLease(item, now) {
	case true:
		// in this case, key queues claimed a bunch of constraints up front and we already have some
		// capacity claimed.
		//
		// run some metrics.
		ttl := item.CapacityLease.LeaseID.Timestamp().Sub(now)
		metrics.HistogramConstraintAPIQueueItemLeaseTTL(ctx, ttl, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"kq": item.RefilledAt != 0,
			},
		})

		if len(item.Data.Semaphores) == 0 {
			// backlog lease covers everything, no semaphores — skip Acquire entirely.
			span.SetAttributes(attribute.Bool("valid_lease", true))
			return ItemLeaseConstraintCheckResult{
				CapacityLease: item.CapacityLease,
			}, nil
		}
	case false:
		// release expired/near-expiring lease in the background so that capacity
		// is freed promptly.  this is safe to call even if the lease scavenger has
		// already released this lease... the release Lua script checks whether the
		// lease details still exist as an idempotency key, so both ops do not
		// conflict.  see release.lua L48-54.
		if item.CapacityLease != nil {
			expiredLease := item.CapacityLease
			service.Go(func() {
				_, err := q.CapacityManager.Release(context.Background(), &constraintapi.CapacityReleaseRequest{
					AccountID:      *shadowPart.AccountID,
					IdempotencyKey: idempotencyKey,
					LeaseID:        expiredLease.LeaseID,
					Source: constraintapi.LeaseSource{
						Location:          constraintapi.CallerLocationItemLease,
						Service:           constraintapi.ServiceExecutor,
						RunProcessingMode: constraintapi.RunProcessingModeBackground,
					},
					LeaseIssuedAt: time.UnixMilli(expiredLease.IssuedAtMS),
				})
				if err != nil {
					l.ReportError(err, "failed to release expired capacity", logger.WithErrorReportTags(map[string]string{
						"account_id":  shadowPart.AccountID.String(),
						"lease_id":    expiredLease.LeaseID.String(),
						"function_id": shadowPart.FunctionID.String(),
					}))
				}
			})
		}

		// claim everything from the constraint api: concurrency, keys, throttles, etc.
		constraintItems = constraintItemsFromBacklog(backlog, constraints)
		config = ConstraintConfigFromConstraints(constraints)
	}

	// always add semaphores to each check, as this must be done per queue item.
	for _, sem := range item.Data.Semaphores {
		constraintItems = append(constraintItems, constraintapi.ConstraintItem{
			Kind: constraintapi.ConstraintKindSemaphore,
			Semaphore: &constraintapi.SemaphoreConstraint{
				ID:         sem.ID,
				UsageValue: sem.UsageValue,
				Weight:     sem.Weight,
				Release:    sem.Release,
			},
		})
		config.Semaphores = append(config.Semaphores, sem)
	}

	if len(constraintItems) == 0 {
		return ItemLeaseConstraintCheckResult{}, nil
	}

	res, err := q.CapacityManager.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID: *shadowPart.AccountID,
		EnvID:     *shadowPart.EnvID,
		AppID:     item.Data.Identifier.AppID,
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
		RequestTime:     enqueuedAtTime(item),
		Duration:        QueueLeaseDuration,
		Constraints:     constraintItems,
		Configuration:   config,
		Amount:          1,
		MaximumLifetime: consts.MaxFunctionTimeout + 30*time.Minute,
		Source: constraintapi.LeaseSource{
			Service:           constraintapi.ServiceExecutor,
			Location:          constraintapi.CallerLocationItemLease,
			RunProcessingMode: constraintapi.RunProcessingModeBackground,
		},
	})
	if err != nil {
		span.RecordError(err)
		l.Error("acquiring capacity lease failed", "err", err, "method", "itemLeaseConstraintCheck", "constraints", constraints, "item", item, "function_id", *shadowPart.FunctionID)
		metrics.IncrQueueItemConstraintCheckCounter(ctx, enums.QueueItemConstraintReasonConstraintAPIError.String(), metrics.CounterOpt{
			PkgName: pkgName,
		})
		return ItemLeaseConstraintCheckResult{}, fmt.Errorf("could not enforce constraints and acquire lease: %w", err)
	}

	// Attach entire response
	if span.IsRecording() {
		resJSON, _ := json.Marshal(res)
		span.SetAttributes(attribute.String("acquire_response", string(resJSON)))
	}

	constraint := enums.QueueConstraintNotLimited
	if len(res.ExhaustedConstraints) > 0 {
		constraint = ConvertLimitingConstraint(constraints, res.ExhaustedConstraints)
	}

	span.SetAttributes(attribute.String("limiting_constraint", constraint.String()))

	if len(res.Leases) == 0 {
		span.SetAttributes(attribute.Bool("constrained", true))

		return ItemLeaseConstraintCheckResult{
			LimitingConstraint: constraint,
			RetryAfter:         res.RetryAfter,
		}, nil
	}

	capacityLeaseID := res.Leases[0].LeaseID

	span.SetAttributes(attribute.String("capacity_lease_id", capacityLeaseID.String()))

	return ItemLeaseConstraintCheckResult{
		CapacityLease: &CapacityLease{
			LeaseID:    capacityLeaseID,
			IssuedAtMS: now.UnixMilli(),
		},
	}, nil
}

func hasValidCapacityLease(item *QueueItem, now time.Time) bool {
	return item.CapacityLease != nil && item.CapacityLease.LeaseID.Timestamp().After(now.Add(2*time.Second))
}

func constraintItemsFromBacklog(backlog *QueueBacklog, latestConstraints PartitionConstraintConfig) []constraintapi.ConstraintItem {
	constraints := []constraintapi.ConstraintItem{}

	// Account concurrency
	if latestConstraints.Concurrency.AccountConcurrency != NoConcurrencyLimit {
		constraints = append(constraints,
			constraintapi.ConstraintItem{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeAccount,
				},
			},
		)
	}

	// Function concurrency
	if latestConstraints.Concurrency.FunctionConcurrency != NoConcurrencyLimit {
		constraints = append(constraints,
			constraintapi.ConstraintItem{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		)
	}

	if backlog.Throttle != nil && latestConstraints.Throttle != nil && backlog.Throttle.ThrottleKeyExpressionHash == latestConstraints.Throttle.ThrottleKeyExpressionHash {
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
			var found bool
			for _, cc := range latestConstraints.Concurrency.CustomConcurrencyKeys {
				if cc.Mode == bck.ConcurrencyMode && cc.Scope == bck.Scope && cc.HashedKeyExpression == bck.HashedKeyExpression {
					found = true
					break
				}
			}

			// If custom concurrency key from backlog is not used in latest constraints,
			// do not include in constraint check request
			if !found {
				continue
			}

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
