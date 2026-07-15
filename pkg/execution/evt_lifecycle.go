package execution

import (
	"context"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

// EventLifecycleListener listens to event-level lifecycle decisions made while
// scheduling or resuming function runs.
type EventLifecycleListener interface {
	// OnFunctionMatch is called when an incoming event has matched a function
	// and scheduling is about to be attempted.
	OnFunctionMatch(context.Context)

	// OnFunctionScheduled is called when a new function run is initialized from
	// a matched event or batch.
	//
	// Note that this does not mean the function immediately starts. A function
	// may start if and when there's capacity due to concurrency.
	OnFunctionScheduled(
		context.Context,
		statev2.Metadata,
		queue.Item,
		[]event.TrackedEvent,
	)

	// OnRateLimited is called when a matched function is not scheduled because
	// the function's rate limit was hit.
	OnRateLimited(
		context.Context,
		ScheduleRequest,
	)

	// OnDebounced is called when a matched function is stored for debounce
	// processing instead of being scheduled immediately.
	OnDebounced(context.Context, ScheduleRequest, debounce.DebounceItem)

	// OnBatched is called when an event is accepted into a batch.
	OnBatched(context.Context)

	// OnSingletonSkipped is called when a matched function is skipped because
	// another singleton run already exists.
	OnSingletonSkipped(context.Context)

	// OnSingletonCancelled is called when a matched function cancels an existing
	// singleton run before continuing.
	OnSingletonCancelled(context.Context)

	// OnRunResumed is called when a paused run is resumed from an event, signal,
	// invoke completion, or timeout.
	OnRunResumed(context.Context)

	// OnRunCancelled is called when a run is cancelled.
	OnRunCancelled(context.Context)
}

var _ EventLifecycleListener = (*NoopEventLifecycleListener)(nil)

// NoopEventLifecycleListener does nothing. This can be embedded into a custom
// implementation allowing other implementations to override specific functions.
type NoopEventLifecycleListener struct{}

func (NoopEventLifecycleListener) OnFunctionMatch(ctx context.Context) {}

func (NoopEventLifecycleListener) OnFunctionScheduled(ctx context.Context, meta statev2.Metadata, qi queue.Item, evts []event.TrackedEvent) {
}

func (NoopEventLifecycleListener) OnRateLimited(ctx context.Context, req ScheduleRequest) {}

func (NoopEventLifecycleListener) OnDebounced(ctx context.Context, req ScheduleRequest, db debounce.DebounceItem) {
}

func (NoopEventLifecycleListener) OnBatched(ctx context.Context) {}

func (NoopEventLifecycleListener) OnSingletonSkipped(ctx context.Context) {}

func (NoopEventLifecycleListener) OnSingletonCancelled(ctx context.Context) {}

func (NoopEventLifecycleListener) OnRunResumed(ctx context.Context) {}

func (NoopEventLifecycleListener) OnRunCancelled(ctx context.Context) {}
