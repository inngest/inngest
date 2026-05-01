package apiv2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

const (
	defaultTraceMaxDepth uint32 = 5
	maxTraceMaxDepth     uint32 = 10
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

	maxDepth, err := traceMaxDepth(req.MaxDepth)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidRequest, err.Error())
	}

	rootSpan, err := s.traces.GetSpansByRunID(ctx, runID)
	if err != nil {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Trace not found")
	}

	trace, err := toFunctionTrace(ctx, s.traces, rootSpan, req.GetIncludeOutput(), maxDepth)
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to build trace response")
	}

	return &apiv2.GetFunctionTraceResponse{
		Data:     trace,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

func (s *Service) GetFunctionTraceSpan(ctx context.Context, req *apiv2.GetFunctionTraceSpanRequest) (*apiv2.GetFunctionTraceSpanResponse, error) {
	if req.RunId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Run ID is required")
	}
	if req.SpanId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Span ID is required")
	}

	if s.traces == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get function trace span is not yet implemented")
	}

	runID, err := ulid.Parse(req.RunId)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Run ID must be a valid ULID")
	}

	maxDepth, err := traceMaxDepth(req.MaxDepth)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidRequest, err.Error())
	}

	rootSpan, err := s.traces.GetSpansByRunID(ctx, runID)
	if err != nil {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Trace not found")
	}

	span, err := loader.ConvertRunSpan(ctx, rootSpan)
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to build trace response")
	}

	target := findTraceSpan(span, req.SpanId)
	if target == nil {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Trace span not found")
	}

	data, err := toTraceSpan(ctx, s.traces, target, req.GetIncludeOutput(), 1, maxDepth)
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to build trace response")
	}

	return &apiv2.GetFunctionTraceSpanResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

func toFunctionRun(run *cqrs.FunctionRun, fn inngest.DeployedFunction, includeOutput bool) *apiv2.FunctionRun {
	queuedAt := timestamppb.New(ulid.Time(run.RunID.Time()))
	startedAt := timestamppb.New(run.RunStartedAt)

	result := &apiv2.FunctionRun{
		Id: run.RunID.String(),
		Function: &apiv2.FunctionRef{
			Id:   functionRefID(fn),
			Name: fn.Function.Name,
		},
		App: &apiv2.AppRef{
			Id: appRefID(fn),
		},
		Status:    toFunctionRunStatus(run.Status),
		QueuedAt:  queuedAt,
		StartedAt: startedAt,
		Trigger: &apiv2.RunTrigger{
			EventIds: []string{run.EventID.String()},
			IsBatch:  run.BatchID != nil,
		},
	}

	if run.BatchID != nil {
		result.Trigger.BatchId = optionalString(run.BatchID.String())
	}

	if run.Cron != nil {
		result.Trigger.CronSchedule = optionalString(*run.Cron)
	}

	if run.EndedAt != nil {
		result.EndedAt = timestamppb.New(*run.EndedAt)
		duration := uint64(run.EndedAt.Sub(run.RunStartedAt) / time.Millisecond)
		result.DurationMs = &duration
	}

	if includeOutput {
		result.Output = jsonToStruct(run.Output)
	}

	return result
}

func functionRefID(fn inngest.DeployedFunction) string {
	if fn.Function.Slug != "" {
		return fn.Function.Slug
	}
	if fn.AppName != "" {
		return strings.TrimPrefix(fn.Slug, fn.AppName+"-")
	}
	return fn.Slug
}

func appRefID(fn inngest.DeployedFunction) string {
	if fn.AppName != "" {
		return fn.AppName
	}
	return fn.AppID.String()
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

func traceMaxDepth(value *uint32) (uint32, error) {
	if value == nil {
		return defaultTraceMaxDepth, nil
	}
	if *value == 0 || *value > maxTraceMaxDepth {
		return 0, fmt.Errorf("max_depth must be between 1 and %d", maxTraceMaxDepth)
	}
	return *value, nil
}

func toFunctionTrace(ctx context.Context, reader FunctionTraceReader, root *cqrs.OtelSpan, includeOutput bool, maxDepth uint32) (*apiv2.FunctionTrace, error) {
	span, err := loader.ConvertRunSpan(ctx, root)
	if err != nil {
		return nil, err
	}

	rootSpan, err := toTraceSpan(ctx, reader, span, includeOutput, 1, maxDepth)
	if err != nil {
		return nil, err
	}

	return &apiv2.FunctionTrace{
		RunId:    root.RunID.String(),
		RootSpan: rootSpan,
	}, nil
}

func toTraceSpan(ctx context.Context, reader FunctionTraceReader, span *models.RunTraceSpan, includeOutput bool, depth uint32, maxDepth uint32) (*apiv2.TraceSpan, error) {
	result := &apiv2.TraceSpan{
		Id:         span.SpanID,
		Name:       span.Name,
		Status:     toTraceSpanStatus(span.Status),
		DurationMs: traceDuration(span),
		QueuedAt:   timestamppb.New(span.QueuedAt),
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

	result.Metadata = toTraceSpanMetadata(span.Metadata)

	if includeOutput && span.OutputID != nil && *span.OutputID != "" {
		output, err := loadTraceOutput(ctx, reader, *span.OutputID)
		if err != nil {
			return nil, err
		}

		result.Input = output.input
		result.Output = output.output
	}

	if depth >= maxDepth {
		result.ChildrenTruncated = len(span.ChildrenSpans) > 0
		return result, nil
	}

	children := make([]*apiv2.TraceSpan, 0, len(span.ChildrenSpans))
	for _, child := range span.ChildrenSpans {
		mapped, err := toTraceSpan(ctx, reader, child, includeOutput, depth+1, maxDepth)
		if err != nil {
			return nil, err
		}
		children = append(children, mapped)
	}
	result.Children = children

	return result, nil
}

func findTraceSpan(root *models.RunTraceSpan, spanID string) *models.RunTraceSpan {
	if root == nil {
		return nil
	}
	if root.SpanID == spanID {
		return root
	}
	for _, child := range root.ChildrenSpans {
		if found := findTraceSpan(child, spanID); found != nil {
			return found
		}
	}
	return nil
}

func toTraceSpanStatus(status models.RunTraceSpanStatus) apiv2.TraceSpanStatus {
	switch status {
	case models.RunTraceSpanStatusCompleted:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_COMPLETED
	case models.RunTraceSpanStatusFailed:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_FAILED
	case models.RunTraceSpanStatusCancelled, models.RunTraceSpanStatusSkipped:
		// TODO(api-v2): Add CANCELLED and SKIPPED trace status enum values to the v2 contract.
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_UNKNOWN
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

func toTraceSpanMetadata(metadata []*models.SpanMetadata) []*apiv2.TraceSpanMetadata {
	if len(metadata) == 0 {
		return nil
	}

	result := make([]*apiv2.TraceSpanMetadata, 0, len(metadata))
	for _, item := range metadata {
		if item == nil {
			continue
		}

		result = append(result, &apiv2.TraceSpanMetadata{
			Scope:     item.Scope.String(),
			Kind:      item.Kind.String(),
			Values:    toTraceSpanMetadataValues(item.Values),
			UpdatedAt: timestamppb.New(item.UpdatedAt),
		})
	}
	return result
}

func toTraceSpanMetadataValues(values map[string]json.RawMessage) map[string]string {
	if len(values) == 0 {
		return nil
	}

	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = string(value)
	}
	return result
}

func traceDuration(span *models.RunTraceSpan) *uint64 {
	if span.Duration != nil && *span.Duration >= 0 {
		value := uint64(*span.Duration)
		return &value
	}

	if span.StartedAt != nil && span.EndedAt != nil {
		value := uint64(span.EndedAt.Sub(*span.StartedAt) / time.Millisecond)
		return &value
	}

	return nil
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
