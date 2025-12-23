package queue

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

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
}

func (sp QueueShadowPartition) Identifier() PartitionIdentifier {
	fnID := uuid.Nil
	if sp.FunctionID != nil {
		fnID = *sp.FunctionID
	}

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	envID := uuid.Nil
	if sp.EnvID != nil {
		envID = *sp.EnvID
	}

	return PartitionIdentifier{
		SystemQueueName: sp.SystemQueueName,

		AccountID:  accountID,
		EnvID:      envID,
		FunctionID: fnID,
	}
}

func (sp QueueShadowPartition) GetAccountID() uuid.UUID {
	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	return accountID
}

func (q *PartitionConstraintConfig) CustomConcurrencyLimit(n int) int {
	if n < 0 || n > len(q.Concurrency.CustomConcurrencyKeys) {
		return 0
	}

	key := q.Concurrency.CustomConcurrencyKeys[n-1]

	return key.Limit
}

// DefaultBacklog returns the default "start" or "continue" backlog for a shadow partition.
//
// This is the backlog items are added to when no keys are configured (or when throttle is configured but we're dealing with non-start items).
//
// This function may return nil if throttle or concurrency keys are configured in the constraints.
func (sp QueueShadowPartition) DefaultBacklog(constraints PartitionConstraintConfig, start bool) *QueueBacklog {
	if sp.SystemQueueName != nil {
		return &QueueBacklog{
			ShadowPartitionID: *sp.SystemQueueName,
			BacklogID:         fmt.Sprintf("system:%s", *sp.SystemQueueName),
		}
	}

	// Function ID should be set for non-system queues
	if sp.FunctionID == nil {
		return nil
	}

	// NOTE: In case custom concurrency keys are configured, we should not use the default
	// function backlog. Instead, all backlogs should include the dynamic key.
	if len(constraints.Concurrency.CustomConcurrencyKeys) > 0 {
		return nil
	}

	// NOTE: In case a start backlog is requested and throttle is used, we should not use
	// the default function backlog. Instead, all backlogs should include the dynamic key.
	if start && constraints.Throttle != nil {
		return nil
	}

	b := &QueueBacklog{
		BacklogID:               fmt.Sprintf("fn:%s", *sp.FunctionID),
		ShadowPartitionID:       sp.FunctionID.String(),
		EarliestFunctionVersion: constraints.FunctionVersion,
		Start:                   start,
	}
	if start {
		b.BacklogID += ":start"
	}

	return b
}

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

func (b QueueBacklog) customConcurrencyKeyID(n int) string {
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

type BacklogRefillResult struct {
	Constraint        enums.QueueConstraint
	Refilled          int
	TotalBacklogCount int
	BacklogCountUntil int
	Capacity          int
	Refill            int
	RefilledItems     []string
	RetryAt           time.Time
}
