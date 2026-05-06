package apiv2

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

type stubFunctionProvider struct {
	fn inngest.DeployedFunction
}

func (s stubFunctionProvider) GetFunction(ctx context.Context, identifier string) (inngest.DeployedFunction, error) {
	return s.fn, nil
}

type stubFunctionRunReader struct {
	run *cqrs.FunctionRun
}

func (s stubFunctionRunReader) GetFunctionRun(ctx context.Context, runID ulid.ULID) (*cqrs.FunctionRun, error) {
	return s.run, nil
}

type stubFunctionTraceReader struct {
	root   *cqrs.OtelSpan
	output *cqrs.SpanOutput
}

func (s stubFunctionTraceReader) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	return s.root, nil
}

func (s stubFunctionTraceReader) GetSpanOutput(ctx context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	return s.output, nil
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
	output, err := json.Marshal(map[string]any{"ok": true})
	require.NoError(t, err)

	service := NewService(ServiceOptions{
		Functions: stubFunctionProvider{
			fn: inngest.DeployedFunction{
				ID:      functionID,
				Slug:    "my-app-test-fn",
				AppID:   appID,
				AppName: "my-app",
				Function: inngest.Function{
					Name: "Test function",
					Slug: "test-fn",
				},
			},
		},
		FunctionRuns: stubFunctionRunReader{
			run: &cqrs.FunctionRun{
				RunID:        runID,
				RunStartedAt: startedAt,
				FunctionID:   functionID,
				EventID:      runID,
				Status:       enums.RunStatusCompleted,
				EndedAt:      &endedAt,
				Output:       output,
			},
		},
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
		require.NotNil(t, resp.Data.Output)
		require.Equal(t, true, resp.Data.Output.Fields["ok"].GetBoolValue())
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
}

func TestToTraceSpanStatus(t *testing.T) {
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_COMPLETED, toTraceSpanStatus(models.RunTraceSpanStatusCompleted))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_FAILED, toTraceSpanStatus(models.RunTraceSpanStatusFailed))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_WAITING, toTraceSpanStatus(models.RunTraceSpanStatusWaiting))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_WAITING, toTraceSpanStatus(models.RunTraceSpanStatusQueued))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_RUNNING, toTraceSpanStatus(models.RunTraceSpanStatusRunning))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_CANCELLED, toTraceSpanStatus(models.RunTraceSpanStatusCancelled))
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_SKIPPED, toTraceSpanStatus(models.RunTraceSpanStatusSkipped))
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
	outputID := mustEncodeSpanIdentifier(t, cqrs.SpanIdentifier{
		SpanID:      "span-output",
		InputSpanID: strPtr("span-input"),
		Preview:     boolPtr(true),
	})

	service := NewService(ServiceOptions{
		FunctionTraces: stubFunctionTraceReader{
			root: &cqrs.OtelSpan{
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
			},
			output: &cqrs.SpanOutput{
				Input: []byte(`{"message":"hello"}`),
				Data:  []byte(`{"ok":true}`),
			},
		},
	})

	t.Run("returns a nested trace response", func(t *testing.T) {
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
		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Run ID is required")
	})

	t.Run("validates run ID format", func(t *testing.T) {
		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{RunId: "not-a-ulid"})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Run ID must be a valid ULID")
	})

	t.Run("omits output when not requested", func(t *testing.T) {
		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{
			RunId:         runID.String(),
			IncludeOutput: boolPtr(false),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, resp.Data.RootSpan.Children[0].Input)
		require.Nil(t, resp.Data.RootSpan.Children[0].Output)
	})

	t.Run("applies max depth", func(t *testing.T) {
		maxDepth := uint32(1)
		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{
			RunId:    runID.String(),
			MaxDepth: &maxDepth,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.Data.RootSpan.ChildrenTruncated)
		require.Empty(t, resp.Data.RootSpan.Children)
	})

	t.Run("rejects oversized max depth", func(t *testing.T) {
		maxDepth := uint32(11)
		resp, err := service.GetFunctionTrace(context.Background(), &apiv2.GetFunctionTraceRequest{
			RunId:    runID.String(),
			MaxDepth: &maxDepth,
		})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "max_depth must be between 1 and 10")
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
