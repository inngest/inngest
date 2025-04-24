package redis_state

import (
	"context"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
)

type CustomConcurrencyLimit struct {
	Scope enums.ConcurrencyScope `json:"s"`
	Key   string                 `json:"k"`
	Limit int                    `json:"l"`
}

type ShadowPartition struct {
	FunctionID      uuid.UUID `json:"fid,omitempty"`
	EnvID           uuid.UUID `json:"eid,omitempty"`
	AccountID       uuid.UUID `json:"aid,omitempty"`
	SystemQueueName *string   `json:"queueName,omitempty"`

	// SystemConcurrency represents the concurrency limit to apply to system queues. Unset on regular function partitions.
	SystemConcurrency int `json:"sc,omitempty"`

	// AccountConcurrency represents the global account concurrency limit. This is unset on system queues.
	AccountConcurrency int `json:"ac,omitempty"`

	// FunctionConcurrency represents the function concurrency limit.
	FunctionConcurrency int `json:"fc,omitempty"`

	// Up to two custom concurrency keys on user-defined scopes, optionally specifying a key. The key is required
	// on env or account level scopes.
	CustomConcurrencyKeys []CustomConcurrencyLimit `json:"cck,omitempty"`

	// Throttle configuration, optionally specifying key. If no key is set, the throttle value will be the function ID.
	Throttle *osqueue.Throttle `json:"t,omitempty"`

	// Flag to pause refilling to the ready queue.
	PauseRefill bool `json:"norefill,omitempty"`

	// Flag to pause enqueues to the shadow partition.
	PauseEnqueue bool `json:"noenqueue,omitempty"`
}

type QueueBacklog struct {
	ConcurrencyScope enums.ConcurrencyScope `json:"cs"`

	ConcurrencyKey         *string `json:"ck,omitempty"`
	ConcurrencyKeyRawValue *string `json:"ckv,omitempty"`

	// Set for backlogs containing start items only for a given throttle configuration
	Throttle            bool    `json:"t,omitempty"`
	ThrottleKey         *string `json:"tk,omitempty"`
	ThrottleKeyRawValue *string `json:"tkv,omitempty"`
}

func (q *queue) ItemBacklogs(ctx context.Context, shard QueueShard, i osqueue.QueueItem) []QueueBacklog {
	backlogs := make([]QueueBacklog, 0)

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
		backlogs = append(backlogs, QueueBacklog{})
		return backlogs
	}

	// Enqueue start items to throttle backlog if throttle is configured
	if i.Data.Throttle != nil && i.Data.Kind == osqueue.KindStart {
		b := QueueBacklog{
			Throttle:    true,
			ThrottleKey: &i.Data.Throttle.Key,
		}

		if i.Data.Throttle.UnhashedThrottleKey != "" {
			unhashedKey := i.Data.Throttle.UnhashedThrottleKey
			// truncate - just in case
			if len(unhashedKey) > 512 {
				unhashedKey = unhashedKey[:512]
			}
			b.ThrottleKeyRawValue = &unhashedKey
		}

		backlogs = append(backlogs, b)
	}

	concurrencyKeys := i.Data.GetConcurrencyKeys()

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

	// Create concurrency key backlogs
	for _, key := range concurrencyKeys {
		scope, _, _, _ := key.ParseKey()

		var rawValue *string
		if key.UnhashedEvaluatedKeyValue != "" {
			rawValue = &key.UnhashedEvaluatedKeyValue
		}

		backlogs = append(backlogs, QueueBacklog{
			ConcurrencyScope:       scope,
			ConcurrencyKey:         &key.Hash,
			ConcurrencyKeyRawValue: rawValue,
		})
	}

	return backlogs
}

func (q *queue) ItemShadowPartition(ctx context.Context, i osqueue.QueueItem) ShadowPartition {
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

		return ShadowPartition{
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

	var customConcurrencyKeyLimits []CustomConcurrencyLimit
	if len(ckeys) > 0 {
		customConcurrencyKeyLimits = make([]CustomConcurrencyLimit, len(ckeys))

		// Up to 2 concurrency keys.
		for j, key := range ckeys {
			scope, _, _, _ := key.ParseKey()

			customConcurrencyKeyLimits[j] = CustomConcurrencyLimit{
				Scope: scope,
				// Key is required to look up the respective limit when checking constraints for a given backlog.
				Key:   key.Hash, // hash(event.data.customerId)
				Limit: key.Limit,
			}
		}
	}

	return ShadowPartition{
		// Identifiers
		FunctionID: i.FunctionID,
		EnvID:      i.WorkspaceID,
		AccountID:  i.Data.Identifier.AccountID,

		// Currently configured limits
		FunctionConcurrency:   fnPartition.ConcurrencyLimit,
		AccountConcurrency:    limits.AccountLimit,
		CustomConcurrencyKeys: customConcurrencyKeyLimits,
		Throttle:              i.Data.Throttle,
	}
}
