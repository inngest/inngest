package execution

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
)

// LifecycleListener listens to lifecycle events on the executor.
type LifecycleListener interface {
	// OnFunctionScheduled is called when a new function is initialized from
	// an event or trigger.
	//
	// Note that this does not mean the function immediately starts.  A function
	// may start if and when there's capacity due to concurrency.
	OnFunctionScheduled(
		context.Context,
		state.Identifier,
		queue.Item,
	)

	// OnFunctionStarted is called when the function starts.  This may be
	// immediately after the function is scheduled, or in the case of increased
	// latency (eg. due to debouncing or concurrency limits) some time after the
	// function is scheduled.
	OnFunctionStarted(
		context.Context,
		state.Identifier,
		queue.Item,
	)

	// OnFunctionFinished is called when a function finishes.  This will
	// be called when a function completes successfully or permanently failed,
	// with the final driver response indicating the type of success.
	//
	// If failed, DriverResponse will contain a non nil Err string.
	OnFunctionFinished(
		context.Context,
		state.Identifier,
		queue.Item,
		state.DriverResponse,
	)

	// OnFunctionCancelled is called when a function is cancelled.  This includes
	// the cancellation request, detailing either the event that cancelled the
	// function or the API request information.
	OnFunctionCancelled(
		context.Context,
		state.Identifier,
		CancelRequest,
	)

	// OnFunctionResumed is called when a function is resumed from waiting for
	// an event.
	OnFunctionResumed(
		context.Context,
		state.Identifier,
		ResumeRequest,
	)

	// OnStepScheduled is called when a new step is scheduled.  It contains the
	// queue item which embeds the next step information.
	OnStepScheduled(
		context.Context,
		state.Identifier,
		queue.Item,
	)

	// OnStepStarted is called when a step begins executing.
	OnStepStarted(
		context.Context,
		state.Identifier,
		queue.Item,
		inngest.Edge,
		inngest.Step,
		state.State,
	)

	// OnStepFinished is called when a step finishes.  This may be
	// a success, a temporary error, or a failure.
	OnStepFinished(
		context.Context,
		state.Identifier,
		queue.Item,
		inngest.Edge,
		inngest.Step,
		state.DriverResponse,
	)

	// OnWaitForEvent is called when a wait for event step is scheduled.  The
	// state.GeneratorOpcode contains the wait for event details.
	OnWaitForEvent(
		context.Context,
		state.Identifier,
		queue.Item,
		state.GeneratorOpcode,
	)

	// OnSleep is called when a sleep step is scheduled.  The
	// state.GeneratorOpcode contains the sleep details.
	OnSleep(
		context.Context,
		state.Identifier,
		queue.Item,
		state.GeneratorOpcode,
	)

	// Close closes the listener and flushes any pending writes.
	//
	// This is backend specific and may be a noop depending on the
	// listener implementation.
	Close() error
}

// NoopLifecyceListener does nothing.  This can be embedded into a custom implementation
// allowing other implementations to override specific functions.
type NoopLifecyceListener struct{}

func (NoopLifecyceListener) OnStepStarted(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	edge inngest.Edge,
	step inngest.Step,
	state state.State,
) {
}

func (NoopLifecyceListener) OnStepFinished(
	context.Context,
	state.Identifier,
	queue.Item,
	inngest.Step,
	*state.DriverResponse,
	error,
) {
}

func (NoopLifecyceListener) OnWaitForEvent(
	context.Context,
	state.Identifier,
	queue.Item,
	state.GeneratorOpcode,
) {
}

func (NoopLifecyceListener) OnFunctionFailed(
	context.Context,
	state.Identifier,
	state.DriverResponse,
) {
}
