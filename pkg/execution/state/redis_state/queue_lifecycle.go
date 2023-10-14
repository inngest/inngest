package redis_state

import (
	"context"

	"github.com/google/uuid"
)

// QueueLifecycleListener represents a lifecycle listener for queue-related specifics.
type QueueLifecycleListener interface {
	// OnConcurrencyLimitReached is called when a queue item cannot be processed due to
	// concurrency constraints.
	//
	// In the future, we should specify which concurrency limit was reached (account,
	// partition, or custom).
	OnConcurrencyLimitReached(ctx context.Context, fnID uuid.UUID) error
}
