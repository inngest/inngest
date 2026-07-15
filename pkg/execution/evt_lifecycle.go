package execution

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/debounce"
)

type EventLifecycleListener interface {
	OnFunctionMatch(context.Context)
	OnFunctionScheduled(context.Context)
	OnRateLimited(context.Context)
	OnDebounced(context.Context, *debounce.DebounceItem)
	OnBatched(context.Context)
	OnSingletonSkipped(context.Context)
	OnSingletonCancelled(context.Context)

	// Pause related actions
	OnRunResumed(context.Context)
	OnRunCancelled(context.Context)
}

var _ EventLifecycleListener = (*NoopEventLifecycleListener)(nil)

type NoopEventLifecycleListener struct{}

func (NoopEventLifecycleListener) OnFunctionMatch(ctx context.Context)                        {}
func (NoopEventLifecycleListener) OnFunctionScheduled(ctx context.Context)                    {}
func (NoopEventLifecycleListener) OnRateLimited(ctx context.Context)                          {}
func (NoopEventLifecycleListener) OnDebounced(ctx context.Context, db *debounce.DebounceItem) {}
func (NoopEventLifecycleListener) OnBatched(ctx context.Context)                              {}
func (NoopEventLifecycleListener) OnSingletonSkipped(ctx context.Context)                     {}
func (NoopEventLifecycleListener) OnSingletonCancelled(ctx context.Context)                   {}
func (NoopEventLifecycleListener) OnRunResumed(ctx context.Context)                           {}
func (NoopEventLifecycleListener) OnRunCancelled(ctx context.Context)                         {}
