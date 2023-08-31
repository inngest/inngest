package history

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
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
	groupID, err := toUUID(item.GroupID)
	if err != nil {
		l.log.Error(
			"error parsing group ID",
			"error", err,
			"group_id", item.GroupID,
			"run_id", id.RunID.String(),
		)
	}

	h := History{
		AccountID:       id.AccountID,
		Attempt:         int64(item.Attempt),
		BatchID:         id.BatchID,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionScheduled.String(),
		WorkspaceID:     id.WorkspaceID,
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onFunctionScheduled", "error", err)
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
	groupID, err := toUUID(item.GroupID)
	if err != nil {
		l.log.Error(
			"error parsing group ID",
			"error", err,
			"group_id", item.GroupID,
			"run_id", id.RunID.String(),
		)
	}

	latency, _ := redis_state.GetItemLatency(ctx)
	latencyMS := latency.Milliseconds()

	h := History{
		AccountID:       id.AccountID,
		Attempt:         int64(item.Attempt),
		BatchID:         id.BatchID,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		LatencyMS:       &latencyMS,
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionStarted.String(),
		WorkspaceID:     id.WorkspaceID,
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
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
	groupID, err := toUUID(item.GroupID)
	if err != nil {
		l.log.Error(
			"error parsing group ID",
			"error", err,
			"group_id", item.GroupID,
			"run_id", id.RunID.String(),
		)
	}

	h := History{
		AccountID:       id.AccountID,
		Attempt:         int64(item.Attempt),
		BatchID:         id.BatchID,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		Result:          result(resp),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionCompleted.String(),
		WorkspaceID:     id.WorkspaceID,
	}
	if resp.Err != nil {
		h.Type = enums.HistoryTypeFunctionFailed.String()
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onFunctionFinished", "error", err)
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
	var groupID *uuid.UUID

	h := History{
		AccountID:       id.AccountID,
		BatchID:         id.BatchID,
		Cancel:          &req,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionCancelled.String(),
		WorkspaceID:     id.WorkspaceID,
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onFunctionCancelled", "error", err)
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
	edge, _ := queue.GetEdge(item)
	if edge == nil {
		return
	}

	groupID, err := toUUID(item.GroupID)
	if err != nil {
		l.log.Error(
			"error parsing group ID",
			"error", err,
			"group_id", item.GroupID,
			"run_id", id.RunID.String(),
		)
	}

	h := History{
		AccountID:       id.AccountID,
		Attempt:         int64(item.Attempt),
		BatchID:         id.BatchID,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		RunID:           id.RunID,
		StepID:          &edge.Edge.Incoming, // TODO: Add step name to edge.
		StepName:        &edge.Edge.Incoming,
		Type:            enums.HistoryTypeStepScheduled.String(),
		WorkspaceID:     id.WorkspaceID,
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepScheduled", "error", err)
		}
	}
}

func (l lifecycle) OnStepStarted(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	edge inngest.Edge,
	step inngest.Step,
	state state.State,
) {
	groupID, err := toUUID(item.GroupID)
	if err != nil {
		l.log.Error(
			"error parsing group ID",
			"error", err,
			"group_id", item.GroupID,
			"run_id", id.RunID.String(),
		)
	}

	latency, _ := redis_state.GetItemLatency(ctx)
	latencyMS := latency.Milliseconds()

	h := History{
		AccountID:       id.AccountID,
		Attempt:         int64(item.Attempt),
		BatchID:         id.BatchID,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		LatencyMS:       &latencyMS,
		RunID:           id.RunID,
		StepID:          &edge.Incoming, // TODO: Add step name to edge.
		StepName:        &edge.Incoming,
		Type:            enums.HistoryTypeStepStarted.String(),
		URL:             &step.URI,
		WorkspaceID:     id.WorkspaceID,
	}

	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
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
	groupID, err := toUUID(item.GroupID)
	if err != nil {
		l.log.Error(
			"error parsing group ID",
			"error", err,
			"group_id", item.GroupID,
			"run_id", id.RunID.String(),
		)
	}

	h := History{
		AccountID:       id.AccountID,
		Attempt:         int64(item.Attempt),
		BatchID:         id.BatchID,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		Result:          result(resp),
		RunID:           id.RunID,
		StepID:          &edge.Incoming,
		StepName:        &resp.Step.Name,
		Type:            enums.HistoryTypeStepCompleted.String(),
		URL:             &step.URI,
		WorkspaceID:     id.WorkspaceID,
	}

	// TODO: CompletedStepCount

	if resp.Err != nil && resp.Retryable() {
		h.Type = enums.HistoryTypeStepErrored.String()
	}
	if resp.Err != nil && !resp.Retryable() {
		h.Type = enums.HistoryTypeStepFailed.String()
	}

	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
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
	groupID, err := toUUID(item.GroupID)
	if err != nil {
		l.log.Error(
			"error parsing group ID",
			"error", err,
			"group_id", item.GroupID,
			"run_id", id.RunID.String(),
		)
	}

	opts, _ := op.WaitForEventOpts()
	expires, _ := opts.Expires()
	// nothing right now.
	h := History{
		AccountID:       id.AccountID,
		Attempt:         int64(item.Attempt),
		BatchID:         id.BatchID,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		RunID:           id.RunID,
		StepID:          &op.ID,
		StepName:        &op.Name,
		Type:            enums.HistoryTypeStepSleeping.String(),
		WaitForEvent: &WaitForEvent{
			EventName:  opts.Event,
			Expression: opts.If,
			Timeout:    expires,
		},
		WorkspaceID: id.WorkspaceID,
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
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
	var groupID *uuid.UUID

	h := History{
		AccountID:       id.AccountID,
		BatchID:         id.BatchID,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepStarted.String(),
		WaitResult: &WaitResult{
			EventID: req.EventID,
			Timeout: req.EventID == nil,
		},
		WorkspaceID: id.WorkspaceID,
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onWaitForEventResumed", "error", err)
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
	groupID, err := toUUID(item.GroupID)
	if err != nil {
		l.log.Error(
			"error parsing group ID",
			"error", err,
			"group_id", item.GroupID,
			"run_id", id.RunID.String(),
		)
	}

	h := History{
		AccountID:       id.AccountID,
		Attempt:         int64(item.Attempt),
		BatchID:         id.BatchID,
		CreatedAt:       time.Now(),
		EventID:         id.EventID,
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		IdempotencyKey:  id.IdempotencyKey(),
		RunID:           id.RunID,
		Sleep: &Sleep{
			Until: until,
		},
		StepID:      &op.ID,
		StepName:    &op.Name,
		Type:        enums.HistoryTypeStepSleeping.String(),
		WorkspaceID: id.WorkspaceID,
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onSleep", "error", err)
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

func toUUID(id string) (*uuid.UUID, error) {
	if id == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &parsed, nil

}
