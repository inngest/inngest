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
	"github.com/inngest/inngest/pkg/tracing/meta"
)

type reconstructResult struct {
	fromStepID string
	fromStepOp *enums.Opcode
}

func reconstruct(ctx context.Context, tr cqrs.TraceReader, req execution.ScheduleRequest, newState *sv2.CreateState) (*reconstructResult, error) {
	root, err := tr.GetSpansByRunID(ctx, *req.OriginalRunID)
	if err != nil {
		return nil, fmt.Errorf("error loading original trace spans: %w", err)
	}
	if root == nil {
		return nil, fmt.Errorf("original run trace not found")
	}

	fromStepID, err := resolveRerunStepID(root, req.FromStep.StepID)
	if err != nil {
		return nil, err
	}

	//
	// executor.step spans identify completed step order and output IDs.
	stepsToCopy, _ := reconstructSteps(root, fromStepID)

	result := &reconstructResult{
		fromStepID: fromStepID,
		fromStepOp: stepsToCopy.fromStepOp(),
	}

	steps := []state.MemoizedStep{}

	for _, step := range stepsToCopy {
		if step.id == fromStepID {
			break
		}

		outputID := step.stepSpan.GetOutputID()
		if outputID == nil {
			if isNoOutputStep(step.stepSpan) {
				steps = append(steps, state.MemoizedStep{ID: step.id, Data: nil})
				continue
			}

			return nil, fmt.Errorf("step output not found in original run")
		}

		var outputIdentifier cqrs.SpanIdentifier
		if err := outputIdentifier.Decode(*outputID); err != nil {
			return nil, fmt.Errorf("error decoding span output ID: %w", err)
		}
		if outputIdentifier.Preview == nil || !*outputIdentifier.Preview {
			return nil, fmt.Errorf("span output is not trace-v2 output")
		}

		output, err := tr.GetSpanOutput(ctx, outputIdentifier)
		if err != nil {
			return nil, fmt.Errorf("error loading span output: %w", err)
		}

		var data any
		_ = json.Unmarshal(output.Data, &data)

		memoizedStep := state.MemoizedStep{
			ID:   step.id,
			Data: map[string]any{"data": data},
		}
		if output.IsError {
			memoizedStep.Data = map[string]any{"error": data}
		}

		steps = append(steps, memoizedStep)
	}

	newState.Steps = steps

	if req.FromStep.Input != nil {
		//
		// Rerun from step can alter input for FromStep, so keep it out of completed step output.
		newState.StepInputs = []state.MemoizedStep{
			{
				ID:   fromStepID,
				Data: req.FromStep.Input,
			},
		}
	}

	return result, nil
}

func resolveRerunStepID(root *cqrs.OtelSpan, requested string) (string, error) {
	matchingIDs := map[string]struct{}{}

	walkOtelSpans(root, func(span *cqrs.OtelSpan) {
		if span == nil || span.Attributes == nil || span.Name != meta.SpanNameStep {
			return
		}
		if span.Attributes.StepID == nil || *span.Attributes.StepID == "" {
			return
		}

		stepID := *span.Attributes.StepID
		if stepID == requested {
			matchingIDs = map[string]struct{}{stepID: {}}
			return
		}
		if span.GetStepName() == requested {
			matchingIDs[stepID] = struct{}{}
		}
	})

	if _, ok := matchingIDs[requested]; ok {
		return requested, nil
	}
	if len(matchingIDs) == 0 {
		return "", fmt.Errorf("step to run from not found in original run")
	}
	if len(matchingIDs) > 1 {
		return "", fmt.Errorf("step name matches multiple steps in original run")
	}
	for stepID := range matchingIDs {
		return stepID, nil
	}
	return "", fmt.Errorf("step to run from not found in original run")
}

type reconstructStepsResult []reconstructStep

type reconstructStep struct {
	id         string
	at         time.Time
	attempt    int
	isFromStep bool
	stepOp     *enums.Opcode
	stepSpan   *cqrs.OtelSpan
}

func reconstructSteps(root *cqrs.OtelSpan, fromStepID string) (reconstructStepsResult, bool) {
	stepsByID := map[string]reconstructStep{}
	foundStepToRunFrom := false

	walkOtelSpans(root, func(span *cqrs.OtelSpan) {
		if span == nil || span.Attributes == nil {
			return
		}

		if span.Name != meta.SpanNameStep {
			return
		}

		if span.Attributes.StepID == nil || *span.Attributes.StepID == "" {
			return
		}

		stepID := *span.Attributes.StepID
		if stepID == fromStepID {
			foundStepToRunFrom = true
		}

		next := reconstructStep{
			id:         stepID,
			at:         span.StartTime,
			attempt:    reconstructStepAttempt(span),
			isFromStep: stepID == fromStepID,
			stepOp:     span.Attributes.StepOp,
			stepSpan:   span,
		}

		//
		// Retries can produce more than one span for a step ID; use the latest attempt.
		if current, ok := stepsByID[stepID]; !ok || reconstructStepPreferred(current, next) {
			stepsByID[stepID] = next
		}
	})

	steps := make([]reconstructStep, 0, len(stepsByID))
	for _, step := range stepsByID {
		steps = append(steps, step)
	}

	sort.SliceStable(steps, func(i, j int) bool {
		return reconstructStepLess(steps[i], steps[j])
	})

	return steps, foundStepToRunFrom
}

func (r reconstructStepsResult) fromStepOp() *enums.Opcode {
	for _, step := range r {
		if step.isFromStep {
			return step.stepOp
		}
	}
	return nil
}

func reconstructStepPreferred(current, next reconstructStep) bool {
	if current.attempt != next.attempt {
		return current.attempt < next.attempt
	}

	return reconstructStepLess(current, next)
}

func reconstructStepLess(a, b reconstructStep) bool {
	if !a.at.Equal(b.at) {
		return a.at.Before(b.at)
	}

	if a.attempt != b.attempt {
		return a.attempt < b.attempt
	}

	if a.stepSpan.GetSpanID() != b.stepSpan.GetSpanID() {
		return a.stepSpan.GetSpanID() < b.stepSpan.GetSpanID()
	}

	return a.id < b.id
}

func reconstructStepAttempt(span *cqrs.OtelSpan) int {
	if span.Attributes.StepAttempt == nil {
		return 0
	}

	return *span.Attributes.StepAttempt
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

func isNoOutputStep(span *cqrs.OtelSpan) bool {
	if span == nil || span.Attributes == nil || span.Attributes.StepOp == nil {
		return false
	}

	if *span.Attributes.StepOp == enums.OpcodeSleep {
		return true
	}

	return span.Attributes.StepWaitExpired != nil &&
		*span.Attributes.StepWaitExpired &&
		(*span.Attributes.StepOp == enums.OpcodeWaitForEvent || *span.Attributes.StepOp == enums.OpcodeWaitForSignal)
}
