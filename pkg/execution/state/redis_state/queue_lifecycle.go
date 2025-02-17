package redis_state

import (
	"context"

	"github.com/google/uuid"
)

// QueueLifecycleListener represents a lifecycle listener for queue-related specifics.
type QueueLifecycleListener interface {
	// OnFnConcurrencyLimitReached is called when a queue item cannot be processed due to
	// its function concurrency limit.
	OnFnConcurrencyLimitReached(ctx context.Context, fnID uuid.UUID)

	// OnCustomKeyConcurrencyLimitReached is called when a queue item cannot be processed due to
	// a custom key concurrency limit.
	OnCustomKeyConcurrencyLimitReached(ctx context.Context, key string)

	// OnAccountConcurrencyLimitReached is called when a queue item cannot be processed due to
	// its account's concurrency limit.
	OnAccountConcurrencyLimitReached(
		ctx context.Context,
		accountID uuid.UUID,
		workspaceID *uuid.UUID,
	)
}

type QueueLifecycleListeners []QueueLifecycleListener

var _ QueueLifecycleListener = QueueLifecycleListeners{}

func (l QueueLifecycleListeners) GoEach(fn func(listener QueueLifecycleListener)) {
	for _, listener := range l {
		go fn(listener)
	}
}

func (l QueueLifecycleListeners) OnFnConcurrencyLimitReached(ctx context.Context, fnID uuid.UUID) {
	l.GoEach(func(listener QueueLifecycleListener) {
		listener.OnFnConcurrencyLimitReached(ctx, fnID)
	})
}

func (l QueueLifecycleListeners) OnAccountConcurrencyLimitReached(
	ctx context.Context,
	acctID uuid.UUID,
	workspaceID *uuid.UUID,
) {
	l.GoEach(func(listener QueueLifecycleListener) {
		listener.OnAccountConcurrencyLimitReached(ctx, acctID, workspaceID)
	})
}

func (l QueueLifecycleListeners) OnCustomKeyConcurrencyLimitReached(ctx context.Context, key string) {
	l.GoEach(func(listener QueueLifecycleListener) {
		listener.OnCustomKeyConcurrencyLimitReached(ctx, key)
	})
}
