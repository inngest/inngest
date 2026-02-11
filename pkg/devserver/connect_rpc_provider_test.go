package devserver

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	apiv2pb "github.com/inngest/inngest/proto/gen/api/v2"
	rpbv2 "github.com/inngest/inngest/proto/gen/run/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRunStatusString(t *testing.T) {
	testCases := []struct {
		name     string
		status   enums.RunStatus
		expected string
	}{
		{"Scheduled", enums.RunStatusScheduled, "QUEUED"},
		{"Running", enums.RunStatusRunning, "RUNNING"},
		{"Completed", enums.RunStatusCompleted, "COMPLETED"},
		{"Failed", enums.RunStatusFailed, "FAILED"},
		{"Overflowed", enums.RunStatusOverflowed, "FAILED"},
		{"Cancelled", enums.RunStatusCancelled, "CANCELLED"},
		{"Skipped", enums.RunStatusSkipped, "SKIPPED"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := runStatusString(tc.status)
			require.Equal(t, tc.expected, result)
		})
	}

	t.Run("unknown status returns UNKNOWN", func(t *testing.T) {
		result := runStatusString(enums.RunStatus(999))
		require.Equal(t, "UNKNOWN", result)
	})
}

func TestMapSpanStatus(t *testing.T) {
	testCases := []struct {
		name     string
		input    rpbv2.SpanStatus
		expected apiv2pb.RunTraceSpanStatus
	}{
		{"RUNNING", rpbv2.SpanStatus_RUNNING, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_RUNNING},
		{"WAITING", rpbv2.SpanStatus_WAITING, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_WAITING},
		{"COMPLETED", rpbv2.SpanStatus_COMPLETED, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_COMPLETED},
		{"OK", rpbv2.SpanStatus_OK, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_COMPLETED},
		{"FAILED", rpbv2.SpanStatus_FAILED, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_FAILED},
		{"ERORR", rpbv2.SpanStatus_ERORR, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_FAILED},
		{"CANCELLED", rpbv2.SpanStatus_CANCELLED, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_CANCELLED},
		{"QUEUED", rpbv2.SpanStatus_QUEUED, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_QUEUED},
		{"SCHEDULED", rpbv2.SpanStatus_SCHEDULED, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_QUEUED},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapSpanStatus(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}

	t.Run("unknown status returns UNSPECIFIED", func(t *testing.T) {
		result := mapSpanStatus(rpbv2.SpanStatus(999))
		require.Equal(t, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_UNSPECIFIED, result)
	})
}

func TestMapStepOp(t *testing.T) {
	testCases := []struct {
		name     string
		input    rpbv2.SpanStepOp
		expected apiv2pb.StepOp
	}{
		{"INVOKE", rpbv2.SpanStepOp_INVOKE, apiv2pb.StepOp_STEP_OP_INVOKE},
		{"SLEEP", rpbv2.SpanStepOp_SLEEP, apiv2pb.StepOp_STEP_OP_SLEEP},
		{"WAIT_FOR_EVENT", rpbv2.SpanStepOp_WAIT_FOR_EVENT, apiv2pb.StepOp_STEP_OP_WAIT_FOR_EVENT},
		{"AI_GATEWAY", rpbv2.SpanStepOp_AI_GATEWAY, apiv2pb.StepOp_STEP_OP_AI_GATEWAY},
		{"WAIT_FOR_SIGNAL", rpbv2.SpanStepOp_WAIT_FOR_SIGNAL, apiv2pb.StepOp_STEP_OP_WAIT_FOR_SIGNAL},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapStepOp(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}

	t.Run("unknown op returns RUN", func(t *testing.T) {
		result := mapStepOp(rpbv2.SpanStepOp(999))
		require.Equal(t, apiv2pb.StepOp_STEP_OP_RUN, result)
	})
}

func TestMapStepInfo(t *testing.T) {
	t.Run("returns nil for nil input", func(t *testing.T) {
		result := mapStepInfo(nil)
		require.Nil(t, result)
	})

	t.Run("maps invoke step info", func(t *testing.T) {
		timeout := timestamppb.New(time.Now().Add(5 * time.Minute))
		input := &rpbv2.StepInfo{
			Info: &rpbv2.StepInfo_Invoke{
				Invoke: &rpbv2.StepInfoInvoke{
					TriggeringEventId: "evt-123",
					FunctionId:        "fn-456",
					Timeout:           timeout,
				},
			},
		}

		result := mapStepInfo(input)

		require.NotNil(t, result)
		invoke := result.GetInvoke()
		require.NotNil(t, invoke)
		require.Equal(t, "evt-123", invoke.TriggeringEventId)
		require.Equal(t, "fn-456", invoke.FunctionId)
	})

	t.Run("maps sleep step info", func(t *testing.T) {
		sleepUntil := timestamppb.New(time.Now().Add(10 * time.Minute))
		input := &rpbv2.StepInfo{
			Info: &rpbv2.StepInfo_Sleep{
				Sleep: &rpbv2.StepInfoSleep{
					SleepUntil: sleepUntil,
				},
			},
		}

		result := mapStepInfo(input)

		require.NotNil(t, result)
		sleep := result.GetSleep()
		require.NotNil(t, sleep)
		require.NotEmpty(t, sleep.SleepUntil)
	})

	t.Run("maps wait for event step info", func(t *testing.T) {
		timeout := timestamppb.New(time.Now().Add(5 * time.Minute))
		expression := "event.data.id == async.data.id"
		input := &rpbv2.StepInfo{
			Info: &rpbv2.StepInfo_Wait{
				Wait: &rpbv2.StepInfoWaitForEvent{
					EventName:  "user/signup.completed",
					Timeout:    timeout,
					Expression: &expression,
				},
			},
		}

		result := mapStepInfo(input)

		require.NotNil(t, result)
		wait := result.GetWaitForEvent()
		require.NotNil(t, wait)
		require.Equal(t, "user/signup.completed", wait.EventName)
		require.NotNil(t, wait.Expression)
		require.Equal(t, expression, *wait.Expression)
	})

	t.Run("maps run step info", func(t *testing.T) {
		stepType := "custom"
		input := &rpbv2.StepInfo{
			Info: &rpbv2.StepInfo_Run{
				Run: &rpbv2.StepInfoRun{
					Type: &stepType,
				},
			},
		}

		result := mapStepInfo(input)

		require.NotNil(t, result)
		run := result.GetRun()
		require.NotNil(t, run)
		require.NotNil(t, run.Type)
		require.Equal(t, "custom", *run.Type)
	})

	t.Run("maps wait for signal step info", func(t *testing.T) {
		timeout := timestamppb.New(time.Now().Add(5 * time.Minute))
		input := &rpbv2.StepInfo{
			Info: &rpbv2.StepInfo_WaitForSignal{
				WaitForSignal: &rpbv2.StepInfoWaitForSignal{
					Signal:  "approval",
					Timeout: timeout,
				},
			},
		}

		result := mapStepInfo(input)

		require.NotNil(t, result)
		waitSignal := result.GetWaitForSignal()
		require.NotNil(t, waitSignal)
		require.Equal(t, "approval", waitSignal.Signal)
	})
}

func TestMapUserlandSpan(t *testing.T) {
	t.Run("returns nil for nil input", func(t *testing.T) {
		result := mapUserlandSpan(nil)
		require.Nil(t, result)
	})

	t.Run("maps all fields", func(t *testing.T) {
		serviceName := "my-service"
		scopeName := "my-scope"
		scopeVersion := "1.0.0"
		input := &rpbv2.UserlandSpan{
			SpanName:      "test-span",
			SpanKind:      "internal",
			ServiceName:   &serviceName,
			ScopeName:     &scopeName,
			ScopeVersion:  &scopeVersion,
			SpanAttrs:     []byte(`{"key":"value"}`),
			ResourceAttrs: []byte(`{"service":"test"}`),
		}

		result := mapUserlandSpan(input)

		require.NotNil(t, result)
		require.NotNil(t, result.SpanName)
		require.Equal(t, "test-span", *result.SpanName)
		require.NotNil(t, result.SpanKind)
		require.Equal(t, "internal", *result.SpanKind)
		require.NotNil(t, result.ServiceName)
		require.Equal(t, "my-service", *result.ServiceName)
		require.NotNil(t, result.SpanAttrs)
		require.NotNil(t, result.ResourceAttrs)
	})

	t.Run("handles empty fields", func(t *testing.T) {
		input := &rpbv2.UserlandSpan{}

		result := mapUserlandSpan(input)

		require.NotNil(t, result)
		require.Nil(t, result.SpanName)
		require.Nil(t, result.SpanKind)
	})
}

func TestRunSpanToRunTraceSpan(t *testing.T) {
	t.Run("returns nil for nil input", func(t *testing.T) {
		result := runSpanToRunTraceSpan(nil)
		require.Nil(t, result)
	})

	t.Run("maps basic fields", func(t *testing.T) {
		queuedAt := timestamppb.New(time.Now())
		input := &rpbv2.RunSpan{
			Name:       "test-span",
			Status:     rpbv2.SpanStatus_RUNNING,
			QueuedAt:   queuedAt,
			IsRoot:     true,
			SpanId:     "span-123",
			IsUserland: false,
		}

		result := runSpanToRunTraceSpan(input)

		require.NotNil(t, result)
		require.Equal(t, "test-span", result.Name)
		require.Equal(t, apiv2pb.RunTraceSpanStatus_RUN_TRACE_SPAN_STATUS_RUNNING, result.Status)
		require.True(t, result.IsRoot)
		require.Equal(t, "span-123", result.SpanId)
		require.False(t, result.IsUserland)
	})

	t.Run("maps optional fields when present", func(t *testing.T) {
		queuedAt := timestamppb.New(time.Now())
		startedAt := timestamppb.New(time.Now())
		endedAt := timestamppb.New(time.Now().Add(time.Second))
		outputID := "output-123"
		stepID := "step-456"
		stepOp := rpbv2.SpanStepOp_SLEEP

		input := &rpbv2.RunSpan{
			Name:      "test-span",
			Status:    rpbv2.SpanStatus_COMPLETED,
			QueuedAt:  queuedAt,
			StartedAt: startedAt,
			EndedAt:   endedAt,
			Attempts:  3,
			OutputId:  &outputID,
			StepId:    &stepID,
			StepOp:    &stepOp,
		}

		result := runSpanToRunTraceSpan(input)

		require.NotNil(t, result)
		require.NotNil(t, result.StartedAt)
		require.NotNil(t, result.EndedAt)
		require.NotNil(t, result.Attempts)
		require.Equal(t, int32(3), *result.Attempts)
		require.NotNil(t, result.OutputId)
		require.Equal(t, "output-123", *result.OutputId)
		require.NotNil(t, result.StepId)
		require.Equal(t, "step-456", *result.StepId)
		require.NotNil(t, result.StepOp)
		require.Equal(t, apiv2pb.StepOp_STEP_OP_SLEEP, *result.StepOp)
	})

	t.Run("recursively maps children", func(t *testing.T) {
		queuedAt := timestamppb.New(time.Now())
		input := &rpbv2.RunSpan{
			Name:     "parent",
			Status:   rpbv2.SpanStatus_RUNNING,
			QueuedAt: queuedAt,
			Children: []*rpbv2.RunSpan{
				{Name: "child-1", Status: rpbv2.SpanStatus_COMPLETED, QueuedAt: queuedAt},
				{Name: "child-2", Status: rpbv2.SpanStatus_RUNNING, QueuedAt: queuedAt},
			},
		}

		result := runSpanToRunTraceSpan(input)

		require.NotNil(t, result)
		require.Len(t, result.ChildrenSpans, 2)
		require.Equal(t, "child-1", result.ChildrenSpans[0].Name)
		require.Equal(t, "child-2", result.ChildrenSpans[1].Name)
	})
}

func TestFormatTime(t *testing.T) {
	t.Run("formats time in RFC3339Nano UTC", func(t *testing.T) {
		ts := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)
		result := formatTime(ts)

		require.Equal(t, "2024-01-15T10:30:45.123456789Z", result)
	})

	t.Run("converts non-UTC to UTC", func(t *testing.T) {
		loc, _ := time.LoadLocation("America/New_York")
		ts := time.Date(2024, 1, 15, 5, 30, 45, 0, loc)
		result := formatTime(ts)

		require.Contains(t, result, "T10:30:45")
		require.Contains(t, result, "Z")
	})
}

func TestFormatTimePtr(t *testing.T) {
	t.Run("returns pointer to formatted time", func(t *testing.T) {
		ts := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
		result := formatTimePtr(ts)

		require.NotNil(t, result)
		require.Equal(t, "2024-01-15T10:30:45Z", *result)
	})
}

func TestNewConnectRPCProvider(t *testing.T) {
	t.Run("creates provider with data manager", func(t *testing.T) {
		//
		// We can only test the constructor without a real cqrs.Manager
		provider := NewConnectRPCProvider(nil)
		require.NotNil(t, provider)
	})
}
