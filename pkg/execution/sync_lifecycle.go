package execution

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
)

// SyncLifecycleListener listens to the same run/step lifecycle events as
// LifecycleListener, plus event ingestion, but is invoked synchronously
// (inline, not via a spawned goroutine) by the executor and event-ingestion
// path. Implementations MUST NOT block: no I/O, no locking, no flush logic in
// any hook body — a slow implementation here adds latency directly to the
// execution/ingestion critical path. See docs/plans/006-duckdb-poc-subprocess-dual-write.md.
var _ SyncLifecycleListener = (*NoopSyncLifecycleListener)(nil)

type SyncLifecycleListener interface {
	// OnFunctionScheduled is called synchronously when a new function is
	// initialized from an event or trigger.
	OnFunctionScheduled(
		context.Context,
		statev2.Metadata,
		queue.Item,
		[]event.TrackedEvent,
	)

	// OnFunctionStarted is called synchronously when the function starts.
	OnFunctionStarted(
		context.Context,
		statev2.Metadata,
		queue.Item,
		[]json.RawMessage,
	)

	// OnFunctionFinished is called synchronously when a function finishes,
	// successfully or with a permanent failure.
	OnFunctionFinished(
		context.Context,
		statev2.Metadata,
		queue.Item,
		[]json.RawMessage,
		statev1.DriverResponse,
	)

	// OnFunctionCancelled is called synchronously when a function is cancelled.
	OnFunctionCancelled(
		ctx context.Context,
		md statev2.Metadata,
		cr CancelRequest,
		fnEvents []json.RawMessage,
	)

	// OnStepScheduled is called synchronously when a new step is scheduled.
	OnStepScheduled(
		context.Context,
		statev2.Metadata,
		queue.Item,
		*string,
	)

	// OnStepStarted is called synchronously when a step begins executing.
	OnStepStarted(
		ctx context.Context,
		md statev2.Metadata,
		item queue.Item,
		edge inngest.Edge,
		url string,
	)

	// OnStepFinished is called synchronously when a step finishes (success,
	// temporary error, or failure).
	OnStepFinished(
		context.Context,
		statev2.Metadata,
		queue.Item,
		inngest.Edge,
		*statev1.DriverResponse,
		error,
	)

	// OnStepGatewayRequestFinished is called synchronously when a step's
	// offloaded request finishes, successfully or not.
	OnStepGatewayRequestFinished(
		context.Context,
		statev2.Metadata,
		queue.Item,
		inngest.Edge,
		statev1.GeneratorOpcode,
		*http.Response,
		*statev1.UserError,
	)

	// OnSleep is called synchronously when a sleep step is scheduled. The
	// statev1.GeneratorOpcode contains the sleep details.
	OnSleep(
		context.Context,
		statev2.Metadata,
		queue.Item,
		statev1.GeneratorOpcode,
		time.Time, // Sleeping until this time.
	)

	// OnWaitForEvent is called synchronously when a wait for event step is
	// scheduled. The statev1.GeneratorOpcode contains the wait for event
	// details.
	OnWaitForEvent(
		context.Context,
		statev2.Metadata,
		queue.Item,
		statev1.GeneratorOpcode,
		statev1.Pause,
	)

	// OnWaitForEventResumed is called synchronously when a function is
	// resumed from waiting for an event.
	OnWaitForEventResumed(
		context.Context,
		statev2.Metadata,
		statev1.Pause,
		ResumeRequest,
	)

	// OnWaitForSignal is called synchronously when a wait for signal step is
	// scheduled. The statev1.GeneratorOpcode contains the wait for signal
	// details.
	OnWaitForSignal(
		context.Context,
		statev2.Metadata,
		queue.Item,
		statev1.GeneratorOpcode,
		statev1.Pause,
	)

	// OnWaitForSignalResumed is called synchronously when a function is
	// resumed from waiting for a signal.
	OnWaitForSignalResumed(
		context.Context,
		statev2.Metadata,
		statev1.Pause,
		ResumeRequest,
	)

	// OnInvokeFunction is called synchronously when a function is invoked
	// from a step.
	OnInvokeFunction(
		context.Context,
		statev2.Metadata,
		queue.Item,
		statev1.GeneratorOpcode,
		event.Event,
	)

	// OnInvokeFunctionResumed is called synchronously when a function is
	// resumed from an invoke function step, either because the invoked
	// function completed or because the step timed out while waiting.
	OnInvokeFunctionResumed(
		context.Context,
		statev2.Metadata,
		statev1.Pause,
		ResumeRequest,
	)

	// OnEventReceived is called synchronously at the point an event is
	// durably created, regardless of whether (or how many times) it matches
	// a function. There is no equivalent hook on EventLifecycleListener,
	// whose hooks describe per-match scheduling decisions instead.
	OnEventReceived(context.Context, event.TrackedEvent)
}

// NoopSyncLifecycleListener does nothing. Embed this into a custom
// implementation to override only the hooks you need.
type NoopSyncLifecycleListener struct{}

func (NoopSyncLifecycleListener) OnFunctionScheduled(context.Context, statev2.Metadata, queue.Item, []event.TrackedEvent) {
}

func (NoopSyncLifecycleListener) OnFunctionStarted(context.Context, statev2.Metadata, queue.Item, []json.RawMessage) {
}

func (NoopSyncLifecycleListener) OnFunctionFinished(context.Context, statev2.Metadata, queue.Item, []json.RawMessage, statev1.DriverResponse) {
}

func (NoopSyncLifecycleListener) OnFunctionCancelled(context.Context, statev2.Metadata, CancelRequest, []json.RawMessage) {
}

func (NoopSyncLifecycleListener) OnStepScheduled(context.Context, statev2.Metadata, queue.Item, *string) {
}

func (NoopSyncLifecycleListener) OnStepStarted(context.Context, statev2.Metadata, queue.Item, inngest.Edge, string) {
}

func (NoopSyncLifecycleListener) OnStepFinished(context.Context, statev2.Metadata, queue.Item, inngest.Edge, *statev1.DriverResponse, error) {
}

func (NoopSyncLifecycleListener) OnStepGatewayRequestFinished(context.Context, statev2.Metadata, queue.Item, inngest.Edge, statev1.GeneratorOpcode, *http.Response, *statev1.UserError) {
}

func (NoopSyncLifecycleListener) OnSleep(context.Context, statev2.Metadata, queue.Item, statev1.GeneratorOpcode, time.Time) {
}

func (NoopSyncLifecycleListener) OnWaitForEvent(context.Context, statev2.Metadata, queue.Item, statev1.GeneratorOpcode, statev1.Pause) {
}

func (NoopSyncLifecycleListener) OnWaitForEventResumed(context.Context, statev2.Metadata, statev1.Pause, ResumeRequest) {
}

func (NoopSyncLifecycleListener) OnWaitForSignal(context.Context, statev2.Metadata, queue.Item, statev1.GeneratorOpcode, statev1.Pause) {
}

func (NoopSyncLifecycleListener) OnWaitForSignalResumed(context.Context, statev2.Metadata, statev1.Pause, ResumeRequest) {
}

func (NoopSyncLifecycleListener) OnInvokeFunction(context.Context, statev2.Metadata, queue.Item, statev1.GeneratorOpcode, event.Event) {
}

func (NoopSyncLifecycleListener) OnInvokeFunctionResumed(context.Context, statev2.Metadata, statev1.Pause, ResumeRequest) {
}

func (NoopSyncLifecycleListener) OnEventReceived(context.Context, event.TrackedEvent) {}
