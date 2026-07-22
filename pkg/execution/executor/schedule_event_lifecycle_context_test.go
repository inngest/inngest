package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	execstate "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestScheduleSkippedEventLifecycleReceivesEnrichedRequestContext(t *testing.T) {
	listener := &skippedRequestContextLifecycle{
		done: make(chan any, 1),
	}

	e := &executor{
		log:               logger.From(context.Background()),
		tracerProvider:    newRecordingTracerProvider(),
		conditionalTracer: telemetrytrace.NoopConditionalTracer(),
		evtLifecycles:     []execution.EventLifecycleListener{listener},
	}

	eventID := ulid.Make()
	pausedAt := time.Now().Add(-time.Minute)
	req := execution.ScheduleRequest{
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       uuid.New(),
		URL:         "https://example.com/api/inngest",
		Function: inngest.Function{
			ID:              uuid.New(),
			FunctionVersion: 1,
			Name:            "Send Weekly Email",
		},
		FunctionPausedAt: &pausedAt,
		Events: []event.TrackedEvent{
			event.InternalEvent{
				ID: eventID,
				Event: event.Event{
					ID:        eventID.String(),
					Name:      "test/schedule",
					Timestamp: time.Now().UnixMilli(),
					Data:      map[string]any{},
				},
			},
		},
	}

	_, _, err := e.schedule(context.Background(), req, ulid.Make(), "test-key", false, nil)
	require.ErrorIs(t, err, ErrFunctionSkipped)

	select {
	case observed := <-listener.done:
		require.Equal(t, req.URL, observed)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event lifecycle listener")
	}
}

func TestScheduleIdempotencySkippedEventLifecycleReceivesEnrichedRequestContext(t *testing.T) {
	listener := &idempotencySkippedRequestContextLifecycle{
		done: make(chan any, 1),
	}

	e := &executor{
		log:               logger.From(context.Background()),
		smv2:              tombstoneRunService{},
		tracerProvider:    newRecordingTracerProvider(),
		conditionalTracer: telemetrytrace.NoopConditionalTracer(),
		evtLifecycles:     []execution.EventLifecycleListener{listener},
	}

	eventID := ulid.Make()
	req := execution.ScheduleRequest{
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       uuid.New(),
		URL:         "https://example.com/api/inngest",
		Function: inngest.Function{
			ID:              uuid.New(),
			FunctionVersion: 1,
			Name:            "Send Weekly Email",
		},
		Events: []event.TrackedEvent{
			event.InternalEvent{
				ID: eventID,
				Event: event.Event{
					ID:        eventID.String(),
					Name:      "test/schedule",
					Timestamp: time.Now().UnixMilli(),
					Data:      map[string]any{},
				},
			},
		},
	}

	_, _, err := e.Schedule(context.Background(), req)
	require.ErrorIs(t, err, ErrFunctionSkippedIdempotency)

	select {
	case observed := <-listener.done:
		require.Equal(t, req.URL, observed)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event lifecycle listener")
	}
}

type skippedRequestContextLifecycle struct {
	execution.NoopEventLifecycleListener

	done chan any
}

func (l *skippedRequestContextLifecycle) OnFunctionSkipped(_ context.Context, req execution.ScheduleRequest, _ sv2.Metadata, _ enums.SkipReason) {
	l.done <- req.Context["url"]
}

type idempotencySkippedRequestContextLifecycle struct {
	execution.NoopEventLifecycleListener

	done chan any
}

func (l *idempotencySkippedRequestContextLifecycle) OnFunctionSkippedIdempotency(_ context.Context, req execution.ScheduleRequest, _ execution.IdempotencySkip) {
	l.done <- req.Context["url"]
}

type tombstoneRunService struct {
	sv2.RunService
}

func (t tombstoneRunService) Create(_ context.Context, s sv2.CreateState) (sv2.State, error) {
	return sv2.State{Metadata: s.Metadata}, execstate.ErrIdentifierTombstone
}
