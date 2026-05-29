package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestScheduleRunSpanIncludesFunctionMetadata(t *testing.T) {
	rec := newRecordingTracerProvider()
	e := &executor{
		log:               logger.From(context.Background()),
		tracerProvider:    rec,
		conditionalTracer: telemetrytrace.NoopConditionalTracer(),
	}

	eventID := ulid.Make()
	pausedAt := time.Now().Add(-time.Minute)
	req := execution.ScheduleRequest{
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       uuid.New(),
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

	_, _, err := e.schedule(context.Background(), req, ulid.Make(), "test-key", false)
	require.ErrorIs(t, err, ErrFunctionSkipped)

	var runSpan *createSpanCall
	for _, call := range rec.createCalls {
		if call.name == meta.SpanNameRun {
			runSpan = call
			break
		}
	}
	require.NotNil(t, runSpan)

	functionName, ok := runSpan.opts.Attributes.Get(meta.Attrs.FunctionName.Key()).(*string)
	require.True(t, ok)
	require.Equal(t, "Send Weekly Email", *functionName)

	functionSlug, ok := runSpan.opts.Attributes.Get(meta.Attrs.FunctionSlug.Key()).(*string)
	require.True(t, ok)
	require.Equal(t, "send-weekly-email", *functionSlug)
}
