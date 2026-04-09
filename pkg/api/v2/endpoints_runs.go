package apiv2

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) GetFunctionRun(ctx context.Context, req *apiv2.GetFunctionRunRequest) (*apiv2.GetFunctionRunResponse, error) {
	if req.RunId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Run ID is required")
	}

	if s.runs == nil || s.functions == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get function run is not yet implemented")
	}

	runID, err := ulid.Parse(req.RunId)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Run ID must be a valid ULID")
	}

	run, err := s.runs.GetFunctionRun(ctx, runID)
	if err != nil {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Run not found")
	}

	fn, err := s.functions.GetFunction(ctx, run.FunctionID.String())
	if err != nil {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Function not found")
	}

	return &apiv2.GetFunctionRunResponse{
		Data:     toFunctionRun(run, fn, req.GetIncludeOutput()),
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

func (s *Service) GetFunctionTrace(ctx context.Context, req *apiv2.GetFunctionTraceRequest) (*apiv2.GetFunctionTraceResponse, error) {
	if req.RunId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Run ID is required")
	}

	if s.traces == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get function trace is not yet implemented")
	}

	runID, err := ulid.Parse(req.RunId)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Run ID must be a valid ULID")
	}

	rootSpan, err := s.traces.GetSpansByRunID(ctx, runID)
	if err != nil {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Trace not found")
	}

	trace, err := toFunctionTrace(ctx, s.traces, rootSpan, req.GetIncludeOutput())
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to build trace response")
	}

	return &apiv2.GetFunctionTraceResponse{
		Data:     trace,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

func toFunctionRun(run *cqrs.FunctionRun, fn inngest.DeployedFunction, includeOutput bool) *apiv2.FunctionRun {
	startedAt := timestamppb.New(run.RunStartedAt)

	result := &apiv2.FunctionRun{
		Id:      run.RunID.String(),
		TraceId: "",
		Function: &apiv2.FunctionRef{
			Id:   fn.ID.String(),
			Slug: fn.Slug,
			Name: fn.Function.Name,
		},
		App: &apiv2.AppRef{
			Id:         fn.AppID.String(),
			ExternalId: fn.AppID.String(),
			Name:       "",
		},
		Status:    toFunctionRunStatus(run.Status),
		QueuedAt:  startedAt,
		StartedAt: startedAt,
		Trigger: &apiv2.RunTrigger{
			EventIds: []string{run.EventID.String()},
			IsBatch:  run.BatchID != nil,
			SourceId: optionalString(run.EventID.String()),
		},
		HasAi: false,
	}

	if run.BatchID != nil {
		result.Trigger.BatchId = optionalString(run.BatchID.String())
	}

	if run.Cron != nil {
		result.Trigger.CronSchedule = optionalString(*run.Cron)
	}

	if run.EndedAt != nil {
		result.EndedAt = timestamppb.New(*run.EndedAt)
		result.DurationMs = uint64(run.EndedAt.Sub(run.RunStartedAt) / time.Millisecond)
	}

	if includeOutput {
		result.Output = jsonToStruct(run.Output)
	}

	return result
}

func toFunctionRunStatus(status enums.RunStatus) apiv2.FunctionRunStatus {
	switch status {
	case enums.RunStatusCompleted:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED
	case enums.RunStatusFailed:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_FAILED
	case enums.RunStatusCancelled:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_CANCELLED
	case enums.RunStatusRunning:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_RUNNING
	default:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_QUEUED
	}
}

func jsonToStruct(raw json.RawMessage) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}

	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}

	result, err := structpb.NewStruct(value)
	if err != nil {
		return nil
	}

	return result
}

func optionalString(value string) *string {
	return &value
}

func toFunctionTrace(ctx context.Context, reader FunctionTraceReader, root *cqrs.OtelSpan, includeOutput bool) (*apiv2.FunctionTrace, error) {
	span, err := loader.ConvertRunSpan(ctx, root)
	if err != nil {
		return nil, err
	}

	rootSpan, err := toTraceSpan(ctx, reader, span, includeOutput)
	if err != nil {
		return nil, err
	}

	return &apiv2.FunctionTrace{
		RunId:    root.RunID.String(),
		TraceId:  span.TraceID,
		RootSpan: rootSpan,
	}, nil
}

func toTraceSpan(ctx context.Context, reader FunctionTraceReader, span *models.RunTraceSpan, includeOutput bool) (*apiv2.TraceSpan, error) {
	result := &apiv2.TraceSpan{
		SpanId:     span.SpanID,
		Name:       span.Name,
		Status:     toTraceSpanStatus(span.Status),
		DurationMs: traceDuration(span),
		QueuedAt:   timestamppb.New(span.QueuedAt),
		IsRoot:     span.IsRoot,
		IsUserland: span.IsUserland,
	}

	if span.StepOp != nil {
		stepOp := toTraceStepOp(*span.StepOp)
		result.StepOp = &stepOp
	}

	if span.StepID != nil && *span.StepID != "" {
		result.StepId = span.StepID
	}

	if span.StartedAt != nil {
		result.StartedAt = timestamppb.New(*span.StartedAt)
	}

	if span.EndedAt != nil {
		result.EndedAt = timestamppb.New(*span.EndedAt)
	}

	if span.Response != nil {
		result.Response = &apiv2.TraceResponse{
			StatusCode: int32(span.Response.StatusCode),
			Headers:    compactHeaders(span.Response.Headers),
		}
	}

	if includeOutput && span.OutputID != nil && *span.OutputID != "" {
		output, err := loadTraceOutput(ctx, reader, *span.OutputID)
		if err != nil {
			return nil, err
		}

		result.Input = output.input
		result.Output = output.output
	}

	children := make([]*apiv2.TraceSpan, 0, len(span.ChildrenSpans))
	for _, child := range span.ChildrenSpans {
		mapped, err := toTraceSpan(ctx, reader, child, includeOutput)
		if err != nil {
			return nil, err
		}
		children = append(children, mapped)
	}
	result.Children = children

	return result, nil
}

func toTraceSpanStatus(status models.RunTraceSpanStatus) apiv2.TraceSpanStatus {
	switch status {
	case models.RunTraceSpanStatusCompleted:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_COMPLETED
	case models.RunTraceSpanStatusFailed, models.RunTraceSpanStatusCancelled, models.RunTraceSpanStatusSkipped:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_FAILED
	case models.RunTraceSpanStatusWaiting, models.RunTraceSpanStatusQueued:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_WAITING
	default:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_RUNNING
	}
}

func toTraceStepOp(stepOp models.StepOp) apiv2.TraceStepOp {
	switch stepOp {
	case models.StepOpRun:
		return apiv2.TraceStepOp_TRACE_STEP_OP_RUN
	case models.StepOpSleep:
		return apiv2.TraceStepOp_TRACE_STEP_OP_SLEEP
	case models.StepOpWaitForEvent:
		return apiv2.TraceStepOp_TRACE_STEP_OP_WAIT_FOR_EVENT
	case models.StepOpInvoke:
		return apiv2.TraceStepOp_TRACE_STEP_OP_INVOKE
	case models.StepOpAiGateway:
		return apiv2.TraceStepOp_TRACE_STEP_OP_AI_GATEWAY
	default:
		return apiv2.TraceStepOp_TRACE_STEP_OP_UNSPECIFIED
	}
}

func traceDuration(span *models.RunTraceSpan) uint64 {
	if span.Duration != nil && *span.Duration >= 0 {
		return uint64(*span.Duration)
	}

	if span.StartedAt != nil && span.EndedAt != nil {
		return uint64(span.EndedAt.Sub(*span.StartedAt) / time.Millisecond)
	}

	return 0
}

type traceOutput struct {
	input  *structpb.Struct
	output *structpb.Struct
}

func loadTraceOutput(ctx context.Context, reader FunctionTraceReader, encodedID string) (*traceOutput, error) {
	var id cqrs.SpanIdentifier
	if err := id.Decode(encodedID); err != nil {
		return nil, err
	}

	data, err := reader.GetSpanOutput(ctx, id)
	if err != nil {
		return nil, err
	}

	return &traceOutput{
		input:  jsonToStruct(json.RawMessage(data.Input)),
		output: jsonToStruct(json.RawMessage(data.Data)),
	}, nil
}

func compactHeaders(input map[string][]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	result := make(map[string]string, len(input))
	for key, values := range input {
		if len(values) == 0 {
			continue
		}
		result[key] = values[0]
	}

	return result
}
