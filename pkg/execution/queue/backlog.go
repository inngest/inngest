package queue

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
	"gonum.org/v1/gonum/stat/sampleuv"
)

// BacklogConcurrencyKey represents a custom concurrency key, which can be scoped to the function, environment, or account.
//
// Note: BacklogConcurrencyKey is only used for custom concurrency keys with a defined `key`.
// In the case of configuring concurrency on the function scope without providing a `key`, the default backlog will be used.
type BacklogConcurrencyKey struct {
	// CanonicalKeyID is the combined concurrency key (e.g. a:<account ID>:hash("customer1"))
	CanonicalKeyID string `json:"kid"`

	Scope enums.ConcurrencyScope `json:"cs"`

	// EntityID stores the accountID, envID, or fnID for the respective concurrency scope
	EntityID uuid.UUID `json:"cse"`

	// HashedKeyExpression is the hashed concurrency key expression (e.g. hash("event.data.customerId"))
	HashedKeyExpression string `json:"cke"`

	// HashedValue is the hashed concurrency key value (e.g. hash("customer1"))
	HashedValue string `json:"ckv"`

	// UnhashedValue is the unhashed evaluated key (e.g. "customer1")
	// This may be truncated for long values and may only be used for observability and debugging.
	UnhashedValue string `json:"ckuv"`

	// ConcurrencyMode represents the concurrency mode.
	ConcurrencyMode enums.ConcurrencyMode `json:"mode"`
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
	BacklogID               string `json:"id,omitempty"`
	ShadowPartitionID       string `json:"sid,omitempty"`
	EarliestFunctionVersion int    `json:"fv,omitempty"`

	// Start marks backlogs representing items with KindStart.
	Start bool `json:"start,omitempty"`

	// Set for backlogs representing custom concurrency keys
	ConcurrencyKeys []BacklogConcurrencyKey `json:"ck,omitempty"`

	// Set for backlogs containing start items only for a given throttle configuration
	Throttle *BacklogThrottle `json:"t,omitempty"`

	SuccessiveThrottleConstrained          int `json:"stc,omitzero"`
	SuccessiveCustomConcurrencyConstrained int `json:"sccc,omitzero"`
}

func (b QueueBacklog) IsDefault() bool {
	return b.Throttle == nil && len(b.ConcurrencyKeys) == 0
}

func (b QueueBacklog) IsOutdated(constraints PartitionConstraintConfig) enums.QueueNormalizeReason {
	// If the backlog represents newer items than the constraints we're working on,
	// do not attempt to mark the backlog as outdated. Constraints MUST be >= backlog function version at all times.
	if b.EarliestFunctionVersion > 0 && constraints.FunctionVersion > 0 && b.EarliestFunctionVersion > constraints.FunctionVersion {
		return enums.QueueNormalizeReasonUnchanged
	}

	// If this is the default backlog, don't normalize.
	// If custom concurrency keys were added, previously-enqueued items
	// in the default backlog do not have custom concurrency keys set.
	if b.IsDefault() {
		return enums.QueueNormalizeReasonUnchanged
	}

	// Throttle removed - move items back to default backlog
	if b.Throttle != nil && constraints.Throttle == nil {
		return enums.QueueNormalizeReasonThrottleRemoved
	}

	// Throttle key changed - move from old throttle key backlogs to the new throttle key backlogs
	if b.Throttle != nil && constraints.Throttle != nil && b.Throttle.ThrottleKeyExpressionHash != constraints.Throttle.ThrottleKeyExpressionHash {
		return enums.QueueNormalizeReasonThrottleKeyChanged
	}

	// Concurrency key count does not match
	if len(b.ConcurrencyKeys) != len(constraints.Concurrency.CustomConcurrencyKeys) {
		return enums.QueueNormalizeReasonCustomConcurrencyKeyCountMismatch
	}

	// All concurrency keys on backlog must be found on partition
	// This is quadratic but each backlog and shadow partition can only have up to 2 keys, so it's bounded.
	for _, backlogKey := range b.ConcurrencyKeys {
		hasKey := false
		for _, shadowPartitionKey := range constraints.Concurrency.CustomConcurrencyKeys {
			if shadowPartitionKey.Mode == backlogKey.ConcurrencyMode && shadowPartitionKey.Scope == backlogKey.Scope && shadowPartitionKey.HashedKeyExpression == backlogKey.HashedKeyExpression {
				hasKey = true
				break
			}
		}

		if !hasKey {
			return enums.QueueNormalizeReasonCustomConcurrencyKeyNotFoundOnShadowPartition
		}
	}

	// We don't have to check that all keys on the shadow partition must be found on
	// the backlog as we've compared the length, so the previous check will account for
	// missing/different keys.

	return enums.QueueNormalizeReasonUnchanged
}

func (b QueueBacklog) CustomConcurrencyKeyID(n int) string {
	if n < 0 || n > len(b.ConcurrencyKeys) {
		return ""
	}

	key := b.ConcurrencyKeys[n-1]
	return key.CanonicalKeyID
}

func (b QueueBacklog) requeueBackOff(now time.Time, constraint enums.QueueConstraint) time.Time {
	switch constraint {
	case enums.QueueConstraintCustomConcurrencyKey1, enums.QueueConstraintCustomConcurrencyKey2:
		next := time.Duration(b.SuccessiveCustomConcurrencyConstrained) * time.Second

		if next > PartitionConcurrencyLimitRequeueExtension {
			next = PartitionConcurrencyLimitRequeueExtension
		}

		return now.Add(next)
	default:
		return now.Add(BacklogDefaultRequeueExtension)
	}
}

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

	envID := i.Data.Identifier.WorkspaceID
	if envID == uuid.Nil && i.WorkspaceID != uuid.Nil {
		envID = i.WorkspaceID
	}

	fnID := i.Data.Identifier.WorkflowID
	if fnID == uuid.Nil && i.FunctionID != uuid.Nil {
		fnID = i.FunctionID
	}

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

	if fnID == uuid.Nil {
		stack := string(debug.Stack())
		l.Error("unexpected missing functionID in ItemShadowPartition call", "item", i, "stack", stack)
	}

	return QueueShadowPartition{
		PartitionID:     fnID.String(),
		FunctionVersion: i.Data.Identifier.WorkflowVersion,

		// Identifiers
		FunctionID: &fnID,
		EnvID:      &envID,
		AccountID:  &accountID,
	}
}

// ShuffleBacklog returns shuffled backlogs while applying higher weights to non-start backlogs.
//
// NOTE: Applying a higher weight on non-start backlogs is important to ensure queue items to finalize existing functions have a higher likelihood
// of being refilled to the ready queue.
//
// WARN: This only applies to peeked backlogs. Since we apply a random offset while peeking, we may
// omit the default backlog. This is why we add the default backlog in processShadowPartition
func ShuffleBacklogs(b []*QueueBacklog) []*QueueBacklog {
	weights := make([]float64, len(b))
	for i, backlog := range b {
		if backlog.Start {
			weights[i] = 1.0
		} else {
			weights[i] = 10.0
		}
	}

	w := sampleuv.NewWeighted(weights, rnd)
	result := make([]*QueueBacklog, len(b))
	for n := range result {
		idx, ok := w.Take()
		if !ok {
			return b
		}
		result[n] = b[idx]
	}

	return result
}

// backlogRefillMultiplier calculates the backlog specific multiplier to apply when refilling items.
//
// This is required to ensure fairness among backlogs and to guarantee that existing runs finish before new runs are started.
func BacklogRefillMultiplier(backlogs []*QueueBacklog, backlog *QueueBacklog, constraints PartitionConstraintConfig) int {
	switch {
	case backlog.IsDefault() && constraints.Throttle != nil && len(constraints.Concurrency.CustomConcurrencyKeys) == 0:
		// We are attempting to refill items from the default backlog while throttle is configured. This means
		// - we are refilling items to continue or finish existing runs
		// - we want to apply a higher priority
		// - the first backlog is the default function backlog including items to continue existing runs
		// - all following backlogs include start items and represent individual tenants

		// Multiply based on the number of backlogs.
		// Example: If we end up with 100 backlogs, 1 out of 100 is for continuing runs while 99 are starts.
		// Returning len(backlogs) means we apply a multiplier of 100 to the first backlog.
		return len(backlogs)
	default:
		return 1
	}
}
