package execution

import (
	"context"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

type EventLifecycleListener interface {
	OnFunctionMatch(context.Context)
	OnFunctionScheduled(context.Context, statev2.Metadata,
		queue.Item,
		[]event.TrackedEvent)
	OnRateLimited(context.Context)
	OnDebounced(context.Context, ScheduleRequest, debounce.DebounceItem)
	OnBatched(context.Context)
	OnSingletonSkipped(context.Context)
	OnSingletonCancelled(context.Context)

	// Pause related actions
	OnRunResumed(context.Context)
	OnRunCancelled(context.Context)
}

var _ EventLifecycleListener = (*NoopEventLifecycleListener)(nil)

type NoopEventLifecycleListener struct{}

func (NoopEventLifecycleListener) OnFunctionMatch(ctx context.Context) {}

func (NoopEventLifecycleListener) OnFunctionScheduled(ctx context.Context, meta statev2.Metadata, qi queue.Item, evts []event.TrackedEvent) {
}

func (NoopEventLifecycleListener) OnRateLimited(ctx context.Context) {}

func (NoopEventLifecycleListener) OnDebounced(ctx context.Context, req ScheduleRequest, db debounce.DebounceItem) {
}

func (NoopEventLifecycleListener) OnBatched(ctx context.Context) {}

func (NoopEventLifecycleListener) OnSingletonSkipped(ctx context.Context) {}

func (NoopEventLifecycleListener) OnSingletonCancelled(ctx context.Context) {}

func (NoopEventLifecycleListener) OnRunResumed(ctx context.Context) {}

func (NoopEventLifecycleListener) OnRunCancelled(ctx context.Context) {}
