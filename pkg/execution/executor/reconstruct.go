package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

func reconstruct(ctx context.Context, tr cqrs.TraceReader, req execution.ScheduleRequest, newState *sv2.CreateState) error {

	// Load the original run state and copy the state from the original
	// run to the new run.
	origTraceRun, err := tr.GetTraceRun(ctx, cqrs.TraceRunIdentifier{
		RunID: *req.OriginalRunID,
	})
	if err != nil {
		return fmt.Errorf("error loading original trace run: %w", err)
	}

	spans, err := tr.GetTraceSpansByRun(ctx, cqrs.TraceRunIdentifier{
		AccountID:   req.AccountID,
		WorkspaceID: req.WorkspaceID,
		AppID:       req.AppID,
		FunctionID:  req.Function.ID,
		TraceID:     origTraceRun.TraceID,
		RunID:       *req.OriginalRunID,
	})
	if err != nil {
		return fmt.Errorf("error loading trace spans: %w", err)
	}

	// Get the stack and organize spans by step IDs
	var stack []string
	stepSpans := map[string]*cqrs.Span{}
	foundStepToRunFrom := false

	for _, span := range spans {
		if stepID, ok := span.SpanAttributes[consts.OtelSysStepID]; ok && stepID != "" {
			stepSpans[stepID] = span
			if stepID == req.FromStep.StepID {
				foundStepToRunFrom = true
			}
		}
		if span.SpanName == consts.OtelExecFnOk || span.SpanName == consts.OtelExecFnErr {
			if spanStack, ok := span.SpanAttributes[consts.OtelSysStepStack]; ok {
				stack = strings.Split(spanStack, ",")
			}
		}
	}

	if len(stack) == 0 {
		// This can happen for older runs that don't save the stack; we
		// shouldn't try to recover from this as we could accidentally
		// make the run resolve to a different path without it.
		return fmt.Errorf("stack not found in original run")
	}

	if !foundStepToRunFrom {
		// This implementation has been given a step to run from that
		// doesn't exist in this run.  This is a bad request.
		return fmt.Errorf("step to run from not found in original run")
	}

	steps := []state.MemoizedStep{}

	// Copy the state from the original run to the new run.
	for _, stepID := range stack {
		if stepID == req.FromStep.StepID {
			// We've reached the step to run from, so we can stop
			// copying

			break
		}

		span, ok := stepSpans[stepID]
		if !ok {
			// This signifies that the step was present in the stack but
			// we couldn't find the span that represents it. This
			// indicates a data integrity issue and we should not
			// attempt to recover from this.
			return fmt.Errorf("step found in stack but span not found in original run")
		}

		output, err := tr.GetSpanOutput(ctx, cqrs.SpanIdentifier{
			AccountID:   req.AccountID,
			WorkspaceID: req.WorkspaceID,
			AppID:       req.AppID,
			FunctionID:  req.Function.ID,
			TraceID:     origTraceRun.TraceID,
			SpanID:      span.SpanID,
		})
		if err != nil {
			return fmt.Errorf("error loading span output: %w", err)
		}

		var data any
		_ = json.Unmarshal(output.Data, &data)

		memoizedStep := state.MemoizedStep{
			ID:   stepID,
			Data: map[string]any{"data": data},
		}
		if output.IsError {
			memoizedStep.Data = map[string]any{"error": data}
		}

		steps = append(steps, memoizedStep)
	}

	newState.Steps = steps

	if req.FromStep != nil && req.FromStep.Input != nil {
		newState.StepInputs = []state.MemoizedStep{
			{
				ID:   req.FromStep.StepID,
				Data: req.FromStep.Input,
			},
		}
	}

	return nil
}
