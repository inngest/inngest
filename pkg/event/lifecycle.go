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

type NoopLifecycleListener struct{}

func (NoopLifecycleListener) OnFunctionMatch(ctx context.Context)      {}
func (NoopLifecycleListener) OnFunctionScheduled(ctx context.Context)  {}
func (NoopLifecycleListener) OnRateLimited(ctx context.Context)        {}
func (NoopLifecycleListener) OnDebounced(ctx context.Context)          {}
func (NoopLifecycleListener) OnBatched(ctx context.Context)            {}
func (NoopLifecycleListener) OnSingletonSkipped(ctx context.Context)   {}
func (NoopLifecycleListener) OnSingletonCancelled(ctx context.Context) {}
func (NoopLifecycleListener) OnRunResumed(ctx context.Context)         {}
func (NoopLifecycleListener) OnRunCancelled(ctx context.Context)       {}
