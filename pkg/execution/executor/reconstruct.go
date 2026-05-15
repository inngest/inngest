package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

func reconstruct(ctx context.Context, tr cqrs.TraceReader, req execution.ScheduleRequest, newState *sv2.CreateState) error {
	root, err := tr.GetSpansByRunID(ctx, *req.OriginalRunID)
	if err != nil {
		return fmt.Errorf("error loading original trace spans: %w", err)
	}
	if root == nil {
		return fmt.Errorf("original run trace not found")
	}

	stack, stepSpans, foundStepToRunFrom := reconstructStack(root, req.FromStep.StepID)

	if len(stack) == 0 {
		return fmt.Errorf("stack not found in original run")
	}

	if !foundStepToRunFrom {
		return fmt.Errorf("step to run from not found in original run")
	}

	steps := []state.MemoizedStep{}

	// Copy the state from the original run from the begining until the rerun step
	for _, stepID := range stack {
		if stepID == req.FromStep.StepID {
			break
		}

		span, ok := stepSpans[stepID]
		if !ok {
			// step is in the stack but the span is missing. this
			// is a data integrity issue and we should not
			// attempt to recover from
			return fmt.Errorf("step found in stack but span not found in original run")
		}

		outputID := span.GetOutputID()
		if outputID == nil {
			// no outputs on sleeps.
			if isSleepStep(span) {
				steps = append(steps, state.MemoizedStep{ID: stepID, Data: nil})
				continue
			}

			return fmt.Errorf("step found in stack but output not found in original run")
		}

		var outputIdentifier cqrs.SpanIdentifier
		if err := outputIdentifier.Decode(*outputID); err != nil {
			return fmt.Errorf("error decoding span output ID: %w", err)
		}
		if outputIdentifier.Preview == nil || !*outputIdentifier.Preview {
			return fmt.Errorf("span output is not trace-v2 output")
		}

		output, err := tr.GetSpanOutput(ctx, outputIdentifier)
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

type responseStepIDs struct {
	at  time.Time
	ids []string
}

func reconstructStack(root *cqrs.OtelSpan, fromStepID string) ([]string, map[string]*cqrs.OtelSpan, bool) {
	responseSteps := []responseStepIDs{}
	stepSpans := map[string]*cqrs.OtelSpan{}

	//
	// this gets the response stacks and the spans that constitute each step output
	walkOtelSpans(root, func(span *cqrs.OtelSpan) {
		if span == nil || span.Attributes == nil {
			return
		}

		if span.Attributes.StepID != nil && *span.Attributes.StepID != "" {
			stepID := *span.Attributes.StepID
			//
			// actual output may live on a child execution spans
			if outputSpan := findOutputSpan(span); outputSpan != nil {
				stepSpans[stepID] = outputSpan
			} else if _, ok := stepSpans[stepID]; !ok {
				stepSpans[stepID] = span
			}
		}

		if span.Attributes.ResponseSteps == nil {
			return
		}

		//
		// extract the steps ids
		ids := []string{}
		for _, op := range *span.Attributes.ResponseSteps {
			if op.ID != "" {
				ids = append(ids, op.ID)
			}
		}
		if len(ids) > 0 {
			responseSteps = append(responseSteps, responseStepIDs{
				at:  span.StartTime,
				ids: ids,
			})
		}
	})

	sort.Slice(responseSteps, func(i, j int) bool {
		return responseSteps[i].at.Before(responseSteps[j].at)
	})

	stack := []string{}
	seen := map[string]bool{}
	foundStepToRunFrom := false
	// flatten sdk response stacks, preserve order and dedupe
	for _, responseStep := range responseSteps {
		for _, stepID := range responseStep.ids {
			if seen[stepID] {
				continue
			}
			stack = append(stack, stepID)
			seen[stepID] = true
			if stepID == fromStepID {
				foundStepToRunFrom = true
			}
		}
	}

	return stack, stepSpans, foundStepToRunFrom
}

func walkOtelSpans(span *cqrs.OtelSpan, fn func(*cqrs.OtelSpan)) {
	if span == nil {
		return
	}

	fn(span)
	for _, child := range span.Children {
		walkOtelSpans(child, fn)
	}
}

func findOutputSpan(span *cqrs.OtelSpan) *cqrs.OtelSpan {
	if span == nil {
		return nil
	}

	if span.GetOutputID() != nil {
		return span
	}

	var outputSpan *cqrs.OtelSpan
	for _, child := range span.Children {
		if childOutputSpan := findOutputSpan(child); childOutputSpan != nil {
			outputSpan = childOutputSpan
		}
	}

	return outputSpan
}

func isSleepStep(span *cqrs.OtelSpan) bool {
	return span != nil &&
		span.Attributes != nil &&
		span.Attributes.StepOp != nil &&
		*span.Attributes.StepOp == enums.OpcodeSleep
}
