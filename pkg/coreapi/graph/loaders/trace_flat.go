// convertFlatSpanToGQL is the RunTrace converter for spans sourced from a
// flat, non-dynamic span tree (DuckDB's inngest.run_trace_spans). Unlike
// convertRunSpanToGQL (trace.go), it carries no logic for the old
// dynamic/fragment-merged
// model: no discovery-step hiding, no SDK "inngest.execution" wrapper
// collapsing, no userland-span reparenting, no metadata-span rollup — none
// of those cases can occur in a flat tree, so there are no guard clauses
// for them here. opcodeToGQL/stepStatusToGQL are shared with the other
// converter since they're pure enum mappers with no rollup assumptions.
package loader

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
)

func convertFlatSpanToGQL(ctx context.Context, span *cqrs.OtelSpan) (*models.RunTraceSpan, error) {
	status := models.RunTraceSpanStatusRunning
	if gqlStatus := (&traceReader{}).stepStatusToGQL(span.Attributes.DynamicStatus); gqlStatus != nil {
		status = *gqlStatus
	}

	attempts := span.GetAttempts()
	gqlSpan := &models.RunTraceSpan{
		AppID:          span.GetAppID(),
		Attempts:       &attempts,
		GroupID:        span.Attributes.GroupID,
		EndedAt:        span.GetEndedAtTime(),
		FunctionID:     span.GetFunctionID(),
		IsRoot:         span.GetIsRoot(),
		Name:           span.GetStepName(),
		OutputID:       span.GetOutputID(),
		ParentSpanID:   span.GetParentSpanID(),
		QueuedAt:       span.GetQueuedAtTime(),
		RunID:          span.GetRunID(),
		SpanID:         span.GetSpanID(),
		StartedAt:      span.GetStartedAtTime(),
		ScheduledAt:    span.GetScheduledAtTime(),
		Status:         status,
		TraceID:        span.GetTraceID(),
		DebugRunID:     span.GetDebugRunID(),
		DebugSessionID: span.GetDebugSessionID(),
		SpanTypeName:   span.Name,
	}

	// Duration is nil ("still running", per the schema's own doc comment)
	// until the span has actually ended. Unlike convertRunSpanToGQL, the
	// flat model has no "IsUserland" concept and no fragment-merge quirk
	// where a not-yet-ended span carries a stale EndedAt (see that
	// function's else-branch comment) — a flat span's own EndedAt is
	// already correct for its status, so no clearing is needed here.
	if models.RunTraceEnded(gqlSpan.Status) {
		startedAt := span.GetStartedAtTime()
		endedAt := span.GetEndedAtTime()
		if startedAt != nil && endedAt != nil {
			dur := int(endedAt.Sub(*startedAt).Milliseconds())
			gqlSpan.Duration = &dur
		}
	}

	if span.Attributes.SkipReason != nil {
		reason := span.Attributes.SkipReason.String()
		gqlSpan.SkipReason = &reason
	}
	if span.Attributes.SkipExistingRunID != nil {
		gqlSpan.SkipExistingRunID = span.Attributes.SkipExistingRunID
	}
	if span.Attributes.ResponseStatusCode != nil && span.Attributes.ResponseHeaders != nil {
		gqlSpan.Response = &models.RunTraceSpanResponseInfo{
			StatusCode: *span.Attributes.ResponseStatusCode,
			Headers:    *span.Attributes.ResponseHeaders,
		}
	}
	if span.Attributes.StepID != nil {
		gqlSpan.StepID = span.Attributes.StepID
	}
	if span.Attributes.StepOp != nil {
		gqlSpan.StepOp = (&traceReader{}).opcodeToGQL(span.Attributes.StepOp)
	}

	// convertRunSpanToGQL's equivalent has a third, first-checked branch for
	// gqlSpan.Name == FinalizationSpanName — a name only that rollup-era
	// converter itself ever synthesizes (see its own doc comment), never a
	// real span name, so it can't occur here and is deliberately omitted.
	if span.Attributes.StepRunType != nil {
		gqlSpan.StepType = *span.Attributes.StepRunType
	} else if gqlSpan.StepOp != nil {
		gqlSpan.StepType = gqlSpan.StepOp.String()
	}

	if gqlSpan.StepOp != nil {
		switch *gqlSpan.StepOp {
		case models.StepOpRun:
			gqlSpan.StepInfo = &models.RunStepInfo{Type: span.Attributes.StepRunType}
		case models.StepOpInvoke:
			si := &models.InvokeStepInfo{
				TimedOut:      span.Attributes.StepWaitExpired,
				ReturnEventID: span.Attributes.StepInvokeFinishEventID,
				RunID:         span.Attributes.StepInvokeRunID,
			}
			if span.Attributes.StepInvokeTriggerEventID != nil {
				si.TriggeringEventID = *span.Attributes.StepInvokeTriggerEventID
			}
			if span.Attributes.StepInvokeFunctionID != nil {
				si.FunctionID = *span.Attributes.StepInvokeFunctionID
			}
			if span.Attributes.StepWaitExpiry != nil {
				si.Timeout = *span.Attributes.StepWaitExpiry
			}
			gqlSpan.StepInfo = si
		case models.StepOpSleep:
			if span.Attributes.StepSleepDuration != nil {
				gqlSpan.StepInfo = &models.SleepStepInfo{
					SleepUntil: span.GetQueuedAtTime().Add(*span.Attributes.StepSleepDuration),
				}
			}
		case models.StepOpWaitForEvent:
			si := &models.WaitForEventStepInfo{
				Expression:   span.Attributes.StepWaitForEventIf,
				TimedOut:     span.Attributes.StepWaitExpired,
				FoundEventID: span.Attributes.StepWaitForEventMatchedID,
			}
			if span.Attributes.StepWaitForEventName != nil {
				si.EventName = *span.Attributes.StepWaitForEventName
			}
			if span.Attributes.StepWaitExpiry != nil {
				si.Timeout = *span.Attributes.StepWaitExpiry
			}
			gqlSpan.StepInfo = si
		case models.StepOpWaitForSignal:
			si := &models.WaitForSignalStepInfo{TimedOut: span.Attributes.StepWaitExpired}
			if span.Attributes.StepSignalName != nil {
				si.Signal = *span.Attributes.StepSignalName
			}
			if span.Attributes.StepWaitExpiry != nil {
				si.Timeout = *span.Attributes.StepWaitExpiry
			}
			gqlSpan.StepInfo = si
		}
	}

	gqlSpan.ChildrenSpans = make([]*models.RunTraceSpan, 0, len(span.Children))
	for _, cs := range span.Children {
		child, err := convertFlatSpanToGQL(ctx, cs)
		if err != nil {
			return nil, fmt.Errorf("error converting child span: %w", err)
		}
		gqlSpan.ChildrenSpans = append(gqlSpan.ChildrenSpans, child)
	}

	return gqlSpan, nil
}

// flatSpanSource is implemented by a cqrs.Manager whose GetSpansByRunID
// returns spans from a flat (non-dynamic) tree — pkg/cqrs/duckdbquery.Manager
// implements it. A one-method structural interface rather than an import of
// the concrete type, so this package stays decoupled from duckdbquery.
type flatSpanSource interface{ FlatSpans() bool }
