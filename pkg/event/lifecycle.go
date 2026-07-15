package event

import "context"

type LifecycleListener interface {
	OnFunctionMatch(context.Context)
	OnFunctionScheduled(context.Context)
	OnRateLimited(context.Context)
	OnDebounced(context.Context)
	OnBatched(context.Context)
	OnSingletonSkipped(context.Context)
	OnSingletonCancelled(context.Context)

	// Pause related actions
	OnRunResumed(context.Context)
	OnRunCancelled(context.Context)
}
