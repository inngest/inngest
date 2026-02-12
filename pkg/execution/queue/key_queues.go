package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
)

// NOTE: there's no logic behind this number, it's just a random pick for now
var ThrottleBackoffMultiplierThreshold = 15 * time.Second

var (
	ErrBacklogNotFound = fmt.Errorf("backlog not found")

	ErrBacklogPeekMaxExceedsLimits = fmt.Errorf("backlog peek exceeded the maximum limit")

	ErrBacklogGarbageCollected = fmt.Errorf("backlog was garbage-collected")
)

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

func (o QueueOptions) ItemEnableKeyQueues(ctx context.Context, item QueueItem) bool {
	isSystem := item.QueueName != nil || item.Data.QueueName != nil
	if isSystem {
		return false
	}

	if item.Data.Identifier.AccountID != uuid.Nil && o.AllowKeyQueues != nil {
		return o.AllowKeyQueues(ctx, item.Data.Identifier.AccountID, item.Data.Identifier.WorkspaceID, item.FunctionID)
	}

	return false
}
