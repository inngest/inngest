package history

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"golang.org/x/exp/slog"
)

func NewLifecycleListener(l *slog.Logger, d ...Driver) execution.LifecycleListener {
	if l == nil {
		l = slog.Default()
	}
	return lifecycle{
		log:     l,
		drivers: d,
	}
}

type lifecycle struct {
	log     *slog.Logger
	drivers []Driver
}

func (l lifecycle) Close() error {
	var err error
	for _, d := range l.drivers {
		err = errors.Join(err, d.Close())
	}
	return err
}

// OnFunctionScheduled is called when a new function is initialized from
// an event or trigger.
//
// Note that this does not mean the function immediately starts.  A function
// may start if and when there's capacity due to concurrency.
func (l lifecycle) OnFunctionScheduled(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
) {
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionScheduled.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
	}
	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepFinished", "error", err)
		}
	}
}

// OnFunctionStarted is called when the function starts.  This may be
// immediately after the function is scheduled, or in the case of increased
// latency (eg. due to debouncing or concurrency limits) some time after the
// function is scheduled.
func (l lifecycle) OnFunctionStarted(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
) {
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionStarted.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
	}
	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepFinished", "error", err)
		}
	}
}

// OnFunctionFinished is called when a function finishes.  This will
// be called when a function completes successfully or permanently failed,
// with the final driver response indicating the type of success.
//
// If failed, DriverResponse will contain a non nil Err string.
func (l lifecycle) OnFunctionFinished(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	resp state.DriverResponse,
) {
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionCompleted.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		Result:          result(resp),
	}
	if resp.Err != nil {
		h.Type = enums.HistoryTypeFunctionFailed.String()
	}
	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepFinished", "error", err)
		}
	}
}

// OnFunctionCancelled is called when a function is cancelled.  This includes
// the cancellation request, detailing either the event that cancelled the
// function or the API request information.
func (l lifecycle) OnFunctionCancelled(
	ctx context.Context,
	id state.Identifier,
	req execution.CancelRequest,
) {
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionCompleted.String(),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
	}
	if req.EventID != nil {
		h.CancelEvent = &CancelEvent{
			EventID:    *req.EventID,
			Expression: req.Expression,
		}
	}
	if req.UserID != nil {
		h.CancelUser = &CancelUser{UserID: *req.UserID}
	}
	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepFinished", "error", err)
		}
	}
}

// OnStepScheduled is called when a new step is scheduled.  It contains the
// queue item which embeds the next step information.
func (l lifecycle) OnStepScheduled(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
) {
}

func (l lifecycle) OnStepStarted(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	edge inngest.Edge,
	step inngest.Step,
	state state.State,
) {
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepStarted.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		StepName:        &edge.Incoming,
		StepID:          &edge.Incoming, // TODO: Add step name to edge.
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		URL:             &step.URI,
	}
	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepStarted", "error", err)
		}
	}
}

func (l lifecycle) OnStepFinished(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	edge inngest.Edge,
	step inngest.Step,
	resp state.DriverResponse,
) {
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepCompleted.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		StepName:        &resp.Step.Name,
		StepID:          &edge.Incoming,
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		URL:             &step.URI,
		Result:          result(resp),
	}

	// TODO: CompletedStepCount

	if resp.Err != nil && resp.Retryable() {
		h.Type = enums.HistoryTypeStepErrored.String()
	}
	if resp.Err != nil && !resp.Retryable() {
		h.Type = enums.HistoryTypeStepFailed.String()
	}

	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepFinished", "error", err)
		}
	}
}

func (l lifecycle) OnWaitForEvent(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	op state.GeneratorOpcode,
) {
	opts, _ := op.WaitForEventOpts()
	expires, _ := opts.Expires()
	// nothing right now.
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepSleeping.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		StepName:        &op.Name,
		StepID:          &op.ID,
		WaitForEvent: &WaitForEvent{
			EventName:  opts.Event,
			Expression: opts.If,
			Timeout:    expires,
		},
	}
	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onWaitForEvent", "error", err)
		}
	}
}

// OnWaitForEventResumed is called when a function is resumed from waiting for
// an event.
func (l lifecycle) OnWaitForEventResumed(
	ctx context.Context,
	id state.Identifier,
	req execution.ResumeRequest,
) {
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepStarted.String(),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		WaitResult: &WaitResult{
			EventID: req.EventID,
			Timeout: req.EventID == nil,
		},
	}
	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepStarted", "error", err)
		}
	}
}

// OnSleep is called when a sleep step is scheduled.  The
// state.GeneratorOpcode contains the sleep details.
func (l lifecycle) OnSleep(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	op state.GeneratorOpcode,
	until time.Time,
) {
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepSleeping.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		StepName:        &op.Name,
		StepID:          &op.ID,
		Sleep: &Sleep{
			Until: until,
		},
	}
	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepStarted", "error", err)
		}
	}
}

func result(resp state.DriverResponse) *Result {
	return &Result{
		Output:     resp.Output,
		DurationMS: int(resp.Duration.Milliseconds()),
		SizeBytes:  resp.OutputSize,
		// XXX: Add more fields here
	}
}
