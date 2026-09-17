package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestScheduleQueueShardNotFoundCleansFreshState(t *testing.T) {
	deleteErr := errors.New("state store unavailable")
	tests := []struct {
		name             string
		deleteErr        error
		wantRoutingError bool
	}{
		{
			name:             "deletes state and returns routing error",
			wantRoutingError: true,
		},
		{
			name:             "already missing state returns routing error",
			deleteErr:        state.ErrRunNotFound,
			wantRoutingError: true,
		},
		{
			name:      "state cleanup failure remains retryable",
			deleteErr: deleteErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runService := &queueRoutingRunService{deleteErr: tt.deleteErr}
			e := &executor{
				log:               logger.VoidLogger(),
				queue:             queueRoutingErrorQueue{},
				smv2:              runService,
				tracerProvider:    tracing.NewNoopTracerProvider(),
				conditionalTracer: telemetrytrace.NoopConditionalTracer(),
			}

			eventID := ulid.Make()
			req := execution.ScheduleRequest{
				AccountID:   uuid.New(),
				WorkspaceID: uuid.New(),
				AppID:       uuid.New(),
				Function: inngest.Function{
					ID:              uuid.New(),
					FunctionVersion: 1,
					Name:            "Queue routing test",
				},
				Events: []event.TrackedEvent{
					event.InternalEvent{
						ID: eventID,
						Event: event.Event{
							ID:        eventID.String(),
							Name:      "test/routing",
							Timestamp: time.Now().UnixMilli(),
							Data:      map[string]any{},
						},
					},
				},
			}

			_, _, err := e.schedule(context.Background(), req, ulid.Make(), "test-key", false, nil)
			require.Error(t, err)
			require.Equal(t, tt.wantRoutingError, errors.Is(err, queue.ErrQueueShardNotFound))
			if tt.deleteErr != nil && !errors.Is(tt.deleteErr, state.ErrRunNotFound) {
				require.ErrorIs(t, err, tt.deleteErr)
			}
			require.Len(t, runService.deleted, 1)
		})
	}
}

type queueRoutingErrorQueue struct {
	queue.Queue
}

func (queueRoutingErrorQueue) Enqueue(context.Context, queue.Item, time.Time, queue.EnqueueOpts) error {
	return queue.ErrQueueShardNotFound
}

type queueRoutingRunService struct {
	sv2.RunService
	deleteErr error
	deleted   []sv2.ID
}

func (r *queueRoutingRunService) Create(_ context.Context, s sv2.CreateState) (sv2.State, error) {
	return sv2.State{Metadata: s.Metadata}, nil
}

func (r *queueRoutingRunService) Delete(_ context.Context, id sv2.ID, _ ...sv2.DeleteOption) error {
	r.deleted = append(r.deleted, id)
	return r.deleteErr
}
