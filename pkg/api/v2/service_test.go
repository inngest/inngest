package apiv2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing/meta"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestService_Health(t *testing.T) {
	service := NewService(ServiceOptions{})

	t.Run("returns health status with timestamp", func(t *testing.T) {
		ctx := context.Background()
		req := &apiv2.HealthRequest{}

		before := time.Now()
		resp, err := service.Health(ctx, req)
		after := time.Now()

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Equal(t, "ok", resp.Data.Status)
		require.NotNil(t, resp.Metadata)
		require.NotNil(t, resp.Metadata.FetchedAt)
		require.Nil(t, resp.Metadata.CachedUntil)

		fetchedTime := resp.Metadata.FetchedAt.AsTime()
		require.True(t, fetchedTime.After(before) || fetchedTime.Equal(before))
		require.True(t, fetchedTime.Before(after) || fetchedTime.Equal(after))
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		req := &apiv2.HealthRequest{}
		resp, err := service.Health(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "ok", resp.Data.Status)
	})
}

func TestNewService(t *testing.T) {
	t.Run("creates new service instance", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		require.NotNil(t, service)
		require.IsType(t, &Service{}, service)
	})
}

func TestNewHTTPHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("creates HTTP handler without auth middleware", func(t *testing.T) {
		opts := HTTPHandlerOptions{}
		handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)

		require.NoError(t, err)
		require.NotNil(t, handler)
	})

	t.Run("creates HTTP handler with auth middleware", func(t *testing.T) {
		authMiddlewareCalled := false
		authMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authMiddlewareCalled = true
				next.ServeHTTP(w, r)
			})
		}

		opts := HTTPHandlerOptions{
			AuthnMiddleware: authMiddleware,
		}
		handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)

		require.NoError(t, err)
		require.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.True(t, authMiddlewareCalled)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("handles health endpoint correctly", func(t *testing.T) {
		opts := HTTPHandlerOptions{}
		handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)

		require.NoError(t, err)
		require.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
		require.Contains(t, rec.Body.String(), `"status":"ok"`)
		require.Contains(t, rec.Body.String(), `"fetchedAt"`)
	})

	t.Run("strips /api/v2 prefix correctly", func(t *testing.T) {
		opts := HTTPHandlerOptions{}
		handler, err := newTestHTTPHandler(ctx, ServiceOptions{}, opts)

		require.NoError(t, err)
		require.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestService_HealthResponse_Structure(t *testing.T) {
	service := NewService(ServiceOptions{})
	ctx := context.Background()
	req := &apiv2.HealthRequest{}

	resp, err := service.Health(ctx, req)
	require.NoError(t, err)

	t.Run("validates response structure", func(t *testing.T) {
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Metadata)
	})

	t.Run("validates data fields", func(t *testing.T) {
		require.Equal(t, "ok", resp.Data.Status)
	})

	t.Run("validates metadata fields", func(t *testing.T) {
		require.NotNil(t, resp.Metadata.FetchedAt)
		require.Nil(t, resp.Metadata.CachedUntil)

		fetchedAt := resp.Metadata.FetchedAt.AsTime()
		require.False(t, fetchedAt.IsZero())
		require.True(t, fetchedAt.Before(time.Now().Add(time.Second)))
	})
}

func TestService_HealthRequest_Validation(t *testing.T) {
	service := NewService(ServiceOptions{})
	ctx := context.Background()

	t.Run("accepts nil request", func(t *testing.T) {
		resp, err := service.Health(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "ok", resp.Data.Status)
	})

	t.Run("accepts empty request", func(t *testing.T) {
		req := &apiv2.HealthRequest{}
		resp, err := service.Health(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "ok", resp.Data.Status)
	})
}

func TestService_Metadata_Timestamp(t *testing.T) {
	service := NewService(ServiceOptions{})
	ctx := context.Background()
	req := &apiv2.HealthRequest{}

	t.Run("timestamps are consistent and recent", func(t *testing.T) {
		start := time.Now()

		resp1, err := service.Health(ctx, req)
		require.NoError(t, err)

		resp2, err := service.Health(ctx, req)
		require.NoError(t, err)

		end := time.Now()

		time1 := resp1.Metadata.FetchedAt.AsTime()
		time2 := resp2.Metadata.FetchedAt.AsTime()

		require.True(t, time1.After(start) || time1.Equal(start))
		require.True(t, time1.Before(end) || time1.Equal(end))
		require.True(t, time2.After(start) || time2.Equal(start))
		require.True(t, time2.Before(end) || time2.Equal(end))
		require.True(t, time2.After(time1) || time2.Equal(time1))
	})

	t.Run("timestamp format is valid protobuf timestamp", func(t *testing.T) {
		resp, err := service.Health(ctx, req)
		require.NoError(t, err)

		timestamp := resp.Metadata.FetchedAt
		require.NotNil(t, timestamp)

		require.True(t, timestamp.IsValid())

		asTime := timestamp.AsTime()
		require.False(t, asTime.IsZero())

		fromTime := timestamppb.New(asTime)
		require.Equal(t, timestamp.Seconds, fromTime.Seconds)
		require.Equal(t, timestamp.Nanos, fromTime.Nanos)
	})
}

func boolPtr(value bool) *bool {
	return &value
}

func strPtr(value string) *string {
	return &value
}

func TestService_GetFunctionRun(t *testing.T) {
	runID := ulid.MustParse("01hp1zx8m3ng9vp6qn0xk7j4cy")
	functionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	appID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	startedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(2 * time.Second)
	fn := inngest.DeployedFunction{
		ID:      functionID,
		Slug:    "my-app-test-fn",
		AppID:   appID,
		AppName: "my-app",
		Function: inngest.Function{
			Name: "Test function",
			Slug: "test-fn",
		},
	}
	run := &cqrs.FunctionRun{
		RunID:        runID,
		RunStartedAt: startedAt,
		FunctionID:   functionID,
		EventID:      runID,
		Status:       enums.RunStatusCompleted,
		EndedAt:      &endedAt,
	}
	functions := &mockFunctionProvider{}
	functions.On("GetFunction", mock.Anything, functionID.String()).Return(fn, nil).Once()
	runs := &mockFunctionRunReader{}
	runs.On("GetFunctionRun", mock.Anything, runID, GetFunctionRunOpts{IncludeOutput: true}).Return(run, nil).Once()

	service := NewService(ServiceOptions{
		Functions:    functions,
		FunctionRuns: runs,
	})
	t.Cleanup(func() {
		functions.AssertExpectations(t)
		runs.AssertExpectations(t)
	})

	t.Run("returns mapped run data", func(t *testing.T) {
		resp, err := service.GetFunctionRun(context.Background(), &apiv2.GetFunctionRunRequest{
			RunId:         runID.String(),
			IncludeOutput: boolPtr(true),
		})
		require.NoError(t, err)
		require.Equal(t, runID.String(), resp.Data.Id)
		require.Equal(t, apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED, resp.Data.Status)
		require.Equal(t, ulid.Time(runID.Time()).UTC(), resp.Data.QueuedAt.AsTime())
		require.Equal(t, startedAt, resp.Data.StartedAt.AsTime())
		require.Equal(t, "test-fn", resp.Data.Function.Id)
		require.Equal(t, "Test function", resp.Data.Function.Name)
		require.Equal(t, "my-app", resp.Data.App.Id)
		require.Nil(t, resp.Data.Output)
		require.NotNil(t, resp.Data.DurationMs)
		require.Equal(t, uint64(2000), *resp.Data.DurationMs)
	})

	t.Run("requires run id", func(t *testing.T) {
		resp, err := service.GetFunctionRun(context.Background(), &apiv2.GetFunctionRunRequest{})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "Run ID is required")
	})

	t.Run("validates run id format", func(t *testing.T) {
		resp, err := service.GetFunctionRun(context.Background(), &apiv2.GetFunctionRunRequest{
			RunId: "not-a-ulid",
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "Run ID must be a valid ULID")
	})

	t.Run("returns not found when run is missing", func(t *testing.T) {
		runs := &mockFunctionRunReader{}
		runs.On("GetFunctionRun", mock.Anything, runID, GetFunctionRunOpts{}).Return(nil, errors.New("missing")).Once()
		t.Cleanup(func() {
			runs.AssertExpectations(t)
		})

		service := NewService(ServiceOptions{
			Functions:    &mockFunctionProvider{},
			FunctionRuns: runs,
		})

		resp, err := service.GetFunctionRun(context.Background(), &apiv2.GetFunctionRunRequest{
			RunId: runID.String(),
		})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "Run not found")
	})

	t.Run("returns not found when function is missing", func(t *testing.T) {
		runs := &mockFunctionRunReader{}
		runs.On("GetFunctionRun", mock.Anything, runID, GetFunctionRunOpts{}).Return(run, nil).Once()
		functions := &mockFunctionProvider{}
		functions.On("GetFunction", mock.Anything, functionID.String()).Return(inngest.DeployedFunction{}, errors.New("missing")).Once()
		t.Cleanup(func() {
			runs.AssertExpectations(t)
			functions.AssertExpectations(t)
		})

		service := NewService(ServiceOptions{
			Functions:    functions,
			FunctionRuns: runs,
		})

		resp, err := service.GetFunctionRun(context.Background(), &apiv2.GetFunctionRunRequest{
			RunId: runID.String(),
		})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "Function not found")
	})

	t.Run("uses root trace output when requested", func(t *testing.T) {
		inputSpanID := "input-span"
		outputIdentifier := cqrs.SpanIdentifier{
			SpanID:      "output-span",
			InputSpanID: &inputSpanID,
			Preview:     boolPtr(true),
		}
		outputID, err := outputIdentifier.Encode()
		require.NoError(t, err)

		runs := &mockFunctionRunReader{}
		runs.On("GetFunctionRun", mock.Anything, runID, GetFunctionRunOpts{IncludeOutput: true}).Return(&cqrs.FunctionRun{
			RunID:        runID,
			RunStartedAt: startedAt,
			FunctionID:   functionID,
			EventID:      runID,
			Status:       enums.RunStatusCompleted,
			EndedAt:      &endedAt,
			Output:       json.RawMessage(`""`),
		}, nil).Once()
		functions := &mockFunctionProvider{}
		functions.On("GetFunction", mock.Anything, functionID.String()).Return(fn, nil).Once()
		traces := &mockFunctionTraceReader{}
		traces.On("GetSpansByRunID", mock.Anything, runID).Return(&cqrs.OtelSpan{
			RunID:    runID,
			OutputID: &outputID,
		}, nil).Once()
		traces.On("GetSpanOutput", mock.Anything, outputIdentifier).Return(&cqrs.SpanOutput{
			Data: []byte(`{"body":"Hello, World!"}`),
		}, nil).Once()
		t.Cleanup(func() {
			runs.AssertExpectations(t)
			functions.AssertExpectations(t)
			traces.AssertExpectations(t)
		})

		service := NewService(ServiceOptions{
			Functions:      functions,
			FunctionRuns:   runs,
			FunctionTraces: traces,
		})

		resp, err := service.GetFunctionRun(context.Background(), &apiv2.GetFunctionRunRequest{
			RunId:         runID.String(),
			IncludeOutput: boolPtr(true),
		})

		require.NoError(t, err)
		require.NotNil(t, resp.Data.Output)
		require.Equal(t, "Hello, World!", resp.Data.Output.Fields["body"].GetStringValue())
	})

	t.Run("does not fall back to run output", func(t *testing.T) {
		runs := &mockFunctionRunReader{}
		runs.On("GetFunctionRun", mock.Anything, runID, GetFunctionRunOpts{IncludeOutput: true}).Return(&cqrs.FunctionRun{
			RunID:        runID,
			RunStartedAt: startedAt,
			FunctionID:   functionID,
			EventID:      runID,
			Status:       enums.RunStatusCompleted,
			EndedAt:      &endedAt,
			Output:       json.RawMessage(`{"old":true}`),
		}, nil).Once()
		functions := &mockFunctionProvider{}
		functions.On("GetFunction", mock.Anything, functionID.String()).Return(fn, nil).Once()
		t.Cleanup(func() {
			runs.AssertExpectations(t)
			functions.AssertExpectations(t)
		})

		service := NewService(ServiceOptions{
			Functions:    functions,
			FunctionRuns: runs,
		})

		resp, err := service.GetFunctionRun(context.Background(), &apiv2.GetFunctionRunRequest{
			RunId:         runID.String(),
			IncludeOutput: boolPtr(true),
		})

		require.NoError(t, err)
		require.Nil(t, resp.Data.Output)
	})
}

func TestService_GetEventRuns(t *testing.T) {
	eventID := ulid.MustParse("01hp1zyb8p2nb5kvm2a6x1h9ae")
	runID := ulid.MustParse("01hp1zx8m3ng9vp6qn0xk7j4cy")
	nextRunID := ulid.MustParse("01hp1zx8m3ng9vp6qn0xk7j4cz")
	startedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(2 * time.Second)

	run := &RunListItem{
		RunID:        runID,
		RunStartedAt: startedAt,
		EventID:      eventID,
		Status:       enums.RunStatusCompleted,
		EndedAt:      &endedAt,
		Output:       json.RawMessage(`{"ok":true}`),
		FunctionID:   "test-fn",
		FunctionName: "Test function",
		AppID:        "my-app",
	}
	nextRun := &RunListItem{
		RunID:        nextRunID,
		RunStartedAt: startedAt.Add(time.Minute),
		EventID:      eventID,
		Status:       enums.RunStatusRunning,
		FunctionID:   "next-fn",
		FunctionName: "Next function",
		AppID:        "my-app",
	}

	t.Run("returns mapped event runs", func(t *testing.T) {
		reader := &mockRunsReader{}
		reader.On("GetRuns", mock.Anything, GetRunsOpts{
			EventID:       eventID,
			Limit:         defaultEventRunsLimit,
			IncludeOutput: true,
		}).Return(&GetRunsResult{Runs: []*RunListItem{run}}, nil).Once()
		t.Cleanup(func() {
			reader.AssertExpectations(t)
		})

		service := NewService(ServiceOptions{RunList: reader})
		resp, err := service.GetEventRuns(context.Background(), &apiv2.GetEventRunsRequest{
			EventId:       eventID.String(),
			IncludeOutput: boolPtr(true),
		})

		require.NoError(t, err)
		require.Len(t, resp.Data, 1)
		require.Equal(t, runID.String(), resp.Data[0].Id)
		require.Equal(t, "test-fn", resp.Data[0].Function.Id)
		require.Equal(t, "Test function", resp.Data[0].Function.Name)
		require.Equal(t, "my-app", resp.Data[0].App.Id)
		require.Equal(t, []string{eventID.String()}, resp.Data[0].Trigger.EventIds)
		require.NotNil(t, resp.Data[0].Output)
		require.True(t, resp.Data[0].Output.Fields["ok"].GetBoolValue())
		require.NotNil(t, resp.Page)
		require.False(t, resp.Page.HasMore)
		require.Equal(t, int32(defaultEventRunsLimit), resp.Page.Limit)
	})

	t.Run("passes pagination to reader", func(t *testing.T) {
		reader := &mockRunsReader{}
		reader.On("GetRuns", mock.Anything, GetRunsOpts{
			EventID: eventID,
			Limit:   1,
		}).Return(&GetRunsResult{Runs: []*RunListItem{run}, HasMore: true}, nil).Once()
		reader.On("GetRuns", mock.Anything, GetRunsOpts{
			EventID: eventID,
			Cursor:  runID,
			Limit:   1,
		}).Return(&GetRunsResult{Runs: []*RunListItem{nextRun}}, nil).Once()
		t.Cleanup(func() {
			reader.AssertExpectations(t)
		})

		service := NewService(ServiceOptions{RunList: reader})
		limit := int32(1)

		first, err := service.GetEventRuns(context.Background(), &apiv2.GetEventRunsRequest{
			EventId: eventID.String(),
			Limit:   &limit,
		})
		require.NoError(t, err)
		require.Len(t, first.Data, 1)
		require.Equal(t, runID.String(), first.Data[0].Id)
		require.True(t, first.Page.HasMore)
		require.NotNil(t, first.Page.Cursor)
		require.Equal(t, runID.String(), first.Page.GetCursor())

		second, err := service.GetEventRuns(context.Background(), &apiv2.GetEventRunsRequest{
			EventId: eventID.String(),
			Cursor:  first.Page.Cursor,
			Limit:   &limit,
		})
		require.NoError(t, err)
		require.Len(t, second.Data, 1)
		require.Equal(t, nextRunID.String(), second.Data[0].Id)
		require.False(t, second.Page.HasMore)
		require.Nil(t, second.Page.Cursor)
	})

	t.Run("requires event id", func(t *testing.T) {
		service := NewService(ServiceOptions{})
		resp, err := service.GetEventRuns(context.Background(), &apiv2.GetEventRunsRequest{})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "Event ID is required")
	})

	t.Run("validates event id format", func(t *testing.T) {
		service := NewService(ServiceOptions{RunList: &mockRunsReader{}})
		resp, err := service.GetEventRuns(context.Background(), &apiv2.GetEventRunsRequest{
			EventId: "not-a-ulid",
		})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "Event ID must be a valid ULID")
	})

	t.Run("returns internal error when reader fails", func(t *testing.T) {
		reader := &mockRunsReader{}
		reader.On("GetRuns", mock.Anything, GetRunsOpts{
			EventID: eventID,
			Limit:   defaultEventRunsLimit,
		}).Return(nil, errors.New("reader failed")).Once()
		t.Cleanup(func() {
			reader.AssertExpectations(t)
		})

		service := NewService(ServiceOptions{RunList: reader})
		resp, err := service.GetEventRuns(context.Background(), &apiv2.GetEventRunsRequest{
			EventId: eventID.String(),
		})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "Unable to fetch event runs")
	})

	t.Run("validates pagination", func(t *testing.T) {
		service := NewService(ServiceOptions{RunList: &mockRunsReader{}})
		invalidLimit := int32(maxEventRunsLimit + 1)

		resp, err := service.GetEventRuns(context.Background(), &apiv2.GetEventRunsRequest{
			EventId: eventID.String(),
			Limit:   &invalidLimit,
		})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "Limit cannot exceed 40")
	})
}

func TestToTraceSpanStatus(t *testing.T) {
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_COMPLETED, toTraceSpanStatus(models.RunTraceSpanStatusCompleted))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_FAILED, toTraceSpanStatus(models.RunTraceSpanStatusFailed))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_WAITING, toTraceSpanStatus(models.RunTraceSpanStatusWaiting))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_WAITING, toTraceSpanStatus(models.RunTraceSpanStatusQueued))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_RUNNING, toTraceSpanStatus(models.RunTraceSpanStatusRunning))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_CANCELLED, toTraceSpanStatus(models.RunTraceSpanStatusCancelled))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_SKIPPED, toTraceSpanStatus(models.RunTraceSpanStatusSkipped))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_UNKNOWN, toTraceSpanStatus(models.RunTraceSpanStatus("UNKNOWN")))
}

func TestService_GetFunctionTrace(t *testing.T) {
	runID := ulid.MustParse("01jpq5jcxm8qhg2x61v61bh8p0")
	queuedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	startedAt := queuedAt.Add(time.Second)
	endedAt := startedAt.Add(2 * time.Second)
	stepName := "Fetch data"
	stepID := "step-1"
	stepStatus := enums.StepStatusCompleted
	stepOp := enums.OpcodeStepRun
	statusCode := 200
	responseHeaders := headers.Compact{"content-type": {"application/json"}}
	outputIdentifier := cqrs.SpanIdentifier{
		SpanID:      "span-output",
		InputSpanID: strPtr("span-input"),
		Preview:     boolPtr(true),
	}
	outputID := mustEncodeSpanIdentifier(t, outputIdentifier)

	root := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:      meta.SpanNameRun,
			SpanID:    "run-span",
			TraceID:   "trace-123",
			StartTime: queuedAt,
			EndTime:   endedAt,
		},
		RunID: runID,
		Attributes: &meta.ExtractedValues{
			QueuedAt:      &queuedAt,
			StartedAt:     &startedAt,
			EndedAt:       &endedAt,
			DynamicStatus: &stepStatus,
		},
		Children: []*cqrs.OtelSpan{
			{
				RawOtelSpan: cqrs.RawOtelSpan{
					Name:      meta.SpanNameStep,
					SpanID:    "step-span",
					TraceID:   "trace-123",
					StartTime: startedAt,
					EndTime:   endedAt,
				},
				OutputID: &outputID,
				Attributes: &meta.ExtractedValues{
					QueuedAt:           &startedAt,
					StartedAt:          &startedAt,
					EndedAt:            &endedAt,
					StepName:           &stepName,
					StepID:             &stepID,
					StepOp:             &stepOp,
					DynamicStatus:      &stepStatus,
					ResponseStatusCode: &statusCode,
					ResponseHeaders:    &responseHeaders,
				},
				Children: []*cqrs.OtelSpan{
					{
						RawOtelSpan: cqrs.RawOtelSpan{
							Name:      meta.SpanNameStep,
							SpanID:    "nested-step-span",
							TraceID:   "trace-123",
							StartTime: startedAt.Add(100 * time.Millisecond),
							EndTime:   endedAt.Add(-100 * time.Millisecond),
						},
						Attributes: &meta.ExtractedValues{
							QueuedAt:      &startedAt,
							StartedAt:     &startedAt,
							EndedAt:       &endedAt,
							DynamicStatus: &stepStatus,
						},
					},
				},
			},
		},
	}

	newService := func(t *testing.T, includeOutput bool) *Service {
		t.Helper()

		reader := &mockFunctionTraceReader{}
		reader.On("GetSpansByRunID", mock.Anything, runID).Return(root, nil).Once()
		if includeOutput {
			reader.On("GetSpanOutput", mock.Anything, outputIdentifier).Return(&cqrs.SpanOutput{
				Input: []byte(`{"message":"hello"}`),
				Data:  []byte(`{"ok":true}`),
			}, nil).Once()
		}
		t.Cleanup(func() {
			reader.AssertExpectations(t)
		})

		return NewService(ServiceOptions{
			FunctionTraces: reader,
		})
	}
	validationService := NewService(ServiceOptions{
		FunctionTraces: &mockFunctionTraceReader{},
	})

	t.Run("returns a nested trace response", func(t *testing.T) {
		service := newService(t, true)

		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{
			RunId:         runID.String(),
			IncludeOutput: boolPtr(true),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Equal(t, runID.String(), resp.Data.RunId)
		require.NotNil(t, resp.Data.RootSpan)
		require.Equal(t, "Run", resp.Data.RootSpan.Name)
		require.Equal(t, "run-span", resp.Data.RootSpan.Id)
		require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_COMPLETED, resp.Data.RootSpan.Status)
		require.Len(t, resp.Data.RootSpan.Children, 1)

		child := resp.Data.RootSpan.Children[0]
		require.Equal(t, "Fetch data", child.Name)
		require.Equal(t, "step-span", child.Id)
		require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_COMPLETED, child.Status)
		require.NotNil(t, child.StepOp)
		require.Equal(t, apiv2.TraceStepOp_TRACE_STEP_OP_RUN, *child.StepOp)
		require.NotNil(t, child.StepId)
		require.Equal(t, "step-1", *child.StepId)
		require.NotNil(t, child.Input)
		require.Equal(t, "hello", child.Input.Fields["message"].GetStringValue())
		require.NotNil(t, child.Output)
		require.True(t, child.Output.Fields["ok"].GetBoolValue())
	})

	t.Run("validates missing run ID", func(t *testing.T) {
		resp, err := validationService.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Run ID is required")
	})

	t.Run("validates run ID format", func(t *testing.T) {
		resp, err := validationService.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{RunId: "not-a-ulid"})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Run ID must be a valid ULID")
	})

	t.Run("omits output when not requested", func(t *testing.T) {
		service := newService(t, false)

		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{
			RunId:         runID.String(),
			IncludeOutput: boolPtr(false),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, resp.Data.RootSpan.Children[0].Input)
		require.Nil(t, resp.Data.RootSpan.Children[0].Output)
	})

	t.Run("returns not found when trace is missing", func(t *testing.T) {
		reader := &mockFunctionTraceReader{}
		reader.On("GetSpansByRunID", mock.Anything, runID).Return(nil, errors.New("missing")).Once()
		t.Cleanup(func() {
			reader.AssertExpectations(t)
		})

		service := NewService(ServiceOptions{
			FunctionTraces: reader,
		})

		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{
			RunId: runID.String(),
		})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "Trace not found")
	})

	t.Run("returns internal error when trace mapping fails", func(t *testing.T) {
		badOutputID := "not-base64"
		reader := &mockFunctionTraceReader{}
		reader.On("GetSpansByRunID", mock.Anything, runID).Return(&cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{
				Name:      meta.SpanNameRun,
				SpanID:    "run-span",
				TraceID:   "trace-123",
				StartTime: queuedAt,
				EndTime:   endedAt,
			},
			RunID: runID,
			Attributes: &meta.ExtractedValues{
				QueuedAt:      &queuedAt,
				DynamicStatus: &stepStatus,
			},
			Children: []*cqrs.OtelSpan{
				{
					RawOtelSpan: cqrs.RawOtelSpan{
						Name:      meta.SpanNameStep,
						SpanID:    "step-span",
						TraceID:   "trace-123",
						StartTime: startedAt,
						EndTime:   endedAt,
					},
					OutputID: &badOutputID,
					Attributes: &meta.ExtractedValues{
						QueuedAt:      &startedAt,
						DynamicStatus: &stepStatus,
					},
				},
			},
		}, nil).Once()
		t.Cleanup(func() {
			reader.AssertExpectations(t)
		})

		service := NewService(ServiceOptions{
			FunctionTraces: reader,
		})

		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{
			RunId:         runID.String(),
			IncludeOutput: boolPtr(true),
		})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "Unable to build trace response")
	})
}

func mustEncodeSpanIdentifier(t *testing.T, id cqrs.SpanIdentifier) string {
	t.Helper()

	payload, err := json.Marshal(id)
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(payload)
}

func TestService_GetFunctionTraceNotImplemented(t *testing.T) {
	service := NewService(ServiceOptions{})

	t.Run("returns not implemented for valid request", func(t *testing.T) {
		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{
			RunId: "01hp1zx8m3ng9vp6qn0xk7j4cy",
		})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Get function trace is not yet implemented")
	})

	t.Run("validates missing run ID", func(t *testing.T) {
		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Run ID is required")
	})
}
