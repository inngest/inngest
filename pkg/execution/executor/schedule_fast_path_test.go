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
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestScheduleFastPathIntent(t *testing.T) {
	enqueueErr := errors.New("enqueue failed")
	for _, tc := range []struct {
		name       string
		fastPath   execution.FastPathOptions
		mode       enums.RunMode
		enqueueErr error
		wantErr    error
		wantNotify bool
	}{
		{name: "opted in", fastPath: execution.FastPathOptions{Enabled: true}, wantNotify: true},
		{name: "not opted in"},
		{name: "sync opted in", fastPath: execution.FastPathOptions{Enabled: true}, mode: enums.RunModeSync},
		{name: "enqueue failed", fastPath: execution.FastPathOptions{Enabled: true}, enqueueErr: enqueueErr, wantErr: enqueueErr},
		{name: "duplicate enqueue", fastPath: execution.FastPathOptions{Enabled: true}, enqueueErr: queue.ErrQueueItemExists, wantErr: state.ErrIdentifierExists},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shard := &scheduleFastPathShard{err: tc.enqueueErr}
			registry, err := queue.NewSingleShardRegistry(shard)
			require.NoError(t, err)
			q := &scheduleFastPathQueue{producer: queue.NewProducer(registry)}
			listener := &scheduleEnqueueListener{calls: make(chan enqueueNotification, 1), release: make(chan struct{})}
			defer close(listener.release)
			e := &executor{
				log:               logger.VoidLogger(),
				queue:             q,
				smv2:              &queueRoutingRunService{},
				tracerProvider:    tracing.NewNoopTracerProvider(),
				conditionalTracer: telemetrytrace.NoopConditionalTracer(),
				lifecycles:        []execution.LifecycleListener{listener},
			}
			req := execution.ScheduleRequest{
				AccountID: uuid.New(), WorkspaceID: uuid.New(), AppID: uuid.New(),
				Function: inngest.Function{ID: uuid.New(), FunctionVersion: 1, Name: "fast-path"},
				Events: []event.TrackedEvent{event.InternalEvent{ID: ulid.Make(), Event: event.Event{
					Name: "test/fast-path", Timestamp: time.Now().UnixMilli(), Data: map[string]any{},
				}}},
				RunMode: tc.mode, FastPath: tc.fastPath,
				IdempotencyKey: new("request-idempotency-key"),
				Context:        map[string]any{"caller": "original-request"},
			}
			done := make(chan error, 1)
			go func() {
				_, _, err := e.Schedule(t.Context(), req)
				done <- err
			}()
			select {
			case err := <-done:
				require.ErrorIs(t, err, tc.wantErr)
			case <-time.After(time.Second):
				t.Fatal("scheduling must not wait for the enqueue listener")
			}
			if tc.mode == enums.RunModeSync {
				require.Zero(t, q.calls, "sync runs do not enqueue")
			} else {
				require.Equal(t, 1, q.calls)
				require.Equal(t, tc.fastPath.Enabled, q.observed, "unopted requests must not install an enqueue observer")
			}
			if tc.wantNotify {
				select {
				case got := <-listener.calls:
					require.Equal(t, req.FastPath, got.req.FastPath)
					require.Equal(t, req.RunMode, got.req.RunMode)
					require.Equal(t, req.AccountID, got.req.AccountID)
					require.Equal(t, req.WorkspaceID, got.req.WorkspaceID)
					require.Equal(t, req.AppID, got.req.AppID)
					require.Equal(t, req.Function, got.req.Function)
					require.Equal(t, req.Events, got.req.Events)
					require.Equal(t, req.IdempotencyKey, got.req.IdempotencyKey)
					require.Equal(t, "original-request", got.req.Context["caller"])
					require.Equal(t, shard.stored, got.item, "notify with the finalized backend result")
					require.Equal(t, "persisted-item", got.item.ID)
					require.EqualValues(t, 7, got.item.GenerationID)
					require.Equal(t, shard.Name(), got.shard)
					require.NoError(t, got.ctx.Err(), "notification outlives the scheduling context")
				case <-time.After(time.Second):
					t.Fatal("opted-in durable enqueue did not notify")
				}
			} else {
				select {
				case <-listener.calls:
					t.Fatal("unexpected enqueue notification")
				case <-time.After(20 * time.Millisecond):
				}
			}
		})
	}
}

type scheduleFastPathShard struct {
	queue.QueueShard
	stored queue.QueueItem
	err    error
}

func (*scheduleFastPathShard) Name() string { return "selected-shard" }

func (s *scheduleFastPathShard) EnqueueItem(_ context.Context, item queue.QueueItem, _ time.Time, _ queue.EnqueueOpts) (queue.QueueItem, error) {
	item.ID, item.GenerationID, item.EnqueuedAt = "persisted-item", 7, 1234
	s.stored = item
	return item, s.err
}

type scheduleFastPathQueue struct {
	queue.Queue
	producer queue.Producer
	calls    int
	observed bool
}

func (q *scheduleFastPathQueue) Enqueue(ctx context.Context, item queue.Item, at time.Time, opts queue.EnqueueOpts) error {
	q.calls++
	q.observed = opts.OnEnqueued != nil
	return q.producer.Enqueue(ctx, item, at, opts)
}

type enqueueNotification struct {
	ctx   context.Context
	req   execution.ScheduleRequest
	item  queue.QueueItem
	shard string
}

type scheduleEnqueueListener struct {
	execution.NoopLifecyceListener
	calls   chan enqueueNotification
	release chan struct{}
}

func (l *scheduleEnqueueListener) OnFunctionEnqueued(ctx context.Context, req execution.ScheduleRequest, item queue.QueueItem, shard string) {
	l.calls <- enqueueNotification{ctx: ctx, req: req, item: item, shard: shard}
	<-l.release
}
