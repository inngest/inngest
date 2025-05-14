package redis_state

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

type CustomConcurrencyLimit struct {
	Scope enums.ConcurrencyScope `json:"s"`
	Key   string                 `json:"k"`
	Limit int                    `json:"l"`
}

type ShadowPartitionThrottle struct {
	// ThrottleKeyExpressionHash is the hashed throttle key expression, if set.
	ThrottleKeyExpressionHash string `json:"tkh,omitempty"`

	// Limit is the actual rate limit
	Limit int `json:"l"`
	// Burst is the busrsable capacity of the rate limit
	Burst int `json:"b"`
	// Period is the rate limit period, in seconds
	Period int `json:"p"`
}

type QueueShadowPartition struct {
	// PartitionID is the function ID or system queue name. The shadow partition
	// ID is the same as the partition ID used across the queue.
	PartitionID string `json:"id,omitempty"`

	// FunctionVersion represents the current function version represented by this shadow partition.
	// Whenever a newer function version is enqueued, the concurrency keys and limits in here will be adjusted
	// accordingly as part of enqueue_to_backlog().
	// System queues do not have function versions.
	FunctionVersion int `json:"fv"`

	// LeaseID represents a lease on this shadow partition.  If the LeaseID is not nil,
	// this partition can be claimed by a shared-nothing refill worker to work on the
	// backlogs within this shadow partition.
	LeaseID *ulid.ULID `json:"leaseID,omitempty"`

	FunctionID      *uuid.UUID `json:"fid,omitempty"`
	EnvID           *uuid.UUID `json:"eid,omitempty"`
	AccountID       *uuid.UUID `json:"aid,omitempty"`
	SystemQueueName *string    `json:"queueName,omitempty"`

	// SystemConcurrency represents the concurrency limit to apply to system queues. Unset on regular function partitions.
	SystemConcurrency int `json:"sc,omitempty"`

	// AccountConcurrency represents the global account concurrency limit. This is unset on system queues.
	AccountConcurrency int `json:"ac,omitempty"`

	// FunctionConcurrency represents the function concurrency limit.
	FunctionConcurrency int `json:"fc,omitempty"`

	// Up to two custom concurrency keys on user-defined scopes, optionally specifying a key. The key is required
	// on env or account level scopes.
	CustomConcurrencyKeys map[string]CustomConcurrencyLimit `json:"cck,omitempty"`

	// Throttle configuration, optionally specifying key. If no key is set, the throttle value will be the function ID.
	Throttle *ShadowPartitionThrottle `json:"t,omitempty"`

	// Flag to pause refilling to the ready queue.
	PauseRefill bool `json:"norefill,omitempty"`

	// Flag to pause enqueues to the shadow partition.
	PauseEnqueue bool `json:"noenqueue,omitempty"`
}

// BacklogConcurrencyKey represents a custom concurrency key, which can be scoped to the function, environment, or account.
//
// Note: BacklogConcurrencyKey is only used for custom concurrency keys with a defined `key`.
// In the case of configuring concurrency on the function scope without providing a `key`, the default backlog will be used.
type BacklogConcurrencyKey struct {
	ConcurrencyScope enums.ConcurrencyScope `json:"cs,omitempty"`

	// ConcurrencyScopeEntity stores the accountID, envID, or fnID for the respective concurrency scope
	ConcurrencyScopeEntity uuid.UUID `json:"cse,omitempty"`

	// ConcurrencyKey is the hashed concurrency key expression (e.g. hash("event.data.customerId"))
	ConcurrencyKey string `json:"ck,omitempty"`

	// ConcurrencyKeyValue is the hashed evaluated key (e.g. hash("customer1"))
	ConcurrencyKeyValue string `json:"ckv,omitempty"`

	// ConcurrencyKeyUnhashedValue is the unhashed evaluated key (e.g. "customer1")
	// This may be truncated for long values and may only be used for observability and debugging.
	ConcurrencyKeyUnhashedValue string `json:"ckuv,omitempty"`
}

type BacklogThrottle struct {
	// ThrottleKey is the hashed evaluated throttle key (e.g. hash("customer1")) or function ID (e.g. hash(fnID))
	ThrottleKey string `json:"tk,omitempty"`

	// ThrottleKeyRawValue is the unhashed evaluated throttle key (e.g. "customer1") or function ID.
	// This may be truncated for long values and may only be used for observability and debugging.
	ThrottleKeyRawValue string `json:"tkv,omitempty"`

	// ThrottleKeyExpressionHash is the hashed throttle key expression, if set.
	ThrottleKeyExpressionHash string `json:"tkh,omitempty"`
}

type QueueBacklog struct {
	BacklogID         string `json:"id,omitempty"`
	ShadowPartitionID string `json:"sid,omitempty"`

	// Set for backlogs representing custom concurrency keys
	ConcurrencyKeys []BacklogConcurrencyKey `json:"ck,omitempty"`

	// Set for backlogs containing start items only for a given throttle configuration
	Throttle *BacklogThrottle
}

func (q *queue) ItemBacklog(ctx context.Context, i osqueue.QueueItem) QueueBacklog {
	queueName := i.QueueName

	// sanity check: both QueueNames should be set, but sometimes aren't
	if queueName == nil && i.QueueName != nil {
		queueName = i.QueueName
		q.logger.Warn().Interface("item", i).Msg("backlogs encountered queue item with inconsistent custom queue name, should have both i.QueueName and i.Data.QueueName set")
	}

	// sanity check: queueName values must match
	if i.Data.QueueName != nil && i.QueueName != nil && *i.Data.QueueName != *i.QueueName {
		q.logger.Error().Interface("item", i).Msg("backlogs encountered queue item with inconsistent custom queue names, should have matching values for i.QueueName and i.Data.QueueName")
	}

	if queueName != nil {
		// Simply use default backlog for system queues - there shouldn't be any concurrency or throttle keys involved.
		return QueueBacklog{
			ShadowPartitionID: *queueName,
			BacklogID:         fmt.Sprintf("system:%s", *queueName),
		}
	}

	b := QueueBacklog{
		BacklogID:         fmt.Sprintf("fn:%s", i.FunctionID),
		ShadowPartitionID: i.FunctionID.String(),
	}

	// Enqueue start items to throttle backlog if throttle is configured
	if i.Data.Throttle != nil && i.Data.Kind == osqueue.KindStart {
		// This is always specified, even if no key was configured in the function definition.
		// In that case, the Throttle Key is the hashed function ID. See Schedule() for more details.
		b.Throttle = &BacklogThrottle{
			ThrottleKey:               i.Data.Throttle.Key,
			ThrottleKeyExpressionHash: i.Data.Throttle.KeyExpressionHash,
		}

		b.BacklogID += fmt.Sprintf(":t<%s>", i.Data.Throttle.Key)

		if i.Data.Throttle.UnhashedThrottleKey != "" {
			unhashedKey := i.Data.Throttle.UnhashedThrottleKey
			// truncate - just in case
			if len(unhashedKey) > 512 {
				unhashedKey = unhashedKey[:512]
			}
			b.Throttle.ThrottleKeyRawValue = unhashedKey
		}
	}

	concurrencyKeys := i.Data.GetConcurrencyKeys()
	if len(concurrencyKeys) > 0 {
		// NOTE: This is an optimization that ensures we return *updated* concurrency keys
		// for any recently published function configuration.  The embeddeed ckeys from the
		// queue items above may be outdated.
		if q.customConcurrencyLimitRefresher != nil {
			// As an optimization, allow fetching updated concurrency limits if desired.
			updated, _ := duration(ctx, q.primaryQueueShard.Name, "backlog_custom_concurrency_refresher", q.clock.Now(), func(ctx context.Context) ([]state.CustomConcurrency, error) {
				return q.customConcurrencyLimitRefresher(ctx, i), nil
			})
			for _, update := range updated {
				// This is quadratic, but concurrency keys are limited to 2 so it's
				// okay.
				for n, existing := range concurrencyKeys {
					if existing.Key == update.Key {
						concurrencyKeys[n].Limit = update.Limit
					}
				}
			}
		}

		// Create custom concurrency key backlog
		b.ConcurrencyKeys = make([]BacklogConcurrencyKey, len(concurrencyKeys))

		for i, key := range concurrencyKeys {
			scope, entityID, checksum, _ := key.ParseKey()

			b.BacklogID += fmt.Sprintf(":c%d<%s>", i+1, util.XXHash(key.Key))

			b.ConcurrencyKeys[i] = BacklogConcurrencyKey{
				ConcurrencyScope: scope,

				// Account ID, Env ID, or Function ID to apply to the concurrency key to
				ConcurrencyScopeEntity: entityID,

				// Hashed expression to identify which key this is in the shadow partition concurrency key list
				ConcurrencyKey: key.Hash, // hash("event.data.customerID")

				// Evaluated hashed and unhashed values
				ConcurrencyKeyValue:         checksum,                      // hash("customer1")
				ConcurrencyKeyUnhashedValue: key.UnhashedEvaluatedKeyValue, // "customer1"
			}
		}
	}

	return b
}

func (q *queue) ItemShadowPartition(ctx context.Context, i osqueue.QueueItem) QueueShadowPartition {
	var (
		ckeys = i.Data.GetConcurrencyKeys()
	)

	queueName := i.QueueName

	// sanity check: both QueueNames should be set, but sometimes aren't
	if queueName == nil && i.QueueName != nil {
		queueName = i.QueueName
		q.logger.Warn().Interface("item", i).Msg("shadow partitions encountered queue item with inconsistent custom queue name, should have both i.QueueName and i.Data.QueueName set")
	}

	// sanity check: queueName values must match
	if i.Data.QueueName != nil && i.QueueName != nil && *i.Data.QueueName != *i.QueueName {
		q.logger.Error().Interface("item", i).Msg("shadow partitions encountered queue item with inconsistent custom queue names, should have matching values for i.QueueName and i.Data.QueueName")
	}

	// The only case when we manually set a queueName is for system partitions
	if queueName != nil {
		systemPartition := QueuePartition{
			// NOTE: Never remove this. The ID is required to enqueue items to the
			// partition, as it is used for conditional checks in Lua
			ID:        *queueName,
			QueueName: queueName,
		}
		// Fetch most recent system concurrency limit
		systemLimits := q.systemConcurrencyLimitGetter(ctx, systemPartition)
		systemPartition.ConcurrencyLimit = systemLimits.PartitionLimit

		return QueueShadowPartition{
			PartitionID:       *queueName,
			SystemQueueName:   queueName,
			SystemConcurrency: systemLimits.PartitionLimit,
		}
	}

	if i.FunctionID == uuid.Nil {
		q.logger.Error().Interface("item", i).Msg("unexpected missing functionID in ItemPartitions()")
	}

	// NOTE: This is an optimization that ensures we return *updated* concurrency keys
	// for any recently published function configuration.  The embeddeed ckeys from the
	// queue items above may be outdated.
	if q.customConcurrencyLimitRefresher != nil {
		// As an optimization, allow fetching updated concurrency limits if desired.
		updated, _ := duration(ctx, q.primaryQueueShard.Name, "shadow_partition_custom_concurrency_refresher", q.clock.Now(), func(ctx context.Context) ([]state.CustomConcurrency, error) {
			return q.customConcurrencyLimitRefresher(ctx, i), nil
		})
		for _, update := range updated {
			// This is quadratic, but concurrency keys are limited to 2 so it's
			// okay.
			for n, existing := range ckeys {
				if existing.Key == update.Key {
					ckeys[n].Limit = update.Limit
				}
			}
		}
	}

	fnPartition := QueuePartition{
		ID:            i.FunctionID.String(),
		PartitionType: int(enums.PartitionTypeDefault), // Function partition
		FunctionID:    &i.FunctionID,
		AccountID:     i.Data.Identifier.AccountID,
	}

	limits, _ := duration(ctx, q.primaryQueueShard.Name, "shadow_partition_fn_concurrency_getter", q.clock.Now(), func(ctx context.Context) (PartitionConcurrencyLimits, error) {
		return q.concurrencyLimitGetter(ctx, fnPartition), nil
	})

	// The concurrency limit for fns MUST be added for leasing.
	fnPartition.ConcurrencyLimit = limits.FunctionLimit
	if fnPartition.ConcurrencyLimit <= 0 {
		// Use account-level limits, as there are no function level limits
		fnPartition.ConcurrencyLimit = limits.AccountLimit
	}

	var customConcurrencyKeyLimits map[string]CustomConcurrencyLimit
	if len(ckeys) > 0 {
		customConcurrencyKeyLimits = make(map[string]CustomConcurrencyLimit)

		// Up to 2 concurrency keys.
		for _, key := range ckeys {
			scope, _, _, _ := key.ParseKey()

			customConcurrencyKeyLimits[concurrencyKeyID(scope, key.Hash)] = CustomConcurrencyLimit{
				Scope: scope,
				// Key is required to look up the respective limit when checking constraints for a given backlog.
				Key:   key.Hash, // hash(event.data.customerId)
				Limit: key.Limit,
			}
		}
	}

	var throttle *ShadowPartitionThrottle
	if i.Data.Throttle != nil {
		throttle = &ShadowPartitionThrottle{
			ThrottleKeyExpressionHash: i.Data.Throttle.KeyExpressionHash,
			Limit:                     i.Data.Throttle.Limit,
			Burst:                     i.Data.Throttle.Burst,
			Period:                    i.Data.Throttle.Period,
		}
	}

	return QueueShadowPartition{
		PartitionID:     i.FunctionID.String(),
		FunctionVersion: i.Data.Identifier.WorkflowVersion,

		// Identifiers
		FunctionID: &i.FunctionID,
		EnvID:      &i.WorkspaceID,
		AccountID:  &i.Data.Identifier.AccountID,

		// Currently configured limits
		FunctionConcurrency:   fnPartition.ConcurrencyLimit,
		AccountConcurrency:    limits.AccountLimit,
		CustomConcurrencyKeys: customConcurrencyKeyLimits,
		Throttle:              throttle,
	}
}

func (b QueueBacklog) isDefault() bool {
	return b.Throttle == nil && len(b.ConcurrencyKeys) == 0
}

func (b QueueBacklog) isOutdated(sp *QueueShadowPartition) bool {
	// If this is the default backlog, don't normalize.
	// If custom concurrency keys were added, previously-enqueued items
	// in the default backlog do not have custom concurrency keys set.
	if b.isDefault() {
		return false
	}

	// Throttle removed - move items back to default backlog
	if b.Throttle != nil && sp.Throttle == nil {
		return true
	}

	// Throttle key changed - move from old throttle key backlogs to the new throttle key backlogs
	if b.Throttle != nil && sp.Throttle != nil && b.Throttle.ThrottleKeyExpressionHash != sp.Throttle.ThrottleKeyExpressionHash {
		return true
	}

	// Throttle key count does not match
	if len(b.ConcurrencyKeys) != len(sp.CustomConcurrencyKeys) {
		return true
	}

	// All concurrency keys on backlog must be found on partition
	for _, key := range b.ConcurrencyKeys {
		_, ok := sp.CustomConcurrencyKeys[concurrencyKeyID(key.ConcurrencyScope, key.ConcurrencyKey)]
		if !ok {
			return true
		}
	}

	// We don't have to check that all keys on the shadow partition must be found on
	// the backlog as we've compared the length, so the previous check will account for
	// missing/different keys.

	return false
}

func concurrencyKeyID(scope enums.ConcurrencyScope, hash string) string {
	switch scope {
	case enums.ConcurrencyScopeFn:
		return fmt.Sprintf("f:%s", hash)
	case enums.ConcurrencyScopeEnv:
		return fmt.Sprintf("e:%s", hash)
	case enums.ConcurrencyScopeAccount:
		return fmt.Sprintf("a:%s", hash)
	}
	return ""
}
