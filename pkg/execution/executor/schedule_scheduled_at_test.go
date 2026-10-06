package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestScheduleSetsConfigScheduledAt(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)

	tests := []struct {
		name     string
		eventTs  time.Time
		at       time.Time
		batch    bool
		expected time.Time
	}{
		{
			name:     "no delay uses queued time",
			eventTs:  now,
			expected: now,
		},
		{
			name:     "explicit at is used",
			eventTs:  now,
			at:       now.Add(time.Hour),
			expected: now.Add(time.Hour),
		},
		{
			name:     "future event ts is used",
			eventTs:  now.Add(10 * time.Minute),
			expected: now.Add(10 * time.Minute),
		},
		{
			name:     "future event ts is ignored for batches",
			eventTs:  now.Add(10 * time.Minute),
			batch:    true,
			expected: now,
		},
		{
			name:     "past at is fudged to queued time",
			eventTs:  now,
			at:       now.Add(-time.Minute),
			expected: now,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			listener := &scheduledAtSkippedLifecycle{done: make(chan sv2.Metadata, 1)}
			e := &executor{
				log:               logger.From(context.Background()),
				clock:             clockwork.NewFakeClockAt(now),
				tracerProvider:    newRecordingTracerProvider(),
				conditionalTracer: telemetrytrace.NoopConditionalTracer(),
				evtLifecycles:     []execution.EventLifecycleListener{listener},
			}

			// Pause the function so scheduling stops at the skip path, which
			// hands the run's metadata to lifecycles without needing state.
			pausedAt := now.Add(-time.Minute)
			eventID := ulid.Make()
			req := execution.ScheduleRequest{
				AccountID:   uuid.New(),
				WorkspaceID: uuid.New(),
				AppID:       uuid.New(),
				Function: inngest.Function{
					ID:              uuid.New(),
					FunctionVersion: 1,
					Name:            "scheduled-at",
				},
				FunctionPausedAt: &pausedAt,
				Events: []event.TrackedEvent{
					event.InternalEvent{
						ID: eventID,
						Event: event.Event{
							ID:        eventID.String(),
							Name:      "test/schedule",
							Timestamp: tc.eventTs.UnixMilli(),
							Data:      map[string]any{},
						},
					},
				},
			}

			if !tc.at.IsZero() {
				req.At = &tc.at
			}
			if tc.batch {
				batchID := ulid.Make()
				req.BatchID = &batchID
			}

			runID := ulid.MustNew(ulid.Timestamp(now), nil)
			_, _, err := e.schedule(context.Background(), req, runID, "test-key", false, nil)
			require.ErrorIs(t, err, ErrFunctionSkipped)

			select {
			case md := <-listener.done:
				require.True(t, tc.expected.Equal(md.Config.ScheduledAt), "expected %s, got %s", tc.expected, md.Config.ScheduledAt)
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for event lifecycle listener")
			}
		})
	}
}

type scheduledAtSkippedLifecycle struct {
	execution.NoopEventLifecycleListener

	done chan sv2.Metadata
}

func (l *scheduledAtSkippedLifecycle) OnFunctionSkipped(_ context.Context, _ execution.ScheduleRequest, md sv2.Metadata, _ enums.SkipReason) {
	l.done <- md
}
