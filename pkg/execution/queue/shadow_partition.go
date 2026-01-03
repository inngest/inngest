package queue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

func (sp QueueShadowPartition) KeyQueuesEnabled(ctx context.Context, q *QueueOptions) bool {
	if sp.SystemQueueName != nil {
		return false
	}

	if sp.AccountID == nil || sp.FunctionID == nil || q.AllowKeyQueues == nil {
		return false
	}

	return q.AllowKeyQueues(ctx, *sp.AccountID, *sp.FunctionID)
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
