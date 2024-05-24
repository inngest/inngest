package execution

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

// SkipState represents the subset of state.State's data required for OnFunctionSkipped.
type SkipState struct {
	// Reason represents the reason the function was skipped.
	Reason enums.SkipReason

	// CronSchedule, if present, is the cron schedule string that triggered the skipped function.
	CronSchedule *string
}

var _ LifecycleListener = (*NoopLifecyceListener)(nil)

// LifecycleListener listens to lifecycle events on the executor.
type LifecycleListener interface {
	// OnFunctionScheduled is called when a new function is initialized from
	// an event or trigger.
	//
	// Note that this does not mean the function immediately starts.  A function
	// may start if and when there's capacity due to concurrency.
	OnFunctionScheduled(
		context.Context,
		state.Metadata,
		queue.Item,
	)

	// OnFunctionSkipped is called when a function run is skipped.
	// Currently, this happens iff the function is paused.
	OnFunctionSkipped(
		context.Context,
		state.Metadata,
		SkipState,
	)

	// OnFunctionStarted is called when the function starts.  This may be
	// immediately after the function is scheduled, or in the case of increased
	// latency (e.g. due to debouncing or concurrency limits) some time after the
	// function is scheduled.
	OnFunctionStarted(
		context.Context,
		state.Metadata,
		queue.Item,
	)

	// OnFunctionFinished is called when a function finishes.  This will
	// be called when a function completes successfully or permanently failed,
	// with the final driver response indicating the type of success.
	//
	// If failed, DriverResponse will contain a non nil Err string.
	OnFunctionFinished(
		context.Context,
		state.Metadata,
		queue.Item,
		statev1.DriverResponse,
	)

	// OnFunctionCancelled is called when a function is cancelled.  This includes
	// the cancellation request, detailing either the event that cancelled the
	// function or the API request information.
	OnFunctionCancelled(
		context.Context,
		state.Metadata,
		CancelRequest,
	)

	// OnStepScheduled is called when a new step is scheduled.  It contains the
	// queue item which embeds the next step information.
	OnStepScheduled(
		context.Context,
		state.Metadata,
		queue.Item,
		*string, // Step name.
	)

	// OnStepStarted is called when a step begins executing.
	OnStepStarted(
		ctx context.Context,
		md state.Metadata,
		item queue.Item,
		edge inngest.Edge,
		url string,
	)

	// OnStepFinished is called when a step finishes.  This may be
	// a success, a temporary error, or a failure.
	OnStepFinished(
		context.Context,
		state.Metadata,
		queue.Item,
		inngest.Edge,
		inngest.Step,
		statev1.DriverResponse,
	)

	// OnWaitForEvent is called when a wait for event step is scheduled.  The
	// statev1.GeneratorOpcode contains the wait for event details.
	OnWaitForEvent(
		context.Context,
		state.Metadata,
		queue.Item,
		statev1.GeneratorOpcode,
	)

	// OnWaitForEventResumed is called when a function is resumed from waiting for
	// an event.
	OnWaitForEventResumed(
		context.Context,
		state.Metadata,
		ResumeRequest,
		string,
	)

	// OnInvokeFunction is called when a function is invoked from a step.
	OnInvokeFunction(
		context.Context,
		state.Metadata,
		queue.Item,
		statev1.GeneratorOpcode,
		ulid.ULID,
		string,
	)

	// OnInvokeFunctionResumed is called when a function is resumed from an
	// invoke function step. This happens when the invoked function has
	// completed or the step timed out whilst waiting.
	OnInvokeFunctionResumed(
		context.Context,
		state.Metadata,
		ResumeRequest,
		string,
	)

	// OnSleep is called when a sleep step is scheduled.  The
	// statev1.GeneratorOpcode contains the sleep details.
	OnSleep(
		context.Context,
		state.Metadata,
		queue.Item,
		statev1.GeneratorOpcode,
		time.Time, // Sleeping until this time.
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

// OnFunctionScheduled is called when a new function is initialized from
// an event or trigger.
//
// Note that this does not mean the function immediately starts.  A function
// may start if and when there's capacity due to concurrency.
func (NoopLifecyceListener) OnFunctionScheduled(
	context.Context,
	state.Metadata,
	queue.Item,
) {
}

// OnFunctionSkipped is called when a function run is skipped.
func (NoopLifecyceListener) OnFunctionSkipped(
	context.Context,
	state.Metadata,
	SkipState,
) {
}

// OnFunctionStarted is called when the function starts.  This may be
// immediately after the function is scheduled, or in the case of increased
// latency (eg. due to debouncing or concurrency limits) some time after the
// function is scheduled.
func (NoopLifecyceListener) OnFunctionStarted(
	context.Context,
	state.Metadata,
	queue.Item,
) {
}

// OnFunctionFinished is called when a function finishes.  This will
// be called when a function completes successfully or permanently failed,
// with the final driver response indicating the type of success.
//
// If failed, DriverResponse will contain a non nil Err string.
func (NoopLifecyceListener) OnFunctionFinished(
	context.Context,
	state.Metadata,
	queue.Item,
	statev1.DriverResponse,
) {
}

// OnFunctionCancelled is called when a function is cancelled.  This includes
// the cancellation request, detailing either the event that cancelled the
// function or the API request information.
func (NoopLifecyceListener) OnFunctionCancelled(
	context.Context,
	state.Metadata,
	CancelRequest,
) {
}

// OnStepScheduled is called when a new step is scheduled.  It contains the
// queue item which embeds the next step information.
func (NoopLifecyceListener) OnStepScheduled(
	context.Context,
	state.Metadata,
	queue.Item,
	*string, // Step name
) {
}

func (NoopLifecyceListener) OnStepStarted(
	ctx context.Context,
	id state.Metadata,
	item queue.Item,
	edge inngest.Edge,
	url string,
) {
}

func (NoopLifecyceListener) OnStepFinished(
	context.Context,
	state.Metadata,
	queue.Item,
	inngest.Edge,
	inngest.Step,
	statev1.DriverResponse,
) {
}

func (NoopLifecyceListener) OnWaitForEvent(
	context.Context,
	state.Metadata,
	queue.Item,
	statev1.GeneratorOpcode,
) {
}

// OnWaitForEventResumed is called when a function is resumed from waiting for
// an event.
func (NoopLifecyceListener) OnWaitForEventResumed(
	context.Context,
	state.Metadata,
	ResumeRequest,
	string,
) {
}

// OnInvokeFunction is called when a function is invoked from a step.
func (NoopLifecyceListener) OnInvokeFunction(
	context.Context,
	state.Metadata,
	queue.Item,
	statev1.GeneratorOpcode,
	ulid.ULID,
	string,
) {
}

// OnInvokeFunctionResumed is called when a function is resumed from an
// invoke function step. This happens when the invoked function has
// completed or the step timed out whilst waiting.
func (NoopLifecyceListener) OnInvokeFunctionResumed(
	context.Context,
	state.Metadata,
	ResumeRequest,
	string,
) {
}

// OnSleep is called when a sleep step is scheduled.  The
// statev1.GeneratorOpcode contains the sleep details.
func (NoopLifecyceListener) OnSleep(
	context.Context,
	state.Metadata,
	queue.Item,
	statev1.GeneratorOpcode,
	time.Time,
) {
}

func (NoopLifecyceListener) Close() error { return nil }
