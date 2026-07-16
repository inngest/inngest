package execution

import (
	"context"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
)

// IdempotencySkip describes a matched function that was skipped because the
// idempotency key was already owned by an existing or previous run.
type IdempotencySkip struct {
	// AttemptedRunID is the run ID generated for this scheduling attempt.
	AttemptedRunID ulid.ULID

	// ExistingRunID is set when the executor knows which run owns or previously
	// owned the idempotency key.
	ExistingRunID *ulid.ULID
}

// EventLifecycleListener listens to event-level lifecycle decisions made while
// scheduling or resuming function runs.
type EventLifecycleListener interface {
	OnNoFunctionMatch(context.Context, event.TrackedEvent)

	// OnFunctionMatch is called when an incoming event has matched a function
	// and scheduling is about to be attempted.
	OnFunctionMatch(context.Context, ScheduleRequest)

	// OnFunctionScheduled is called when a new function run is initialized from
	// a matched event or batch.
	//
	// Note that this does not mean the function immediately starts. A function
	// may start if and when there's capacity due to concurrency.
	OnFunctionScheduled(
		context.Context,
		sv2.Metadata,
		queue.Item,
		[]event.TrackedEvent,
	)

	// OnRateLimited is called when a matched function is not scheduled because
	// the function's rate limit was hit.
	OnRateLimited(
		context.Context,
		ScheduleRequest,
	)

	// OnFunctionSkipped is called when a matched function run is skipped before
	// it is scheduled to start.
	OnFunctionSkipped(context.Context, ScheduleRequest, sv2.Metadata, enums.SkipReason)

	// OnFunctionSkippedIdempotency is called when a matched function is skipped
	// because an existing or previous run already owns the idempotency key.
	OnFunctionSkippedIdempotency(context.Context, ScheduleRequest, IdempotencySkip)

	// OnFunctionScheduleFailed is called when scheduling a matched function
	// fails before a terminal scheduling decision can be made.
	OnFunctionScheduleFailed(context.Context, ScheduleRequest, error)

	// OnDebounced is called when a matched function is stored for debounce
	// processing instead of being scheduled immediately.
	OnDebounced(context.Context, ScheduleRequest, debounce.DebounceItem, *ulid.ULID)

	// OnBatched is called when an event is accepted into a batch. The append
	// result describes whether the item created, appended to, deduplicated
	// within, or filled the batch.
	OnBatched(context.Context, batch.BatchItem, ulid.ULID, *batch.BatchAppendResult)

	// OnSingletonCancelled is called when a matched function cancels an existing
	// singleton run before continuing.
	OnSingletonCancelled(context.Context, ScheduleRequest, sv2.ID)

	// OnRunResumed is called when a paused run is resumed from an event, signal,
	// invoke completion, or timeout.
	OnRunResumed(context.Context, sv2.ID, ResumeRequest, enums.Opcode)

	// OnRunCancelled is called when a run is cancelled.
	OnRunCancelled(context.Context, sv2.ID, CancelRequest)
}

var _ EventLifecycleListener = (*NoopEventLifecycleListener)(nil)

// NoopEventLifecycleListener does nothing. This can be embedded into a custom
// implementation allowing other implementations to override specific functions.
type NoopEventLifecycleListener struct{}

func (NoopEventLifecycleListener) OnNoFunctionMatch(ctx context.Context, evt event.TrackedEvent) {}

func (NoopEventLifecycleListener) OnFunctionMatch(ctx context.Context, req ScheduleRequest) {}

func (NoopEventLifecycleListener) OnFunctionScheduled(ctx context.Context, meta sv2.Metadata, qi queue.Item, evts []event.TrackedEvent) {
}

func (NoopEventLifecycleListener) OnRateLimited(ctx context.Context, req ScheduleRequest) {}

func (NoopEventLifecycleListener) OnFunctionSkipped(ctx context.Context, req ScheduleRequest, meta sv2.Metadata, reason enums.SkipReason) {
}

func (NoopEventLifecycleListener) OnFunctionSkippedIdempotency(ctx context.Context, req ScheduleRequest, skip IdempotencySkip) {
}

func (NoopEventLifecycleListener) OnFunctionScheduleFailed(ctx context.Context, req ScheduleRequest, err error) {
}

func (NoopEventLifecycleListener) OnDebounced(ctx context.Context, req ScheduleRequest, db debounce.DebounceItem, debounceID *ulid.ULID) {
}

func (NoopEventLifecycleListener) OnBatched(ctx context.Context, bi batch.BatchItem, batchID ulid.ULID, result *batch.BatchAppendResult) {
}

func (NoopEventLifecycleListener) OnSingletonCancelled(ctx context.Context, req ScheduleRequest, id sv2.ID) {
}

func (NoopEventLifecycleListener) OnRunResumed(ctx context.Context, id sv2.ID, rr ResumeRequest, opcode enums.Opcode) {
}

func (NoopEventLifecycleListener) OnRunCancelled(ctx context.Context, id sv2.ID, cr CancelRequest) {}
