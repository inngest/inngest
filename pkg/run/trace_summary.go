package run

import (
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
)

type SpanSummary struct {
	Status     *enums.RunStatus
	StartedAt  *time.Time
	EndedAt    *time.Time
	DurationMs *uint64
	OutputID   *string
}

type RunSummaryState struct {
	Status    enums.RunStatus
	StartedAt time.Time
	EndedAt   *time.Time
	HasOutput bool
}

func SummarizeSpanTree(root *cqrs.OtelSpan) (*SpanSummary, error) {
	if root == nil {
		return nil, fmt.Errorf("span is required")
	}

	ensureOtelExtractedValues(root)

	// Start with the root span's defaults
	summary := &SpanSummary{
		Status:    stepStatusToRunStatus(OtelSpanStatus(root)),
		StartedAt: runStartedAt(root),
		EndedAt:   runEndedAt(root),
		OutputID:  root.GetOutputID(),
	}

	// Some current execution paths store the final run output on a descendant
	// execution spans
	if functionOutput := LatestFunctionOutputSpan(root); functionOutput != nil {
		if outputID := functionOutput.GetOutputID(); outputID != nil {
			summary.OutputID = outputID
		}
		if status := stepStatusToRunStatus(OtelSpanStatus(functionOutput)); status != nil {
			summary.Status = status
		}
		if summary.Status != nil && enums.RunStatusEnded(*summary.Status) {
			summary.EndedAt = spanEndedAt(functionOutput)
		}
	}

	// If the root is queued or unknown, prefer a later active child
	if summary.Status == nil || *summary.Status == enums.RunStatusScheduled || *summary.Status == enums.RunStatusUnknown {
		if active := latestActiveSpan(root); active != nil {
			summary.Status = stepStatusToRunStatus(OtelSpanStatus(active))
		}
	}

	// Ended runs should have an end time even when the root span was not
	// updated with one.
	if summary.EndedAt == nil && summary.Status != nil && enums.RunStatusEnded(*summary.Status) {
		if terminal := latestTerminalSpan(root); terminal != nil {
			summary.EndedAt = spanEndedAt(terminal)
		}
	}

	summary.DurationMs = DurationMS(summary.StartedAt, summary.EndedAt)

	return summary, nil
}

// Avoid the extra trace read when we already have the fields we need
func ShouldHydrateRunSummary(state RunSummaryState, includeOutput bool) bool {
	if includeOutput && !state.HasOutput {
		return true
	}
	if state.EndedAt == nil {
		return true
	}

	switch state.Status {
	case enums.RunStatusUnknown, enums.RunStatusScheduled, enums.RunStatusRunning:
		return true
	default:
		return false
	}
}

func (summary *SpanSummary) ApplyTo(state RunSummaryState) RunSummaryState {
	if summary == nil {
		return state
	}
	if summary.Status != nil {
		state.Status = *summary.Status
	}
	if summary.StartedAt != nil {
		state.StartedAt = *summary.StartedAt
	}
	if summary.EndedAt != nil {
		state.EndedAt = summary.EndedAt
	}
	return state
}

func ensureOtelExtractedValues(root *cqrs.OtelSpan) {
	walkOtelSpans(root, func(span *cqrs.OtelSpan) {
		if span.Attributes == nil {
			span.Attributes = &meta.ExtractedValues{}
		}
	})
}

func OtelSpanStatus(span *cqrs.OtelSpan) *enums.StepStatus {
	if span == nil {
		return nil
	}
	// DynamicStatus mirrors the trace converter behavior: updates may land in
	// attributes without being reflected in the rolled-up status field.
	if span.Attributes != nil && span.Attributes.DynamicStatus != nil {
		return span.Attributes.DynamicStatus
	}
	if span.Status == enums.StepStatusUnknown {
		return nil
	}
	return &span.Status
}

func runStartedAt(root *cqrs.OtelSpan) *time.Time {
	if startedAt := spanStartedAt(root, false); startedAt != nil {
		return startedAt
	}

	// Run roots often have a commit timestamp but not a true start timestamp,
	// so use the first child start before falling back to root StartTime.
	var selected *time.Time
	walkOtelSpans(root, func(span *cqrs.OtelSpan) {
		if span == root {
			return
		}
		startedAt := spanStartedAt(span, true)
		if startedAt == nil {
			return
		}
		if selected == nil || startedAt.Before(*selected) {
			selected = startedAt
		}
	})
	if selected != nil {
		return selected
	}

	return spanStartedAt(root, true)
}

func runEndedAt(root *cqrs.OtelSpan) *time.Time {
	if endedAt := root.GetEndedAtTime(); endedAt != nil {
		return endedAt
	}
	return nil
}

func IsFunctionOutputSpan(span *cqrs.OtelSpan) bool {
	return span != nil &&
		span.Attributes != nil &&
		span.Attributes.IsFunctionOutput != nil &&
		*span.Attributes.IsFunctionOutput
}

func LatestFunctionOutputSpan(root *cqrs.OtelSpan) *cqrs.OtelSpan {
	var selected *cqrs.OtelSpan
	walkOtelSpans(root, func(span *cqrs.OtelSpan) {
		// Retries can produce more than one candidate; the latest one is the
		// final run output.
		if IsFunctionOutputSpan(span) && newerOtelSpan(span, selected) {
			selected = span
		}
	})
	return selected
}

func latestActiveSpan(root *cqrs.OtelSpan) *cqrs.OtelSpan {
	var selected *cqrs.OtelSpan
	walkOtelSpans(root, func(span *cqrs.OtelSpan) {
		status := stepStatusToRunStatus(OtelSpanStatus(span))
		if status != nil && !enums.RunStatusEnded(*status) && newerOtelSpan(span, selected) {
			selected = span
		}
	})
	return selected
}

func latestTerminalSpan(root *cqrs.OtelSpan) *cqrs.OtelSpan {
	var selected *cqrs.OtelSpan
	walkOtelSpans(root, func(span *cqrs.OtelSpan) {
		status := stepStatusToRunStatus(OtelSpanStatus(span))
		if status != nil && enums.RunStatusEnded(*status) && newerOtelSpan(span, selected) {
			selected = span
		}
	})
	return selected
}

func walkOtelSpans(root *cqrs.OtelSpan, fn func(*cqrs.OtelSpan)) {
	var walk func(*cqrs.OtelSpan)
	walk = func(span *cqrs.OtelSpan) {
		if span == nil {
			return
		}
		fn(span)
		for _, child := range span.Children {
			walk(child)
		}
	}
	walk(root)
}

func stepStatusToRunStatus(status *enums.StepStatus) *enums.RunStatus {
	if status == nil {
		return nil
	}
	runStatus := enums.StepStatusToRunStatus(*status)
	if runStatus == enums.RunStatusUnknown {
		return nil
	}
	return &runStatus
}

func DurationMS(startedAt *time.Time, endedAt *time.Time) *uint64 {
	if startedAt == nil || endedAt == nil {
		return nil
	}

	duration := endedAt.Sub(*startedAt) / time.Millisecond
	if duration < 0 {
		return nil
	}

	durationMs := uint64(duration)
	return &durationMs
}

func spanStartedAt(span *cqrs.OtelSpan, fallbackToStartTime bool) *time.Time {
	if span == nil {
		return nil
	}
	if startedAt := span.GetStartedAtTime(); startedAt != nil {
		return startedAt
	}
	if fallbackToStartTime && !span.StartTime.IsZero() {
		return &span.StartTime
	}
	return nil
}

func spanEndedAt(span *cqrs.OtelSpan) *time.Time {
	if span == nil {
		return nil
	}
	if endedAt := span.GetEndedAtTime(); endedAt != nil {
		return endedAt
	}
	if !span.EndTime.IsZero() {
		return &span.EndTime
	}
	return nil
}

func newerOtelSpan(candidate *cqrs.OtelSpan, current *cqrs.OtelSpan) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	return spanSortTime(candidate).After(spanSortTime(current))
}

func spanSortTime(span *cqrs.OtelSpan) time.Time {
	if endedAt := spanEndedAt(span); endedAt != nil {
		return *endedAt
	}
	if startedAt := spanStartedAt(span, true); startedAt != nil {
		return *startedAt
	}
	return time.Time{}
}
