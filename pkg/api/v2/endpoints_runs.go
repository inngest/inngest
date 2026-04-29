package apiv2

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
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

	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get function trace is not yet implemented")
}

func (s *Service) GetFunctionTraceSpan(ctx context.Context, req *apiv2.GetFunctionTraceSpanRequest) (*apiv2.GetFunctionTraceSpanResponse, error) {
	if req.RunId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Run ID is required")
	}
	if req.SpanId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Span ID is required")
	}

	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get function trace span is not yet implemented")
}

func toFunctionRun(run *cqrs.FunctionRun, fn inngest.DeployedFunction, includeOutput bool) *apiv2.FunctionRun {
	startedAt := timestamppb.New(run.RunStartedAt)

	result := &apiv2.FunctionRun{
		Id: run.RunID.String(),
		Function: &apiv2.FunctionRef{
			Id:   fn.Slug,
			Name: fn.Function.Name,
		},
		App: &apiv2.AppRef{
			Id: fn.AppID.String(),
		},
		Status:    toFunctionRunStatus(run.Status),
		QueuedAt:  startedAt,
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
