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
	defaultEventRunsLimit = 20
	maxEventRunsLimit     = 40
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

	includeOutput := req.GetIncludeOutput()
	run, err := s.runs.GetFunctionRun(ctx, runID, GetFunctionRunOpts{
		IncludeOutput: includeOutput,
	})
	if err != nil {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Run not found")
	}

	fn, err := s.functions.GetFunction(ctx, run.FunctionID.String())
	if err != nil {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Function not found")
	}

	data := toFunctionRun(run, fn)
	if includeOutput {
		data.Output = s.traceRunOutput(ctx, runID)
	}

	return &apiv2.GetFunctionRunResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

func (s *Service) GetEventRuns(ctx context.Context, req *apiv2.GetEventRunsRequest) (*apiv2.GetEventRunsResponse, error) {
	if req.EventId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Event ID is required")
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_GetEventRuns_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no event runs were fetched.")
	}

	if s.runList == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get event runs is not yet implemented")
	}

	eventID, err := ulid.Parse(req.EventId)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Event ID must be a valid ULID")
	}

	cursor, limit, err := runsPageOpts(req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	result, err := s.runList.GetRuns(ctx, GetRunsOpts{
		EventID:       eventID,
		Cursor:        cursor,
		Limit:         limit,
		IncludeOutput: req.GetIncludeOutput(),
	})
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch event runs")
	}
	if result == nil {
		result = &GetRunsResult{}
	}

	data := make([]*apiv2.FunctionRun, 0, len(result.Runs))
	for _, run := range result.Runs {
		item := toAPIRunListItem(run)
		s.hydrateRunListItemFromTrace(ctx, item, run.RunID, req.GetIncludeOutput())
		data = append(data, item)
	}

	return &apiv2.GetEventRunsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     runsPage(result.Runs, limit, result.HasMore),
	}, nil
}

func runsPageOpts(cursor string, requestedLimit int32) (ulid.ULID, int, error) {
	limit := int(requestedLimit)
	if limit == 0 {
		limit = defaultEventRunsLimit
	}
	if limit < 1 {
		return ulid.Zero, 0, fmt.Errorf("Limit must be at least 1")
	}
	if limit > maxEventRunsLimit {
		return ulid.Zero, 0, fmt.Errorf("Limit cannot exceed %d", maxEventRunsLimit)
	}

	parsedCursor := ulid.Zero
	if cursor != "" {
		decodedCursor, err := decodeEventRunsCursor(cursor)
		if err != nil {
			return ulid.Zero, 0, fmt.Errorf("Cursor is invalid")
		}
		parsedCursor = decodedCursor
	}

	return parsedCursor, limit, nil
}

func runsPage(runs []*RunListItem, limit int, hasMore bool) *apiv2.Page {
	page := &apiv2.Page{
		HasMore: hasMore,
		Limit:   int32(limit),
	}
	if hasMore && len(runs) > 0 {
		nextCursor := encodeEventRunsCursor(runs[len(runs)-1].RunID)
		page.Cursor = &nextCursor
	}
	return page
}

func encodeEventRunsCursor(runID ulid.ULID) string {
	return runID.String()
}

func decodeEventRunsCursor(cursor string) (ulid.ULID, error) {
	return ulid.Parse(cursor)
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

func toFunctionRun(run *cqrs.FunctionRun, fn inngest.DeployedFunction) *apiv2.FunctionRun {
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

	return result
}

func toAPIRunListItem(run *RunListItem) *apiv2.FunctionRun {
	queuedAt := timestamppb.New(ulid.Time(run.RunID.Time()))
	startedAt := timestamppb.New(run.RunStartedAt)

	result := &apiv2.FunctionRun{
		Id: run.RunID.String(),
		Function: &apiv2.FunctionRef{
			Id:   run.FunctionID,
			Name: run.FunctionName,
		},
		App: &apiv2.AppRef{
			Id: run.AppID,
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

	if len(run.Output) > 0 {
		result.Output = jsonToStruct(run.Output)
	}

	return result
}

func (s *Service) hydrateRunListItemFromTrace(ctx context.Context, item *apiv2.FunctionRun, runID ulid.ULID, includeOutput bool) {
	if s.traces == nil {
		return
	}

	root, err := s.traces.GetSpansByRunID(ctx, runID)
	if err != nil || root == nil {
		return
	}

	span, err := loader.ConvertRunSpan(ctx, root)
	if err != nil || span == nil {
		return
	}

	item.Status = toFunctionRunStatusFromTrace(span.Status)
	if span.StartedAt != nil {
		item.StartedAt = timestamppb.New(*span.StartedAt)
	}
	if span.EndedAt != nil {
		item.EndedAt = timestamppb.New(*span.EndedAt)
	}
	if duration := traceDuration(span); duration != nil {
		item.DurationMs = duration
	}

	if includeOutput {
		outputID := traceRunOutputID(span)
		if outputID == nil {
			return
		}
		output, err := loadTraceOutput(ctx, s.traces, *outputID)
		if err == nil && output != nil {
			item.Output = output.output
		}
	}
}

func traceRunOutputID(span *models.RunTraceSpan) *string {
	if span == nil {
		return nil
	}

	var fallback *string
	var walk func(*models.RunTraceSpan) *string
	walk = func(current *models.RunTraceSpan) *string {
		if current.OutputID != nil && *current.OutputID != "" {
			if current.Name == loader.FinalizationSpanName {
				return current.OutputID
			}
			fallback = current.OutputID
		}

		for _, child := range current.ChildrenSpans {
			if outputID := walk(child); outputID != nil {
				return outputID
			}
		}

		return nil
	}

	if outputID := walk(span); outputID != nil {
		return outputID
	}
	return fallback
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

func toFunctionRunStatusFromTrace(status models.RunTraceSpanStatus) apiv2.FunctionRunStatus {
	switch status {
	case models.RunTraceSpanStatusCompleted:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED
	case models.RunTraceSpanStatusFailed:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_FAILED
	case models.RunTraceSpanStatusCancelled:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_CANCELLED
	case models.RunTraceSpanStatusRunning:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_RUNNING
	default:
		return apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_QUEUED
	}
}

func jsonToStruct(raw json.RawMessage) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}

	if object, ok := value.(map[string]any); ok {
		result, err := structpb.NewStruct(object)
		if err != nil {
			return nil
		}

		return result
	}

	wrapped, err := structpb.NewValue(value)
	if err != nil {
		return nil
	}

	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"data": wrapped,
		},
	}
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
		RootSpan: rootSpan,
	}, nil
}

func toTraceSpan(ctx context.Context, reader FunctionTraceReader, span *models.RunTraceSpan, includeOutput bool) (*apiv2.TraceSpan, error) {
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
	case models.RunTraceSpanStatusFailed:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_FAILED
	case models.RunTraceSpanStatusCancelled:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_CANCELLED
	case models.RunTraceSpanStatusSkipped:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_SKIPPED
	case models.RunTraceSpanStatusWaiting, models.RunTraceSpanStatusQueued:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_WAITING
	case models.RunTraceSpanStatusRunning:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_RUNNING
	default:
		return apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_UNKNOWN
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
	case models.StepOpWaitForSignal:
		return apiv2.TraceStepOp_TRACE_STEP_OP_WAIT_FOR_SIGNAL
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
		duration := span.EndedAt.Sub(*span.StartedAt) / time.Millisecond
		if duration < 0 {
			return nil
		}

		value := uint64(duration)
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

func (s *Service) traceRunOutput(ctx context.Context, runID ulid.ULID) *structpb.Struct {
	if s.traces == nil {
		return nil
	}

	root, err := s.traces.GetSpansByRunID(ctx, runID)
	if err != nil || root == nil {
		return nil
	}

	outputID := root.GetOutputID()
	if outputID == nil {
		return nil
	}

	output, err := loadTraceOutput(ctx, s.traces, *outputID)
	if err != nil || output == nil {
		return nil
	}

	return output.output
}
