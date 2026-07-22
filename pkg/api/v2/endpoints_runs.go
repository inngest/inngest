package apiv2

import (
	"context"
	"encoding/json"
	"errors"
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
	defaultRunsLimit      = 20
	maxRunsLimit          = 100
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
	run, err := s.runs.GetRun(ctx, runID, GetRunOpts{
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

func (s *Service) ListRuns(ctx context.Context, req *apiv2.ListRunsRequest) (*apiv2.ListRunsResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListRuns_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no runs were fetched.")
	}

	if s.runs == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "List runs is not yet implemented")
	}

	opts, err := listRunsOpts(req)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	return s.listRuns(ctx, opts)
}

func (s *Service) ListFunctionRuns(ctx context.Context, req *apiv2.ListFunctionRunsRequest) (*apiv2.ListFunctionRunsResponse, error) {
	if req.AppId == "" || req.FunctionId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "App ID and function ID are required")
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListFunctionRuns_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no runs were fetched.")
	}

	if s.runs == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "List runs is not yet implemented")
	}

	opts, err := listRunsOpts(&apiv2.ListRunsRequest{
		IncludeOutput: req.IncludeOutput,
		Cursor:        req.Cursor,
		Limit:         req.Limit,
		From:          req.From,
		Until:         req.Until,
		TimeField:     req.TimeField,
		Status:        req.Status,
		AppId:         []string{decodePathParam(req.AppId)},
		FunctionId:    []string{decodePathParam(req.FunctionId)},
		IsDeferred:    req.IsDeferred,
		Order:         req.Order,
	})
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	resp, err := s.listRuns(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &apiv2.ListFunctionRunsResponse{
		Data:     resp.Data,
		Metadata: resp.Metadata,
		Page:     resp.Page,
	}, nil
}

func (s *Service) listRuns(ctx context.Context, opts GetRunsOpts) (*apiv2.ListRunsResponse, error) {
	result, err := s.runs.GetRuns(ctx, opts)
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch runs")
	}
	if result == nil {
		result = &GetRunsResult{}
	}

	data := make([]*apiv2.FunctionRun, 0, len(result.Runs))
	for _, run := range result.Runs {
		data = append(data, toAPIRunListItem(run))
	}

	return &apiv2.ListRunsResponse{
		Data:     data,
		Metadata: runsResponseMetadata(opts.From, opts.Until),
		Page:     runsPage(result.Runs, opts.Limit, result.HasMore),
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

	if s.runs == nil {
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
	result, err := s.runs.GetRuns(ctx, GetRunsOpts{
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
		data = append(data, toAPIRunListItem(run))
	}

	return &apiv2.GetEventRunsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     runsPage(result.Runs, limit, result.HasMore),
	}, nil
}

func (s *Service) Rerun(ctx context.Context, req *apiv2.RerunRequest) (*apiv2.RerunResponse, error) {
	if req.RunId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Run ID is required")
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_Rerun_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no run was rerun.")
	}

	if s.runs == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Rerun is not yet implemented")
	}

	runID, err := ulid.Parse(req.RunId)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Run ID must be a valid ULID")
	}

	reqOpts := RerunOpts{}
	if req.FromStep != nil {
		if req.FromStep.StepId == "" {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Step ID is required")
		}

		fromStep := &RerunFromStep{StepID: req.FromStep.StepId}
		if req.FromStep.Input != nil {
			input, err := json.Marshal(req.FromStep.Input.AsSlice())
			if err != nil {
				return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Step input must be a valid JSON array")
			}
			fromStep.Input = input
		}

		reqOpts.FromStep = fromStep
	}

	newRunID, err := s.runs.Rerun(ctx, runID, reqOpts)
	if err != nil {
		switch {
		case errors.Is(err, ErrRunNotFound):
			return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Run not found")
		case errors.Is(err, ErrCronRerunNotSupported):
			return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Rerunning cron-triggered runs is not yet supported")
		case errors.Is(err, ErrRerunStepNotFound):
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidRequest, "Step not found in original run")
		case errors.Is(err, ErrRerunStepAmbiguous):
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidRequest, "Step name matches multiple steps in original run")
		}
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to rerun run")
	}

	return &apiv2.RerunResponse{
		Data: &apiv2.RerunData{
			RunId: newRunID.String(),
		},
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

func listRunsOpts(req *apiv2.ListRunsRequest) (GetRunsOpts, error) {
	if len(req.GetFunctionId()) > 0 && len(req.GetAppId()) == 0 {
		return GetRunsOpts{}, fmt.Errorf("appId is required when filtering by functionId")
	}

	cursor, limit, err := listRunsPageOpts(req.GetCursor(), req.GetLimit())
	if err != nil {
		return GetRunsOpts{}, err
	}

	from, err := optionalTimestamp(req.GetFrom(), "from")
	if err != nil {
		return GetRunsOpts{}, err
	}
	until, err := optionalTimestamp(req.GetUntil(), "until")
	if err != nil {
		return GetRunsOpts{}, err
	}
	if from != nil && until != nil && !from.Before(*until) {
		return GetRunsOpts{}, fmt.Errorf("from must be before until")
	}

	status, err := runStatusesFromAPI(req.GetStatus())
	if err != nil {
		return GetRunsOpts{}, err
	}
	timeField, err := runTimeFieldFromAPI(req.GetTimeField())
	if err != nil {
		return GetRunsOpts{}, err
	}
	order, err := orderDirectionFromAPI(req.GetOrder())
	if err != nil {
		return GetRunsOpts{}, err
	}

	return GetRunsOpts{
		Cursor:        cursor,
		Limit:         limit,
		IncludeOutput: req.GetIncludeOutput(),
		From:          from,
		Until:         until,
		TimeField:     timeField,
		Status:        status,
		AppIDs:        req.GetAppId(),
		FunctionIDs:   req.GetFunctionId(),
		IsDeferred:    req.IsDeferred,
		Order:         order,
	}, nil
}

func listRunsPageOpts(cursor string, requestedLimit int32) (string, int, error) {
	return parseRunsPageOpts(cursor, requestedLimit, defaultRunsLimit, maxRunsLimit)
}

func runsPageOpts(cursor string, requestedLimit int32) (string, int, error) {
	return parseRunsPageOpts(cursor, requestedLimit, defaultEventRunsLimit, maxEventRunsLimit)
}

func parseRunsPageOpts(cursor string, requestedLimit int32, defaultLimit, maxLimit int) (string, int, error) {
	limit := int(requestedLimit)
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 {
		return "", 0, fmt.Errorf("Limit must be at least 1")
	}
	if limit > maxLimit {
		return "", 0, fmt.Errorf("Limit cannot exceed %d", maxLimit)
	}
	if cursor != "" && !validRunsCursor(cursor) {
		return "", 0, fmt.Errorf("Cursor is invalid")
	}
	return cursor, limit, nil
}

func validRunsCursor(cursor string) bool {
	pageCursor := cqrs.TracePageCursor{}
	return pageCursor.Decode(cursor) == nil && pageCursor.ID != "" && len(pageCursor.Cursors) > 0
}

func runsPage(runs []*RunListItem, limit int, hasMore bool) *apiv2.Page {
	page := &apiv2.Page{
		HasMore: hasMore,
		Limit:   int32(limit),
	}
	if hasMore && len(runs) > 0 {
		nextCursor := runs[len(runs)-1].Cursor
		page.Cursor = &nextCursor
	}
	return page
}

func optionalTimestamp(ts *timestamppb.Timestamp, field string) (*time.Time, error) {
	if ts == nil {
		return nil, nil
	}
	if err := ts.CheckValid(); err != nil {
		return nil, fmt.Errorf("%s must be a valid timestamp", field)
	}
	value := ts.AsTime().UTC()
	return &value, nil
}

func runStatusesFromAPI(statuses []string) ([]enums.RunStatus, error) {
	result := make([]enums.RunStatus, 0, len(statuses))
	for _, status := range statuses {
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case "QUEUED":
			result = append(result, enums.RunStatusScheduled)
		case "RUNNING":
			result = append(result, enums.RunStatusRunning)
		case "COMPLETED":
			result = append(result, enums.RunStatusCompleted)
		case "FAILED":
			result = append(result, enums.RunStatusFailed)
		case "CANCELLED":
			result = append(result, enums.RunStatusCancelled)
		default:
			return nil, fmt.Errorf("Status is invalid")
		}
	}
	return result, nil
}

func runTimeFieldFromAPI(field string) (RunTimeField, error) {
	switch normalizeRunFilterToken(field) {
	case "", "QUEUEDAT", "QUEUED_AT":
		return RunTimeFieldQueuedAt, nil
	case "STARTEDAT", "STARTED_AT":
		return RunTimeFieldStartedAt, nil
	case "ENDEDAT", "ENDED_AT":
		return RunTimeFieldEndedAt, nil
	default:
		return RunTimeFieldQueuedAt, fmt.Errorf("timeField is invalid")
	}
}

func orderDirectionFromAPI(direction string) (OrderDirection, error) {
	switch normalizeRunFilterToken(direction) {
	case "", "DESC":
		return OrderDirectionDesc, nil
	case "ASC":
		return OrderDirectionAsc, nil
	default:
		return OrderDirectionDesc, fmt.Errorf("order is invalid")
	}
}

func normalizeRunFilterToken(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func runsResponseMetadata(from, until *time.Time) *apiv2.ResponseMetadata {
	metadata := &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()}
	if from != nil && until != nil {
		metadata.TimeRange = &apiv2.TimeRange{
			From:  timestamppb.New(*from),
			Until: timestamppb.New(*until),
		}
	}
	return metadata
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

func functionRefID(fn inngest.DeployedFunction) string {
	return PublicFunctionID(fn.AppName, fn.Slug, fn.Function.Slug)
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
