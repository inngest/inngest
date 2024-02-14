package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
)

// instance represents a single executor instance for a function run.
type instance struct {
	*executor

	state state.State
}

// Execute loads a workflow and the current run state, then executes the
// function's step via the necessary driver.
func (e *instance) execute(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge, stackIndex int) (*state.DriverResponse, error) {
	if e.fl == nil {
		return nil, fmt.Errorf("no function loader specified running step")
	}

	s := e.state

	md := s.Metadata()
	if md.Status == enums.RunStatusCancelled {
		return nil, state.ErrFunctionCancelled
	}

	if e.steplimit != 0 && len(s.Actions()) >= int(e.steplimit) {
		// Update this function's state to overflowed, if running.
		if md.Status == enums.RunStatusRunning {
			// XXX: Update error to failed, set error message
			if err := e.sm.SetStatus(ctx, id, enums.RunStatusFailed); err != nil {
				return nil, err
			}

			// Create a new driver response to map as the function finished error.
			resp := state.DriverResponse{}
			resp.SetError(state.ErrFunctionOverflowed)
			resp.SetFinal()

			if err := e.runFinishHandler(ctx, id, s, resp); err != nil {
				logger.From(ctx).Error().Err(err).Msg("error running finish handler")
			}

			for _, e := range e.lifecycles {
				go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, resp, s)
			}
		}
		return nil, state.ErrFunctionOverflowed
	}

	// Check if the function is cancelled.
	if e.cancellationChecker != nil {
		cancel, err := e.cancellationChecker.IsCancelled(
			ctx,
			md.Identifier.WorkspaceID,
			md.Identifier.WorkflowID,
			md.Identifier.RunID,
			s.Event(),
		)
		if err != nil {
			logger.StdlibLogger(ctx).Error(
				"error checking cancellation",
				"error", err.Error(),
				"run_id", md.Identifier.RunID,
				"function_id", md.Identifier.WorkflowID,
				"workspace_id", md.Identifier.WorkspaceID,
			)
		}
		if cancel != nil {
			return nil, e.Cancel(ctx, md.Identifier.RunID, execution.CancelRequest{
				CancellationID: &cancel.ID,
			})
		}
	}

	// If this is the trigger, check if we only have one child.  If so, skip to directly executing
	// that child;  we don't need to handle the trigger individually.
	//
	// This cuts down on queue churn.
	//
	// NOTE: This is a holdover from treating functions as a *series* of DAG calls.  In that case,
	// we automatically enqueue all children of the dag from the root node.
	// This can be cleaned up.
	if edge.Incoming == inngest.TriggerName {
		f, err := e.fl.LoadFunction(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("error loading function for run: %w", err)
		}
		// We only support functions with a single step, as we've removed the DAG based approach.
		// This means that we always execute the first step.
		if len(f.Steps) > 1 {
			return nil, fmt.Errorf("DAG-based steps are no longer supported")
		}
		edge.Outgoing = inngest.TriggerName
		edge.Incoming = f.Steps[0].ID
		// Update the payload
		payload := item.Payload.(queue.PayloadEdge)
		payload.Edge = edge
		item.Payload = payload
		// Add retries from the step to our queue item.  Increase as retries is
		// always one less than attempts.
		retries := f.Steps[0].RetryCount() + 1
		item.MaxAttempts = &retries

		// Only just starting:  run lifecycles on first attempt.
		if item.Attempt == 0 {
			for _, e := range e.lifecycles {
				go e.OnFunctionStarted(context.WithoutCancel(ctx), id, item, s)
			}
		}
	}

	// Ensure that if users requeue steps we never re-execute.
	incoming := edge.Incoming
	if edge.IncomingGeneratorStep != "" {
		incoming = edge.IncomingGeneratorStep
	}
	if resp, _ := s.ActionID(incoming); resp != nil {
		// This has already successfully been executed.
		return &state.DriverResponse{
			Output: resp,
			Err:    nil,
		}, nil
	}

	resp, err := e.run(ctx, id, item, edge, s, stackIndex)
	if resp == nil && err != nil {
		return nil, err
	}

	err = e.HandleResponse(ctx, id, item, edge, resp)
	return resp, err
}

func (e *instance) HandleResponse(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge, resp *state.DriverResponse) error {
	for _, e := range e.lifecycles {
		// OnStepFinished handles step success and step errors/failures.  It is
		// currently the responsibility of the lifecycle manager to handle the differing
		// step statuses when a step finishes.
		//
		// TODO (tonyhb): This should probably change, as each lifecycle listener has to
		// do the same parsing & conditional checks.
		go e.OnStepFinished(context.WithoutCancel(ctx), id, item, edge, resp.Step, *resp)
	}

	// Check for temporary failures.  The outputs of transient errors are not
	// stored in the state store;  they're tracked via executor lifecycle methods
	// for logging.
	//
	// NOTE: If the SDK was running a step (NOT function code) and quit gracefully,
	// resp.UserError will always be set, even if the step itself throws a non-retriable
	// error.
	//
	// This is purely for network errors or top-level function code errors.
	if resp.Err != nil {
		if resp.Retryable() {
			// Retries are a native aspect of the queue;  returning errors always
			// retries steps if possible.
			for _, e := range e.lifecycles {
				// Run the lifecycle method for this retry, which is baked into the queue.
				item.Attempt += 1
				go e.OnStepScheduled(context.WithoutCancel(ctx), id, item, &resp.Step.Name)
			}

			return resp
		}

		// If resp.Err != nil, we don't know whether to invoke the fn again
		// with per-step errors, as we don't know if the intent behind this queue item
		// is a step.
		//
		// In this case, for non-retryable errors, we ignore and fail the function;
		// only OpcodeStepError causes try/catch to be handled and us to continue
		// on error.
		//
		// TODO: Improve this.

		// Check if this step permanently failed.  If so, the function is a failure.
		if !resp.Retryable() {
			if serr := e.sm.SetStatus(ctx, id, enums.RunStatusFailed); serr != nil {
				return fmt.Errorf("error marking function as complete: %w", serr)
			}
			// TODO: Remove this call.
			s, err := e.sm.Load(ctx, id.RunID)
			if err != nil {
				return fmt.Errorf("unable to load run: %w", err)
			}

			if err := e.runFinishHandler(ctx, id, s, *resp); err != nil {
				logger.From(ctx).Error().Err(err).Msg("error running finish handler")
			}

			for _, e := range e.lifecycles {
				go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, *resp, s)
			}
			return resp
		}
	}

	// This is a success, which means either a generator or a function result.
	if len(resp.Generator) > 0 {
		// Handle generator responses then return.
		if serr := e.HandleGeneratorResponse(ctx, resp, item); serr != nil {
			// If this is an error compiling async expressions, fail the function.
			if strings.Contains(serr.Error(), "error compiling expression") {
				resp.SetError(serr)
				resp.SetFinal()
				_ = e.sm.SaveResponse(ctx, id, resp.Step.ID, resp.Error())
				if serr := e.sm.SetStatus(ctx, id, enums.RunStatusFailed); serr != nil {
					return fmt.Errorf("error marking function as complete: %w", serr)
				}
				// TODO: Remove load call.
				s, err := e.sm.Load(ctx, id.RunID)
				if err != nil {
					return fmt.Errorf("unable to load run: %w", err)
				}
				if err := e.runFinishHandler(ctx, id, s, *resp); err != nil {
					logger.From(ctx).Error().Err(err).Msg("error running finish handler")
				}
				for _, e := range e.lifecycles {
					go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, *resp, s)
				}
				return nil
			}
			return fmt.Errorf("error handling generator response: %w", serr)
		}
		return nil
	}

	// This is the function result.

	// TODO: Use state loaded before function call instead of loading once again
	// to reduce load.  That way, we never need to call SaveResponse and Load().
	//
	// Save this in the state store (which will inevitably be GC'd), and end
	output, err := json.Marshal(resp.Output)
	if err != nil {
		return err
	}

	if serr := e.sm.SaveResponse(ctx, id, resp.Step.ID, string(output)); serr != nil {
		// Final function responses can be duplicated if multiple parallel
		// executions reach the end at the same time. Steps themselves are
		// de-duplicated in the queue.
		if serr == state.ErrDuplicateResponse {
			return resp
		}
		return fmt.Errorf("error saving function output: %w", serr)
	}

	// TODO: Remove load call.
	s, err := e.sm.Load(ctx, id.RunID)
	if err != nil {
		return fmt.Errorf("unable to load run: %w", err)
	}
	// end todo

	if err := e.runFinishHandler(ctx, id, s, *resp); err != nil {
		logger.From(ctx).Error().Err(err).Msg("error running finish handler")
	}

	for _, e := range e.lifecycles {
		go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, *resp, s)
	}

	if serr := e.sm.SetStatus(ctx, id, enums.RunStatusCompleted); serr != nil {
		return fmt.Errorf("error marking function as complete: %w", serr)
	}

	return nil
}

func (e *instance) HandleGeneratorResponse(ctx context.Context, resp *state.DriverResponse, item queue.Item) error {
	// TODO: xxx
	md := e.state.Metadata()

	{
		// The following code helps with parallelism and the V2 -> V3 executor changes
		var update *state.MetadataUpdate
		// NOTE: We only need to set hash versions when handling generator responses, else the
		// fn is ending and it doesn't matter.
		if md.RequestVersion == -1 {
			update = &state.MetadataUpdate{
				Context:                   md.Context,
				Debugger:                  md.Debugger,
				DisableImmediateExecution: md.DisableImmediateExecution,
				RequestVersion:            resp.RequestVersion,
			}
		}
		if len(resp.Generator) > 1 {
			if !md.DisableImmediateExecution {
				// With parallelism, we currently instruct the SDK to disable immediate execution,
				// enforcing that every step becomes pre-planned.
				if update == nil {
					update = &state.MetadataUpdate{
						Context:                   md.Context,
						Debugger:                  md.Debugger,
						DisableImmediateExecution: true,
						RequestVersion:            resp.RequestVersion,
					}
				}
				update.DisableImmediateExecution = true
			}
		}
		if update != nil {
			if err := e.sm.UpdateMetadata(ctx, item.Identifier.RunID, *update); err != nil {
				return fmt.Errorf("error updating function metadata: %w", err)
			}
		}
	}

	groups := opGroups(resp.Generator).All()
	for _, group := range groups {
		if err := e.handleGeneratorGroup(ctx, group, resp, item); err != nil {
			return err
		}
	}

	return nil
}

func (e *instance) handleGeneratorGroup(ctx context.Context, group OpcodeGroup, resp *state.DriverResponse, item queue.Item) error {
	eg := errgroup.Group{}
	for _, op := range group.Opcodes {
		if op == nil {
			// This is clearly an error.
			if e.log != nil {
				e.log.Error().Err(fmt.Errorf("nil generator returned")).Msg("error handling generator")
			}
			continue
		}
		copied := *op

		newItem := item
		if group.ShouldStartHistoryGroup {
			// Give each opcode its own group ID, since we want to track each
			// parellel step individually.
			newItem.GroupID = uuid.New().String()
		}

		eg.Go(func() error { return e.HandleGenerator(ctx, copied, newItem) })
	}
	if err := eg.Wait(); err != nil {
		if resp.NoRetry {
			return queue.NeverRetryError(err)
		}
		if resp.RetryAt != nil {
			return queue.RetryAtError(err, resp.RetryAt)
		}
		return err
	}

	return nil
}

func (e *instance) HandleGenerator(ctx context.Context, gen state.GeneratorOpcode, item queue.Item) error {
	// Grab the edge that triggered this step execution.
	edge, ok := item.Payload.(queue.PayloadEdge)
	if !ok {
		return fmt.Errorf("unknown queue item type handling generator: %T", item.Payload)
	}

	switch gen.Op {
	case enums.OpcodeNone:
		// OpcodeNone essentially terminates this "thread" or execution path.  We don't need to do
		// anything - including scheduling future steps.
		//
		// This is necessary for parallelization:  we may fan out from 1 step -> 10 parallel steps,
		// then need to coalesce back to a single thread after all 10 have finished.  We expect
		// drivers/the SDK to return OpcodeNone for all but the last of parallel steps.
		return nil
	case enums.OpcodeStep, enums.OpcodeStepRun:
		return e.handleGeneratorStep(ctx, gen, item, edge)
	case enums.OpcodeStepError:
		return e.handleStepError(ctx, gen, item, edge)
	case enums.OpcodeStepPlanned:
		return e.handleGeneratorStepPlanned(ctx, gen, item, edge)
	case enums.OpcodeSleep:
		return e.handleGeneratorSleep(ctx, gen, item, edge)
	case enums.OpcodeWaitForEvent:
		return e.handleGeneratorWaitForEvent(ctx, gen, item, edge)
	case enums.OpcodeInvokeFunction:
		return e.handleGeneratorInvokeFunction(ctx, gen, item, edge)
	}

	return fmt.Errorf("unknown opcode: %s", gen.Op)
}

// handleGeneratorStep handles OpcodeStep and OpcodeStepRun, both indicating that a function step
// has finished
func (e *instance) handleGeneratorStep(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Going from the current step
		Incoming: edge.Edge.Incoming, // And re-calling the incoming function in a loop
	}

	// Save the response to the state store.
	output, err := gen.Output()
	if err != nil {
		return err
	}

	if err := e.sm.SaveResponse(ctx, item.Identifier, gen.ID, output); err != nil {
		return err
	}

	// Update the group ID in context;  we've already saved this step's success and we're now
	// running the step again, needing a new history group
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	// Re-enqueue the exact same edge to run now.
	jobID := fmt.Sprintf("%s-%s", item.Identifier.IdempotencyKey(), gen.ID)
	nextItem := queue.Item{
		JobID:       &jobID,
		WorkspaceID: item.WorkspaceID,
		GroupID:     groupID,
		Kind:        queue.KindEdge,
		Identifier:  item.Identifier,
		Attempt:     0,
		MaxAttempts: item.MaxAttempts,
		Payload:     queue.PayloadEdge{Edge: nextEdge},
	}
	err = e.queue.Enqueue(ctx, nextItem, time.Now())
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		// We can't specify step name here since that will result in the
		// "followup discovery step" having the same name as its predecessor.
		var stepName *string = nil
		go l.OnStepScheduled(ctx, item.Identifier, nextItem, stepName)
	}

	return err
}

func (e *instance) handleStepError(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	// With the introduction of the StepError opcode, step errors are handled graceully and we can
	// finally distinguish between application level errors (this function) and network errors/other
	// errors (as the SDK didn't return this opcode).
	//
	// Here, we need to process the error and ensure that we reschedule the job for the future.
	//
	// Things to bear in mind:
	// - Steps throwing/returning NonRetriableErrors are still OpcodeStepError
	// - We are now in charge of rescheduling the entire function

	if gen.Error == nil {
		// This should never happen.
		logger.StdlibLogger(ctx).Error("OpcodeStepError handled without user error", "gen", gen)
		return fmt.Errorf("no user error defined in OpcodeStepError")
	}

	// If this is the last attempt, store the error in the state store, with a
	// wrapping of "error".  The wrapping allows SDKs to understand whether the
	// memoized step data is an error (and they should throw/return an error) or
	// real data.
	//
	// State stored for each step MUST always be wrapped with either "error" or "data".
	retryable := true

	if gen.Error.NoRetry {
		// This is a NonRetryableError thrown in a step.
		retryable = false
	}
	if !queue.ShouldRetry(nil, item.Attempt, item.GetMaxAttempts()) {
		// This is the last attempt as per the attempt in the queue, which
		// means we've failed N times, and so it is not retryable.
		retryable = false
	}

	if retryable {
		// Return an error to trigger standard queue retries.
		for _, l := range e.lifecycles {
			item.Attempt += 1
			go l.OnStepScheduled(ctx, item.Identifier, item, &gen.Name)
		}
		return ErrHandledStepError
	}

	// This was the final step attempt and we still failed.
	//
	// First, save the error to our state store.
	//
	// Note that `onStepFinished` is called immediately after a step response is returned, so
	// the history for this error will have already been handled.
	output, err := gen.Output()
	if err != nil {
		return err
	}
	if err := e.sm.SaveResponse(ctx, item.Identifier, gen.ID, output); err != nil {
		return err
	}

	// Because this is a final step error that was handled gracefully, enqueue
	// another attempt to the function with a new edge type.
	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Going from the current step
		Incoming: edge.Edge.Incoming, // And re-calling the incoming function in a loop
	}
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)
	jobID := fmt.Sprintf("%s-%s-failure", item.Identifier.IdempotencyKey(), gen.ID)

	nextItem := queue.Item{
		JobID:       &jobID,
		WorkspaceID: item.WorkspaceID,
		GroupID:     groupID,
		Kind:        queue.KindEdgeError,
		Identifier:  item.Identifier,
		Attempt:     0,
		MaxAttempts: item.MaxAttempts,
		Payload:     queue.PayloadEdge{Edge: nextEdge},
	}
	err = e.queue.Enqueue(ctx, nextItem, time.Now())
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		go l.OnStepScheduled(ctx, item.Identifier, nextItem, nil)
	}

	return nil
}

func (e *instance) handleGeneratorStepPlanned(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	nextEdge := inngest.Edge{
		// Planned generator IDs are the same as the actual OpcodeStep IDs.
		// We can't set edge.Edge.Outgoing here because the step hasn't yet ran.
		//
		// We do, though, want to store the incomin step ID name _without_ overriding
		// the actual DAG step, though.
		// Run the same action.
		IncomingGeneratorStep: gen.ID,
		Outgoing:              edge.Edge.Outgoing,
		Incoming:              edge.Edge.Incoming,
	}

	// Update the group ID in context;  we're scheduling a step, and we want
	// to start a new history group for this item.
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	// Re-enqueue the exact same edge to run now.
	jobID := fmt.Sprintf("%s-%s", item.Identifier.IdempotencyKey(), gen.ID+"-plan")
	nextItem := queue.Item{
		JobID:       &jobID,
		GroupID:     groupID, // Ensure we correlate future jobs with this group ID, eg. started/failed.
		WorkspaceID: item.WorkspaceID,
		Kind:        queue.KindEdge,
		Identifier:  item.Identifier,
		Attempt:     0,
		MaxAttempts: item.MaxAttempts,
		Payload: queue.PayloadEdge{
			Edge: nextEdge,
		},
	}
	err := e.queue.Enqueue(ctx, nextItem, time.Now())
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		go l.OnStepScheduled(ctx, item.Identifier, nextItem, &gen.Name)
	}
	return err
}

// handleSleep handles the sleep opcode, ensuring that we enqueue the function to rerun
// at the correct time.
func (e *instance) handleGeneratorSleep(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	dur, err := gen.SleepDuration()
	if err != nil {
		return err
	}

	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Leaving sleep
		Incoming: edge.Edge.Incoming, // To re-call the SDK
	}

	// Create another group for the next item which will run.  We're enqueueing
	// the function to run again after sleep, so need a new group.
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	until := time.Now().Add(dur)

	jobID := fmt.Sprintf("%s-%s", item.Identifier.IdempotencyKey(), gen.ID)
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		WorkspaceID: item.WorkspaceID,
		// Sleeps re-enqueue the step so that we can mark the step as completed
		// in the executor after the sleep is complete.  This will re-call the
		// generator step, but we need the same group ID for correlation.
		GroupID:     groupID,
		Kind:        queue.KindSleep,
		Identifier:  item.Identifier,
		Attempt:     0,
		MaxAttempts: item.MaxAttempts,
		Payload:     queue.PayloadEdge{Edge: nextEdge},
	}, until)
	if err == redis_state.ErrQueueItemExists {
		// Safely ignore this error.
		return nil
	}

	for _, e := range e.lifecycles {
		go e.OnSleep(context.WithoutCancel(ctx), item.Identifier, item, gen, until)
	}

	return err
}

func (e *instance) handleGeneratorInvokeFunction(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	logger.From(ctx).Info().Msg("handling invoke function")
	if e.handleSendingEvent == nil {
		return fmt.Errorf("no handleSendingEvent function specified")
	}

	opts, err := gen.InvokeFunctionOpts()
	if err != nil {
		return fmt.Errorf("unable to parse invoke function opts: %w", err)
	}
	expires, err := opts.Expires()
	if err != nil {
		return fmt.Errorf("unable to parse invoke function expires: %w", err)
	}

	eventName := event.FnFinishedName
	correlationID := item.Identifier.RunID.String() + "." + gen.ID
	strExpr := fmt.Sprintf("async.data.%s == %s", consts.InvokeCorrelationId, strconv.Quote(correlationID))
	_, err = e.newExpressionEvaluator(ctx, strExpr)
	if err != nil {
		return execError{err: fmt.Errorf("failed to create expression to wait for invoked function completion: %w", err)}
	}

	logger.From(ctx).Info().Interface("opts", opts).Time("expires", expires).Str("event", eventName).Str("expr", strExpr).Msg("parsed invoke function opts")

	pauseID := uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte(item.Identifier.RunID.String()+gen.ID),
	)

	opcode := gen.Op.String()
	err = e.sm.SavePause(ctx, state.Pause{
		ID:          pauseID,
		WorkspaceID: item.WorkspaceID,
		Identifier:  item.Identifier,
		GroupID:     item.GroupID,
		Outgoing:    gen.ID,
		Incoming:    edge.Edge.Incoming,
		StepName:    gen.UserDefinedName(),
		Opcode:      &opcode,
		Expires:     state.Time(expires),
		Event:       &eventName,
		Expression:  &strExpr,
		DataKey:     gen.ID,
	})
	if err == state.ErrPauseAlreadyExists {
		if e.log != nil {
			e.log.Warn().
				Str("pause_id", pauseID.String()).
				Str("run_id", item.Identifier.RunID.String()).
				Str("workflow_id", item.Identifier.WorkflowID.String()).
				Msg("created duplicate pause")
		}
		return nil
	}
	if err != nil {
		return err
	}

	// Enqueue a job that will timeout the pause.
	jobID := fmt.Sprintf("%s-%s-%s", item.Identifier.IdempotencyKey(), gen.ID, "invoke")
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		WorkspaceID: item.WorkspaceID,
		// Use the same group ID, allowing us to track the cancellation of
		// the step correctly.
		GroupID:    item.GroupID,
		Kind:       queue.KindPause,
		Identifier: item.Identifier,
		Payload: queue.PayloadPauseTimeout{
			PauseID:   pauseID,
			OnTimeout: true,
		},
	}, expires)
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	// Always create an invocation event.
	evt := event.NewInvocationEvent(event.NewInvocationEventOpts{
		Event:         *opts.Payload,
		FnID:          opts.FunctionID,
		CorrelationID: &correlationID,
	})

	logger.From(ctx).Debug().Interface("evt", evt).Str("gen.ID", gen.ID).Msg("created invocation event")

	err = e.handleSendingEvent(ctx, evt, item)
	if err != nil {
		// XXX; we should remove the pause above.
		return fmt.Errorf("error publishing internal invocation event: %w", err)
	}

	for _, e := range e.lifecycles {
		go e.OnInvokeFunction(context.WithoutCancel(ctx), item.Identifier, item, gen, ulid.MustParse(evt.ID), correlationID)
	}

	return err
}

func (e *instance) handleGeneratorWaitForEvent(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	opts, err := gen.WaitForEventOpts()
	if err != nil {
		return fmt.Errorf("unable to parse wait for event opts: %w", err)
	}
	expires, err := opts.Expires()
	if err != nil {
		return fmt.Errorf("unable to parse wait for event expires: %w", err)
	}

	// Filter the expression data such that it contains only the variables used
	// in the expression.
	data := map[string]any{}
	if opts.If != nil {
		if err := expressions.Validate(ctx, *opts.If); err != nil {
			return execError{err, true}
		}

		expr, err := e.newExpressionEvaluator(ctx, *opts.If)
		if err != nil {
			return execError{err, true}
		}

		// Take the data for expressions based off of state.
		ed := expressions.NewData(state.ExpressionData(ctx, e.state))
		data = expr.FilteredAttributes(ctx, ed).Map()
	}

	pauseID := uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte(item.Identifier.RunID.String()+gen.ID),
	)

	expr := opts.If
	if expr != nil && strings.Contains(*expr, "event.") {
		// Remove `event` data from the expression and replace with actual event
		// data as values, now that we have the event.
		//
		// This improves performance in matching, as we can then use the values within
		// aggregate trees.
		interpolated, err := expressions.Interpolate(ctx, *opts.If, map[string]any{
			"event": e.state.Event(),
		})
		if err != nil {
			logger.StdlibLogger(ctx).Warn(
				"error interpolating waitForEvent expression",
				"error", err,
				"expression", *opts.If,
			)
		}
		expr = &interpolated

		// Update the generator to use the interpolated data, ensuring history is updated.
		opts.If = expr
		gen.Opts = opts
	}

	opcode := gen.Op.String()
	err = e.sm.SavePause(ctx, state.Pause{
		ID:             pauseID,
		WorkspaceID:    item.WorkspaceID,
		Identifier:     item.Identifier,
		GroupID:        item.GroupID,
		Outgoing:       gen.ID,
		Incoming:       edge.Edge.Incoming,
		StepName:       gen.UserDefinedName(),
		Opcode:         &opcode,
		Expires:        state.Time(expires),
		Event:          &opts.Event,
		Expression:     expr,
		ExpressionData: data,
		DataKey:        gen.ID,
	})
	if err == state.ErrPauseAlreadyExists {
		if e.log != nil {
			e.log.Warn().
				Str("pause_id", pauseID.String()).
				Str("run_id", item.Identifier.RunID.String()).
				Str("workflow_id", item.Identifier.WorkflowID.String()).
				Msg("created duplicate pause")
		}
		return nil
	}
	if err != nil {
		return err
	}

	// SDK-based event coordination is called both when an event is received
	// OR on timeout, depending on which happens first.  Both routes consume
	// the pause so this race will conclude by calling the function once, as only
	// one thread can lease and consume a pause;  the other will find that the
	// pause is no longer available and return.
	jobID := fmt.Sprintf("%s-%s-%s", item.Identifier.IdempotencyKey(), gen.ID, "wait")
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		WorkspaceID: item.WorkspaceID,
		// Use the same group ID, allowing us to track the cancellation of
		// the step correctly.
		GroupID:    item.GroupID,
		Kind:       queue.KindPause,
		Identifier: item.Identifier,
		Payload: queue.PayloadPauseTimeout{
			PauseID:   pauseID,
			OnTimeout: true,
		},
	}, expires)
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, e := range e.lifecycles {
		go e.OnWaitForEvent(context.WithoutCancel(ctx), item.Identifier, item, gen)
	}

	return err
}

// run executes the step with the given step ID.
//
// A nil response with an error indicates that an internal error occurred and the step
// did not run.
func (e *instance) run(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge, s state.State, stackIndex int) (*state.DriverResponse, error) {
	f, err := e.fl.LoadFunction(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error loading function for run: %w", err)
	}

	var step *inngest.Step
	for _, s := range f.Steps {
		if s.ID == edge.Incoming {
			step = &s
			break
		}
	}
	if step == nil {
		// Sanity check we've enqueued the right step.
		return nil, newFinalError(fmt.Errorf("unknown vertex: %s", edge.Incoming))
	}

	for _, e := range e.lifecycles {
		go e.OnStepStarted(context.WithoutCancel(ctx), id, item, edge, *step, s)
	}

	// Execute the actual step.
	response, err := e.executeDriverForStep(ctx, id, item, step, s, edge, stackIndex)

	if response.Err != nil && err == nil {
		// This step errored, so always return an error.
		return response, fmt.Errorf("%s", *response.Err)
	}
	return response, err
}

// executeDriverForStep runs the enqueued step by invoking the driver.  It also inspects
// and normalizes responses (eg. max retry attempts).
func (e *instance) executeDriverForStep(ctx context.Context, id state.Identifier, item queue.Item, step *inngest.Step, s state.State, edge inngest.Edge, stackIndex int) (*state.DriverResponse, error) {
	d, ok := e.runtimeDrivers[step.Driver()]
	if !ok {
		return nil, fmt.Errorf("%w: '%s'", ErrNoRuntimeDriver, step.Driver())
	}

	response, err := d.Execute(ctx, s, item, edge, *step, stackIndex, item.Attempt)

	if response == nil {
		response = &state.DriverResponse{
			Step: *step,
		}
	}
	if err != nil && response.Err == nil {
		// Set the response error if it wasn't set, or if Execute had an internal error.
		// This ensures that we only ever need to check resp.Err to handle errors.
		errstr := err.Error()
		response.Err = &errstr
	}
	// Ensure that the step is always set.  This removes the need for drivers to always
	// set this.
	if response.Step.ID == "" {
		response.Step = *step
	}

	// If there's one opcode and it's of type StepError, ensure we set resp.Err to
	// a string containing the response error.
	//
	// TODO: Refactor response.Err
	if len(response.Generator) == 1 && response.Generator[0].Op == enums.OpcodeStepError {
		if !queue.ShouldRetry(nil, item.Attempt, step.RetryCount()+1) {
			response.NoRetry = true
		}
	}

	// Max attempts is encoded at the queue level from step configuration.  If we're at max attempts,
	// ensure the response's NoRetry flag is set, as we shouldn't retry any more.  This also ensures
	// that we properly handle this response as a Failure (permanent) vs an Error (transient).
	if response.Err != nil && !queue.ShouldRetry(nil, item.Attempt, step.RetryCount()+1) {
		response.NoRetry = true
	}

	return response, err
}
