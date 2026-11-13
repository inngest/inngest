package executor

import (
	"context"
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

func TestScheduleQueueRoutingErrorKeepsState(t *testing.T) {
	tests := []struct {
		name             string
		enqueueErr       error
		createErr        error
		wantStateExisted bool
	}{
		{
			name:             "shard not found keeps state created by this call",
			enqueueErr:       queue.ErrQueueShardNotFound,
			wantStateExisted: false,
		},
		{
			name:             "shard not found keeps state that already existed",
			enqueueErr:       queue.ErrQueueShardNotFound,
			createErr:        state.ErrIdentifierExists,
			wantStateExisted: true,
		},
		{
			name:             "shard unavailable keeps state created by this call",
			enqueueErr:       queue.ErrQueueShardUnavailable,
			wantStateExisted: false,
		},
		{
			name:             "shard unavailable keeps state that already existed",
			enqueueErr:       queue.ErrQueueShardUnavailable,
			createErr:        state.ErrIdentifierExists,
			wantStateExisted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runService := &queueRoutingRunService{createErr: tt.createErr}
			e := &executor{
				log: logger.VoidLogger(),
				queue: queueRoutingErrorQueue{
					err: tt.enqueueErr,
				},
				smv2:              runService,
				tracerProvider:    tracing.NewNoopTracerProvider(),
				conditionalTracer: telemetrytrace.NoopConditionalTracer(),
			}

			eventID := ulid.Make()
			runID := ulid.Make()
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

			_, _, err := e.schedule(context.Background(), req, runID, "test-key", false, nil)
			require.Error(t, err)
			require.ErrorIs(t, err, tt.enqueueErr)

			var routingErr *ScheduleQueueRoutingError
			require.ErrorAs(t, err, &routingErr)
			require.Equal(t, tt.wantStateExisted, routingErr.StateExisted)
			require.Equal(t, eventID, routingErr.EventInternalID)
			require.Equal(t, runID, routingErr.ID.RunID)
			require.Equal(t, req.Function.ID, routingErr.ID.FunctionID)
			require.Equal(t, req.AccountID, routingErr.ID.Tenant.AccountID)

			require.Empty(t, runService.deleted)
		})
	}
}

type queueRoutingErrorQueue struct {
	queue.Queue
	err error
}

func (q queueRoutingErrorQueue) Enqueue(context.Context, queue.Item, time.Time, queue.EnqueueOpts) error {
	return q.err
}

type queueRoutingRunService struct {
	sv2.RunService
	createErr error
	deleted   []sv2.ID
}

func (r *queueRoutingRunService) Create(_ context.Context, s sv2.CreateState) (sv2.State, error) {
	return sv2.State{Metadata: s.Metadata}, r.createErr
}

func (r *queueRoutingRunService) Delete(_ context.Context, id sv2.ID, _ ...sv2.DeleteOption) error {
	r.deleted = append(r.deleted, id)
	return nil
}
