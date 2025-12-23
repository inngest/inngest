package queue

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
)

// ItemBacklog creates a backlog for the given item. The returned backlog may represent current _or_ past
// configurations, in case the queue item has existed for some time and the function was updated in the meantime.
//
// For the sake of consistency and cleanup, ItemBacklog *must* always return the same configuration,
// over the complete lifecycle of a queue item. To this end, the function exclusively retrieves data
// from the queue item, has no side effects, and does not make any calls to external data stores.
func ItemBacklog(ctx context.Context, i QueueItem) QueueBacklog {
	l := logger.StdlibLogger(ctx)
	queueName := i.QueueName

	// sanity check: both QueueNames should be set, but sometimes aren't
	if queueName == nil && i.QueueName != nil {
		queueName = i.QueueName
		l.Warn("backlogs encountered queue item with inconsistent custom queue name, should have both i.QueueName and i.Data.QueueName set", "item", i)
	}

	// sanity check: queueName values must match
	if i.Data.QueueName != nil && i.QueueName != nil && *i.Data.QueueName != *i.QueueName {
		l.Error("backlogs encountered queue item with inconsistent custom queue names, should have matching values for i.QueueName and i.Data.QueueName", "item", i)
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

		// Store earliest function version. Since we do not update backlog metadata,
		// this may be older than the latest items in the backlog.
		EarliestFunctionVersion: i.Data.Identifier.WorkflowVersion,

		// Start items should be moved into their own backlog. This is useful for
		// function run concurrency: To determine how many new runs can start, we can
		// calculate the remaining run capacity and refill as many items from the start backlog.
		Start: i.Data.Kind == KindStart,
	}
	if b.Start {
		b.BacklogID += ":start"
	}

	// Enqueue start items to throttle backlog if throttle is configured
	if i.Data.Throttle != nil && b.Start {
		// This is always specified, even if no key was configured in the function definition.
		// In that case, the Throttle Key is the hashed function ID. See Schedule() for more details.
		b.Throttle = &BacklogThrottle{
			ThrottleKey:               i.Data.Throttle.Key,
			ThrottleKeyExpressionHash: i.Data.Throttle.KeyExpressionHash,
		}

		b.BacklogID += fmt.Sprintf(":t<%s:%s>", i.Data.Throttle.KeyExpressionHash, i.Data.Throttle.Key)

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
		// Create custom concurrency key backlog
		b.ConcurrencyKeys = make([]BacklogConcurrencyKey, len(concurrencyKeys))

		for i, key := range concurrencyKeys {
			scope, entityID, checksum, _ := key.ParseKey()

			b.BacklogID += fmt.Sprintf(":c%d<%s:%s>", i+1, key.Hash, util.XXHash(key.Key))

			b.ConcurrencyKeys[i] = BacklogConcurrencyKey{
				CanonicalKeyID: key.Key,

				Scope: scope,

				// Account ID, Env ID, or Function ID to apply to the concurrency key to
				EntityID: entityID,

				// Hashed expression to identify which key this is in the shadow partition concurrency key list
				HashedKeyExpression: key.Hash, // hash("event.data.customerID")

				// Evaluated hashed and unhashed values
				HashedValue: checksum, // hash("customer1")

				// Just for debugging purposes (only passed on Enqueue after Schedule or backlog normalization)
				UnhashedValue: key.UnhashedEvaluatedKeyValue, // "customer1"
			}
		}
	}

	return b
}

func ItemShadowPartition(ctx context.Context, i QueueItem) QueueShadowPartition {
	l := logger.StdlibLogger(ctx)
	queueName := i.QueueName

	// sanity check: both QueueNames should be set, but sometimes aren't
	if queueName == nil && i.QueueName != nil {
		queueName = i.QueueName
		l.Warn("shadow partitions encountered queue item with inconsistent custom queue name, should have both i.QueueName and i.Data.QueueName set", "item", i)
	}

	// sanity check: queueName values must match
	if i.Data.QueueName != nil && i.QueueName != nil && *i.Data.QueueName != *i.QueueName {
		l.Error("shadow partitions encountered queue item with inconsistent custom queue names, should have matching values for i.QueueName and i.Data.QueueName", "item", i)
	}

	accountID := i.Data.Identifier.AccountID

	// The only case when we manually set a queueName is for system partitions
	if queueName != nil {
		var aID *uuid.UUID
		if accountID != uuid.Nil {
			aID = &accountID
		}

		return QueueShadowPartition{
			PartitionID:     *queueName,
			SystemQueueName: queueName,

			AccountID: aID,
		}
	}

	if accountID == uuid.Nil {
		stack := string(debug.Stack())
		l.Error("unexpected missing accountID in ItemShadowPartition call", "item", i, "stack", stack)
	}

	fnID := i.FunctionID
	if fnID == uuid.Nil {
		stack := string(debug.Stack())
		l.Error("unexpected missing functionID in ItemShadowPartition call", "item", i, "stack", stack)
	}

	return QueueShadowPartition{
		PartitionID:     fnID.String(),
		FunctionVersion: i.Data.Identifier.WorkflowVersion,

		// Identifiers
		FunctionID: &fnID,
		EnvID:      &i.WorkspaceID,
		AccountID:  &accountID,
	}
}
