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
// execution/ingestion critical path.
var _ SyncLifecycleListener = (*NoopSyncLifecycleListener)(nil)

type SyncLifecycleListener interface {
	// OnFunctionScheduled is called synchronously when a new function is
	// initialized from an event or trigger.
	OnFunctionScheduled(
		context.Context,
		statev2.Metadata,
		queue.Item,
		[]json.RawMessage,
	)

	// OnFunctionStarted is called synchronously when the function starts.
	OnFunctionStarted(
		context.Context,
		statev2.Metadata,
		queue.Item,
		[]json.RawMessage,
	)

	// OnFunctionFinished is called synchronously when a function finishes,
	// successfully or with a permanent failure. now is the caller's own
	// "this just happened" timestamp (e.g. the executor's own clock), so
	// implementations never need to call time.Now() themselves and every
	// timestamp an implementation derives from this call is consistent with
	// the caller's.
	OnFunctionFinished(
		ctx context.Context,
		md statev2.Metadata,
		item queue.Item,
		fnEvents []json.RawMessage,
		resp statev1.DriverResponse,
		now time.Time,
	)

	// OnFunctionCancelled is called synchronously when a function is
	// cancelled. now is the caller's own "this just happened" timestamp —
	// see OnFunctionFinished.
	OnFunctionCancelled(
		ctx context.Context,
		md statev2.Metadata,
		cr CancelRequest,
		fnEvents []json.RawMessage,
		now time.Time,
	)

	// OnStepScheduled is called synchronously when a new step is scheduled.
	// now is the caller's own "this just happened" timestamp — see
	// OnFunctionFinished.
	OnStepScheduled(
		ctx context.Context,
		md statev2.Metadata,
		item queue.Item,
		stepName *string,
		now time.Time,
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
	// temporary error, or failure). now is the caller's own "this just
	// happened" timestamp — see OnFunctionFinished. reqStart is when the
	// request was sent, captured before dispatch — the real boundary for
	// how long this request took, rather than an implementation deriving it
	// by subtracting resp.Duration from now.
	OnStepFinished(
		ctx context.Context,
		md statev2.Metadata,
		item queue.Item,
		edge inngest.Edge,
		resp *statev1.DriverResponse,
		stepErr error,
		reqStart time.Time,
		now time.Time,
	)

	// OnStepRunFinished is called synchronously when a plain step.run step
	// (enums.OpcodeStep/OpcodeStepRun) finishes successfully — the
	// GeneratorOpcode carries the step's own output. Unlike OnStepFinished
	// (called once per SDK request, which may cover several opcodes at
	// once, or none), this fires once per individual step. now is the
	// caller's own "this just happened" timestamp — see OnFunctionFinished.
	OnStepRunFinished(
		ctx context.Context,
		md statev2.Metadata,
		item queue.Item,
		edge inngest.Edge,
		gen statev1.GeneratorOpcode,
		now time.Time,
	)

	// OnStepGatewayRequestFinished is called synchronously when a step's
	// offloaded request finishes, successfully or not. now is the caller's
	// own "this just happened" timestamp — see OnFunctionFinished.
	OnStepGatewayRequestFinished(
		ctx context.Context,
		md statev2.Metadata,
		item queue.Item,
		edge inngest.Edge,
		gen statev1.GeneratorOpcode,
		resp *http.Response,
		userErr *statev1.UserError,
		now time.Time,
	)

	// OnSleep is called synchronously when a sleep step is scheduled. The
	// statev1.GeneratorOpcode contains the sleep details. now is the
	// caller's own "this just happened" timestamp — see OnFunctionFinished
	// — distinct from until, the time the step is sleeping to.
	OnSleep(
		ctx context.Context,
		md statev2.Metadata,
		item queue.Item,
		gen statev1.GeneratorOpcode,
		until time.Time, // Sleeping until this time.
		now time.Time,
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
	// resumed from waiting for an event. now is the caller's own "this just
	// happened" timestamp — see OnFunctionFinished.
	OnWaitForEventResumed(
		context.Context,
		statev2.Metadata,
		statev1.Pause,
		ResumeRequest,
		time.Time,
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
	// resumed from waiting for a signal. now is the caller's own "this just
	// happened" timestamp — see OnFunctionFinished.
	OnWaitForSignalResumed(
		context.Context,
		statev2.Metadata,
		statev1.Pause,
		ResumeRequest,
		time.Time,
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
	// function completed or because the step timed out while waiting. now
	// is the caller's own "this just happened" timestamp — see
	// OnFunctionFinished.
	OnInvokeFunctionResumed(
		context.Context,
		statev2.Metadata,
		statev1.Pause,
		ResumeRequest,
		time.Time,
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

func (NoopSyncLifecycleListener) OnFunctionScheduled(context.Context, statev2.Metadata, queue.Item, []json.RawMessage) {
}

func (NoopSyncLifecycleListener) OnFunctionStarted(context.Context, statev2.Metadata, queue.Item, []json.RawMessage) {
}

func (NoopSyncLifecycleListener) OnFunctionFinished(context.Context, statev2.Metadata, queue.Item, []json.RawMessage, statev1.DriverResponse, time.Time) {
}

func (NoopSyncLifecycleListener) OnFunctionCancelled(context.Context, statev2.Metadata, CancelRequest, []json.RawMessage, time.Time) {
}

func (NoopSyncLifecycleListener) OnStepScheduled(context.Context, statev2.Metadata, queue.Item, *string, time.Time) {
}

func (NoopSyncLifecycleListener) OnStepStarted(context.Context, statev2.Metadata, queue.Item, inngest.Edge, string) {
}

func (NoopSyncLifecycleListener) OnStepFinished(context.Context, statev2.Metadata, queue.Item, inngest.Edge, *statev1.DriverResponse, error, time.Time, time.Time) {
}

func (NoopSyncLifecycleListener) OnStepRunFinished(context.Context, statev2.Metadata, queue.Item, inngest.Edge, statev1.GeneratorOpcode, time.Time) {
}

func (NoopSyncLifecycleListener) OnStepGatewayRequestFinished(context.Context, statev2.Metadata, queue.Item, inngest.Edge, statev1.GeneratorOpcode, *http.Response, *statev1.UserError, time.Time) {
}

func (NoopSyncLifecycleListener) OnSleep(context.Context, statev2.Metadata, queue.Item, statev1.GeneratorOpcode, time.Time, time.Time) {
}

func (NoopSyncLifecycleListener) OnWaitForEvent(context.Context, statev2.Metadata, queue.Item, statev1.GeneratorOpcode, statev1.Pause) {
}

func (NoopSyncLifecycleListener) OnWaitForEventResumed(context.Context, statev2.Metadata, statev1.Pause, ResumeRequest, time.Time) {
}

func (NoopSyncLifecycleListener) OnWaitForSignal(context.Context, statev2.Metadata, queue.Item, statev1.GeneratorOpcode, statev1.Pause) {
}

func (NoopSyncLifecycleListener) OnWaitForSignalResumed(context.Context, statev2.Metadata, statev1.Pause, ResumeRequest, time.Time) {
}

func (NoopSyncLifecycleListener) OnInvokeFunction(context.Context, statev2.Metadata, queue.Item, statev1.GeneratorOpcode, event.Event) {
}

func (NoopSyncLifecycleListener) OnInvokeFunctionResumed(context.Context, statev2.Metadata, statev1.Pause, ResumeRequest, time.Time) {
}

func (NoopSyncLifecycleListener) OnEventReceived(context.Context, event.TrackedEvent) {}
