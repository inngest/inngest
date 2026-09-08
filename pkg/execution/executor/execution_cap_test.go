package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestScheduleAccountExecutionCap(t *testing.T) {
	pausedAt := time.Now().Add(-time.Minute)

	tests := []struct {
		name       string
		capFn      ExecutionCapFn
		pausedAt   *time.Time
		wantReason enums.SkipReason
	}{
		{
			name:       "exceeded and enforced skips with cap reason",
			capFn:      capDecision(ExecutionCapLimitDecision{Exceeded: true, Enforce: true}),
			wantReason: enums.SkipReasonAccountExecutionCapHit,
		},
		{
			name:       "no cap configured continues to normal skip checks",
			pausedAt:   &pausedAt,
			wantReason: enums.SkipReasonFunctionPaused,
		},
		{
			name:       "exceeded without enforce continues to normal skip checks",
			capFn:      capDecision(ExecutionCapLimitDecision{Exceeded: true, Enforce: false}),
			pausedAt:   &pausedAt,
			wantReason: enums.SkipReasonFunctionPaused,
		},
		{
			name:       "not exceeded continues to normal skip checks",
			capFn:      capDecision(ExecutionCapLimitDecision{Exceeded: false, Enforce: true}),
			pausedAt:   &pausedAt,
			wantReason: enums.SkipReasonFunctionPaused,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := newRecordingTracerProvider()
			e := &executor{
				log:                 logger.From(context.Background()),
				tracerProvider:      rec,
				conditionalTracer:   telemetrytrace.NoopConditionalTracer(),
				accountExecutionCap: tt.capFn,
			}

			req := capScheduleRequest(nil)
			req.FunctionPausedAt = tt.pausedAt

			_, _, err := e.schedule(context.Background(), req, ulid.Make(), "test-key", false, nil)
			require.ErrorIs(t, err, ErrFunctionSkipped)

			var skipped SkippedError
			require.True(t, errors.As(err, &skipped))
			require.Equal(t, tt.wantReason, skipped.Reason)

			var runSpan *createSpanCall
			for _, call := range rec.createCalls {
				if call.name == meta.SpanNameRun {
					runSpan = call
					break
				}
			}
			require.NotNil(t, runSpan)
			spanReason, ok := runSpan.opts.Attributes.Get(meta.Attrs.SkipReason.Key()).(*enums.SkipReason)
			require.True(t, ok)
			require.Equal(t, tt.wantReason, *spanReason)
		})
	}
}

func TestScheduleAccountExecutionCapSkipsSingletonHandling(t *testing.T) {
	sm := &recordingSingletonManager{}
	e := &executor{
		log:                 logger.From(context.Background()),
		tracerProvider:      newRecordingTracerProvider(),
		conditionalTracer:   telemetrytrace.NoopConditionalTracer(),
		singletonMgr:        sm,
		accountExecutionCap: capDecision(ExecutionCapLimitDecision{Exceeded: true, Enforce: true}),
	}

	req := capScheduleRequest(&inngest.Singleton{Mode: enums.SingletonModeCancel})

	_, _, err := e.schedule(context.Background(), req, ulid.Make(), "test-key", false, nil)

	var skipped SkippedError
	require.True(t, errors.As(err, &skipped))
	require.Equal(t, enums.SkipReasonAccountExecutionCapHit, skipped.Reason)
	require.Equal(t, 0, sm.calls)
}

func capDecision(d ExecutionCapLimitDecision) ExecutionCapFn {
	return func(context.Context, uuid.UUID) ExecutionCapLimitDecision { return d }
}

func capScheduleRequest(singleton *inngest.Singleton) execution.ScheduleRequest {
	eventID := ulid.Make()
	return execution.ScheduleRequest{
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       uuid.New(),
		Function: inngest.Function{
			ID:              uuid.New(),
			FunctionVersion: 1,
			Name:            "capped-fn",
			Singleton:       singleton,
		},
		Events: []event.TrackedEvent{
			event.InternalEvent{
				ID: eventID,
				Event: event.Event{
					ID:        eventID.String(),
					Name:      "test/cap",
					Timestamp: time.Now().UnixMilli(),
					Data:      map[string]any{},
				},
			},
		},
	}
}

type recordingSingletonManager struct {
	calls int
}

func (r *recordingSingletonManager) HandleSingleton(context.Context, queue.Scope, string, inngest.Singleton) (*ulid.ULID, error) {
	r.calls++
	return nil, nil
}
