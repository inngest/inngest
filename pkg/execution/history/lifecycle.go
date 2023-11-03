package history

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
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
	s state.State,
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
		Cron:            s.CronSchedule(),
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionScheduled.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
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
	s state.State,
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
		Cron:            s.CronSchedule(),
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		RunID:           id.RunID,
		Type:            enums.HistoryTypeFunctionStarted.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		LatencyMS:       &latencyMS,
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
	s state.State,
) {
	completedStepCount := int64(len(s.Actions()) + len(s.Errors()))

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
		Cron:               s.CronSchedule(),
		ID:                 ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:          id.AccountID,
		WorkspaceID:        id.WorkspaceID,
		CompletedStepCount: &completedStepCount,
		CreatedAt:          time.Now(),
		FunctionID:         id.WorkflowID,
		FunctionVersion:    int64(id.WorkflowVersion),
		GroupID:            groupID,
		RunID:              id.RunID,
		Type:               enums.HistoryTypeFunctionCompleted.String(),
		Attempt:            int64(item.Attempt),
		IdempotencyKey:     id.IdempotencyKey(),
		EventID:            id.EventID,
		BatchID:            id.BatchID,
	}

	err = applyResponse(&h, &resp)
	if err != nil {
		// Swallow error and log, since we don't want a response parsing error
		// to fail history writing.
		l.log.Error(
			"error applying response to history",
			"error", err,
			"run_id", id.RunID.String(),
		)
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
	s state.State,
) {
	completedStepCount := int64(len(s.Actions()) + len(s.Errors()))
	groupID := uuid.New()

	h := History{
		Cron:               s.CronSchedule(),
		ID:                 ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:          id.AccountID,
		WorkspaceID:        id.WorkspaceID,
		CompletedStepCount: &completedStepCount,
		CreatedAt:          time.Now(),
		FunctionID:         id.WorkflowID,
		FunctionVersion:    int64(id.WorkflowVersion),
		GroupID:            &groupID,
		RunID:              id.RunID,
		Type:               enums.HistoryTypeFunctionCancelled.String(),
		IdempotencyKey:     id.IdempotencyKey(),
		EventID:            id.EventID,
		BatchID:            id.BatchID,
		Cancel:             &req,
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
	stepName *string,
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
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepScheduled.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		StepName:        stepName,
		StepID:          &edge.Edge.Incoming, // TODO: Add step name to edge.
		EventID:         id.EventID,
		BatchID:         id.BatchID,
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
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepStarted.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		StepName:        &edge.Incoming,
		StepID:          &edge.Incoming, // TODO: Add step name to edge.
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		URL:             &step.URI,
		LatencyMS:       &latencyMS,
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
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepCompleted.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		StepName:        &resp.Step.Name,
		StepID:          &edge.Incoming,
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		URL:             &step.URI,
	}

	err = applyResponse(&h, &resp)
	if err != nil {
		// Swallow error and log, since we don't want a response parsing error
		// to fail history writing.
		l.log.Error(
			"error applying response to history",
			"error", err,
			"run_id", id.RunID.String(),
		)
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
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepWaiting.String(),
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
	groupID string,
) {
	var groupIDUUID *uuid.UUID
	if groupID != "" {
		val, err := toUUID(groupID)
		if err != nil {
			l.log.Error(
				"error parsing group ID",
				"error", err,
				"group_id", groupID,
				"run_id", id.RunID.String(),
			)
		}
		groupIDUUID = val
	}

	h := History{
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupIDUUID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepCompleted.String(),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		WaitResult: &WaitResult{
			EventID: req.EventID,
			Timeout: req.EventID == nil,
		},
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onWaitForEventResumed", "error", err)
		}
	}
}

// OnInvokeFunction is called when a function is invoked from a step.
func (l lifecycle) OnInvokeFunction(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	op state.GeneratorOpcode,
	eventID ulid.ULID,
	corrID string,
) {
	logger.From(ctx).Debug().Interface("id", id).Msg("OnInvokeFunction")

	groupID, err := toUUID(item.GroupID)
	if err != nil {
		l.log.Error(
			"error parsing group ID",
			"error", err,
			"group_id", item.GroupID,
			"run_id", id.RunID.String(),
		)
	}

	fnID := ""
	expiry := time.Time{}

	opts, err := op.InvokeFunctionOpts()
	if err != nil {
		l.log.Error("error parsing invoke function options", "error", err)
	}

	if opts != nil {
		fnID = opts.FunctionID
		optsExp, err := opts.Expires()
		if err != nil {
			l.log.Error("error parsing invoke function options expiry", "error", err)
		} else {
			expiry = optsExp
		}
	} else {
		l.log.Error("invoke function options are nil")
	}

	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepInvokingFunction.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		StepName:        &op.Name,
		StepID:          &op.ID,
		InvokeFunction: &InvokeFunction{
			EventID:       eventID,
			FunctionID:    fnID,
			Timeout:       expiry,
			CorrelationID: corrID,
		},
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onInvokeFunction", "error", err)
		}
	}
}

// OnInvokeFunctionResumed is called when a function is resumed from an
// invoke function step. This happens when the invoked function has
// completed or the step timed out whilst waiting.
func (l lifecycle) OnInvokeFunctionResumed(
	ctx context.Context,
	id state.Identifier,
	req execution.ResumeRequest,
	groupID string,
) {
	var groupIDUUID *uuid.UUID
	if groupID != "" {
		val, err := toUUID(groupID)
		if err != nil {
			l.log.Error(
				"error parsing group ID",
				"error", err,
				"group_id", groupID,
				"run_id", id.RunID.String(),
			)
		}
		groupIDUUID = val
	}

	h := History{
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupIDUUID,
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepCompleted.String(),
		IdempotencyKey:  id.IdempotencyKey(),
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		InvokeFunctionResult: &InvokeFunctionResult{
			EventID: req.EventID,
			Timeout: req.EventID == nil,
			RunID:   req.RunID,
		},
	}
	for _, d := range l.drivers {
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onInvokeFunctionResumed", "error", err)
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
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:       id.AccountID,
		WorkspaceID:     id.WorkspaceID,
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		GroupID:         groupID,
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
		if err := d.Write(context.WithoutCancel(ctx), h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onSleep", "error", err)
		}
	}
}

func applyResponse(
	h *History,
	resp *state.DriverResponse,
) error {
	h.Result = &Result{
		DurationMS: int(resp.Duration.Milliseconds()),
		RawOutput:  resp.Output,
		SizeBytes:  resp.OutputSize,
		// XXX: Add more fields here
	}

	if outputStr, ok := resp.Output.(string); ok {
		// If it's a completed generator step then some data is stored in the
		// output. We'll try to extract it.
		isGeneratorStep := len(resp.Generator) > 0
		if isGeneratorStep {
			var opcodes []state.GeneratorOpcode
			if err := json.Unmarshal([]byte(outputStr), &opcodes); err == nil {
				if len(opcodes) > 0 && opcodes[0].Op != enums.OpcodeStepPlanned {
					h.StepID = &opcodes[0].ID
					h.StepType = getStepType(opcodes[0])
					h.Result.Output = string(opcodes[0].Data)

					if opcodes[0].DisplayName != nil {
						h.StepName = opcodes[0].DisplayName
					} else {
						// SDK versions < 3.?.? don't respond with the display
						// name, so we we'll use the deprecated name field as a
						// fallback.
						h.StepName = &opcodes[0].Name
					}
				}
				return nil
			}
		}

		// If it's a string and doesn't have extractable data, then
		// assume it's already the stringified JSON for the data
		// returned by the user's step. Some scenarios when that can
		// happen:
		// - FunctionCompleted. It isn't enveloped like generator steps.
		// - StepFailed. It has error-related fields.
		// - StepCompleted preceding parallel steps. Its output schema
		//     conforms to the normal generator StepCompleted schema,
		//     but doesn't contain any of the user's step output data.
		h.Result.Output = outputStr
		return nil
	}

	byt, err := json.Marshal(resp.Output)
	if err != nil {
		return fmt.Errorf("error marshalling step output: %w", err)
	}
	h.Result.Output = string(byt)
	return nil
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

// Returns the user-facing step type. In other words, the returned step type
// should match the code the user wrote (e.g. `step.sleep()` becomes
// enums.HistoryStepTypeSleep).
func getStepType(opcode state.GeneratorOpcode) *enums.HistoryStepType {
	var out enums.HistoryStepType
	switch opcode.Op {
	case enums.OpcodeSleep:
		out = enums.HistoryStepTypeSleep
	case enums.OpcodeStep:
		if opcode.Data == nil {
			// Not a user-facing step.
			return nil
		}

		// This is a hacky way to detect `step.sendEvent()`, but it's all we
		// have until we add an opcode for it.
		if opcode.Name == "sendEvent" {
			out = enums.HistoryStepTypeSend
		} else {
			out = enums.HistoryStepTypeRun
		}
	case enums.OpcodeWaitForEvent:
		out = enums.HistoryStepTypeWait
	default:
		// Not a user-facing step.
		return nil
	}

	return &out
}
