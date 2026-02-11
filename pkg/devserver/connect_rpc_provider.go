package devserver

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/run"
	apiv2pb "github.com/inngest/inngest/proto/gen/api/v2"
	rpbv2 "github.com/inngest/inngest/proto/gen/run/v2"
	"github.com/oklog/ulid/v2"
)

var _ apiv2.ConnectRPCProvider = (*ConnectRPCProvider)(nil)

// ConnectRPCProvider implements apiv2.ConnectRPCProvider for the dev server.
// It uses the local cqrs.Manager to fetch data for ConnectRPC streaming operations.
type ConnectRPCProvider struct {
	data cqrs.Manager
}

func NewConnectRPCProvider(data cqrs.Manager) *ConnectRPCProvider {
	return &ConnectRPCProvider{data: data}
}

func (p *ConnectRPCProvider) GetRunData(ctx context.Context, accountID uuid.UUID, envID uuid.UUID, runID ulid.ULID) (*apiv2pb.RunData, error) {
	traceRun, err := p.data.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: runID})
	if err != nil {
		return nil, fmt.Errorf("failed to get trace run: %w", err)
	}

	fn, err := p.data.GetFunctionByInternalUUID(ctx, traceRun.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	apps, err := p.data.GetApps(ctx, consts.DevServerEnvID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get apps: %w", err)
	}

	var app *cqrs.App
	for _, a := range apps {
		if a.ID == fn.AppID {
			app = a
			break
		}
	}
	if app == nil {
		return nil, fmt.Errorf("app not found for function")
	}

	traceSpan, err := p.getRunTraceSpan(ctx, traceRun)
	if err != nil {
		//
		// Return data without trace if spans aren't available yet
		return &apiv2pb.RunData{
			Id:     runID.String(),
			Status: runStatusString(traceRun.Status),
			HasAi:  traceRun.HasAI,
			Function: &apiv2pb.FunctionInfo{
				Id:   fn.ID.String(),
				Name: fn.Name,
				Slug: fn.Slug,
				App: &apiv2pb.AppInfo{
					Name:       app.Name,
					ExternalId: app.ID.String(),
				},
			},
			Trace: nil,
		}, nil
	}

	return &apiv2pb.RunData{
		Id:     runID.String(),
		Status: runStatusString(traceRun.Status),
		HasAi:  traceRun.HasAI,
		Function: &apiv2pb.FunctionInfo{
			Id:   fn.ID.String(),
			Name: fn.Name,
			Slug: fn.Slug,
			App: &apiv2pb.AppInfo{
				Name:       app.Name,
				ExternalId: app.ID.String(),
			},
		},
		Trace: traceSpan,
	}, nil
}

func (p *ConnectRPCProvider) getRunTraceSpan(ctx context.Context, traceRun *cqrs.TraceRun) (*apiv2pb.RunTraceSpan, error) {
	runID, err := ulid.Parse(traceRun.RunID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse run ID: %w", err)
	}

	spans, err := p.data.GetTraceSpansByRun(ctx, cqrs.TraceRunIdentifier{
		AccountID:   traceRun.AccountID,
		WorkspaceID: traceRun.WorkspaceID,
		AppID:       traceRun.AppID,
		FunctionID:  traceRun.FunctionID,
		TraceID:     traceRun.TraceID,
		RunID:       runID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get trace spans: %w", err)
	}

	if len(spans) == 0 {
		return nil, fmt.Errorf("no spans found")
	}

	tree, err := run.NewRunTree(run.RunTreeOpts{
		AccountID:   traceRun.AccountID,
		WorkspaceID: traceRun.WorkspaceID,
		AppID:       traceRun.AppID,
		FunctionID:  traceRun.FunctionID,
		RunID:       runID,
		Spans:       spans,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build run tree: %w", err)
	}

	rootSpan, err := tree.ToRunSpan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to run span: %w", err)
	}

	return runSpanToRunTraceSpan(rootSpan), nil
}

func runStatusString(status enums.RunStatus) string {
	switch status {
	case enums.RunStatusScheduled:
		return "QUEUED"
	case enums.RunStatusRunning:
		return "RUNNING"
	case enums.RunStatusCompleted:
		return "COMPLETED"
	case enums.RunStatusFailed, enums.RunStatusOverflowed:
		return "FAILED"
	case enums.RunStatusCancelled:
		return "CANCELLED"
	case enums.RunStatusSkipped:
		return "SKIPPED"
	default:
		return "UNKNOWN"
	}
}

func runSpanToRunTraceSpan(pb *rpbv2.RunSpan) *apiv2pb.RunTraceSpan {
	if pb == nil {
		return nil
	}

	span := &apiv2pb.RunTraceSpan{
		Name:       pb.GetName(),
		Status:     mapSpanStatus(pb.GetStatus()),
		QueuedAt:   formatTime(pb.GetQueuedAt().AsTime()),
		IsRoot:     pb.GetIsRoot(),
		SpanId:     pb.GetSpanId(),
		IsUserland: pb.GetIsUserland(),
	}

	if pb.Attempts != 0 {
		val := int32(pb.Attempts)
		span.Attempts = &val
	}
	if pb.GetStartedAt() != nil {
		span.StartedAt = formatTimePtr(pb.GetStartedAt().AsTime())
	}
	if pb.GetEndedAt() != nil {
		span.EndedAt = formatTimePtr(pb.GetEndedAt().AsTime())
	}
	if pb.OutputId != nil {
		span.OutputId = pb.OutputId
	}
	if pb.StepId != nil {
		span.StepId = pb.StepId
	}
	if pb.StepOp != nil {
		op := mapStepOp(*pb.StepOp)
		span.StepOp = &op
	}
	if pb.StepInfo != nil {
		span.StepInfo = mapStepInfo(pb.StepInfo)
	}
	if pb.UserlandSpan != nil {
		span.UserlandSpan = mapUserlandSpan(pb.UserlandSpan)
	}

	if len(pb.Children) > 0 {
		span.ChildrenSpans = make([]*apiv2pb.RunTraceSpan, 0, len(pb.Children))
		for _, c := range pb.Children {
			span.ChildrenSpans = append(span.ChildrenSpans, runSpanToRunTraceSpan(c))
		}
	}

	return span
}

func mapSpanStatus(s rpbv2.SpanStatus) apiv2pb.RunTraceSpanStatus {
	switch s {
	case rpbv2.SpanStatus_RUNNING:
		return apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_RUNNING
	case rpbv2.SpanStatus_WAITING:
		return apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_WAITING
	case rpbv2.SpanStatus_COMPLETED, rpbv2.SpanStatus_OK:
		return apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_COMPLETED
	case rpbv2.SpanStatus_FAILED, rpbv2.SpanStatus_ERORR:
		return apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_FAILED
	case rpbv2.SpanStatus_CANCELLED:
		return apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_CANCELLED
	case rpbv2.SpanStatus_QUEUED, rpbv2.SpanStatus_SCHEDULED:
		return apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_QUEUED
	default:
		return apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_UNSPECIFIED
	}
}

func mapStepOp(op rpbv2.SpanStepOp) apiv2pb.StepOp {
	switch op {
	case rpbv2.SpanStepOp_INVOKE:
		return apiv2pb.StepOp_STEP_OP_INVOKE
	case rpbv2.SpanStepOp_SLEEP:
		return apiv2pb.StepOp_STEP_OP_SLEEP
	case rpbv2.SpanStepOp_WAIT_FOR_EVENT:
		return apiv2pb.StepOp_STEP_OP_WAIT_FOR_EVENT
	case rpbv2.SpanStepOp_AI_GATEWAY:
		return apiv2pb.StepOp_STEP_OP_AI_GATEWAY
	case rpbv2.SpanStepOp_WAIT_FOR_SIGNAL:
		return apiv2pb.StepOp_STEP_OP_WAIT_FOR_SIGNAL
	default:
		return apiv2pb.StepOp_STEP_OP_RUN
	}
}

func mapStepInfo(si *rpbv2.StepInfo) *apiv2pb.StepInfo {
	if si == nil {
		return nil
	}

	if s := si.GetInvoke(); s != nil {
		out := &apiv2pb.InvokeStepInfo{
			TriggeringEventId: s.GetTriggeringEventId(),
			FunctionId:        s.GetFunctionId(),
			Timeout:           formatTime(s.GetTimeout().AsTime()),
		}
		if s.ReturnEventId != nil {
			out.ReturnEventId = s.ReturnEventId
		}
		if s.RunId != nil {
			out.RunId = s.RunId
		}
		if s.TimedOut != nil {
			out.TimedOut = s.TimedOut
		}
		return &apiv2pb.StepInfo{Info: &apiv2pb.StepInfo_Invoke{Invoke: out}}
	}

	if s := si.GetSleep(); s != nil {
		return &apiv2pb.StepInfo{Info: &apiv2pb.StepInfo_Sleep{Sleep: &apiv2pb.SleepStepInfo{
			SleepUntil: formatTime(s.GetSleepUntil().AsTime()),
		}}}
	}

	if s := si.GetWait(); s != nil {
		out := &apiv2pb.WaitForEventStepInfo{
			EventName: s.GetEventName(),
			Timeout:   formatTime(s.GetTimeout().AsTime()),
		}
		if s.Expression != nil {
			out.Expression = s.Expression
		}
		if s.FoundEventId != nil {
			out.FoundEventId = s.FoundEventId
		}
		if s.TimedOut != nil {
			out.TimedOut = s.TimedOut
		}
		return &apiv2pb.StepInfo{Info: &apiv2pb.StepInfo_WaitForEvent{WaitForEvent: out}}
	}

	if s := si.GetRun(); s != nil {
		out := &apiv2pb.RunStepInfo{}
		if s.Type != nil {
			out.Type = s.Type
		}
		return &apiv2pb.StepInfo{Info: &apiv2pb.StepInfo_Run{Run: out}}
	}

	if s := si.GetWaitForSignal(); s != nil {
		out := &apiv2pb.WaitForSignalStepInfo{
			Signal:  s.GetSignal(),
			Timeout: formatTime(s.GetTimeout().AsTime()),
		}
		if s.TimedOut != nil {
			out.TimedOut = s.TimedOut
		}
		return &apiv2pb.StepInfo{Info: &apiv2pb.StepInfo_WaitForSignal{WaitForSignal: out}}
	}

	return nil
}

func mapUserlandSpan(pb *rpbv2.UserlandSpan) *apiv2pb.UserlandSpan {
	if pb == nil {
		return nil
	}
	out := &apiv2pb.UserlandSpan{}
	if pb.SpanName != "" {
		out.SpanName = &pb.SpanName
	}
	if pb.SpanKind != "" {
		out.SpanKind = &pb.SpanKind
	}
	if pb.ServiceName != nil {
		out.ServiceName = pb.ServiceName
	}
	if pb.ScopeName != nil {
		out.ScopeName = pb.ScopeName
	}
	if pb.ScopeVersion != nil {
		out.ScopeVersion = pb.ScopeVersion
	}
	if pb.SpanAttrs != nil {
		s := string(pb.SpanAttrs)
		out.SpanAttrs = &s
	}
	if pb.ResourceAttrs != nil {
		s := string(pb.ResourceAttrs)
		out.ResourceAttrs = &s
	}
	return out
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func formatTimePtr(t time.Time) *string {
	s := formatTime(t)
	return &s
}
