package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// waitForSignalStubPauseMgr stubs pauses.Manager.Write so each subtest can
// force one of the three dispositions handleGeneratorWaitForSignal branches
// on.
type waitForSignalStubPauseMgr struct {
	pauses.Manager
	writeErr error
}

func (m *waitForSignalStubPauseMgr) Write(ctx context.Context, index pauses.Index, ps ...*state.Pause) (int, error) {
	return 0, m.writeErr
}

// waitForSignalStubQueue records every enqueued item and lets a subtest force
// the Enqueue call to fail, mirroring a racing replay that already enqueued
// the same job ID.
type waitForSignalStubQueue struct {
	queue.Queue
	enqueueErr error
	enqueued   []queue.Item
}

func (q *waitForSignalStubQueue) Enqueue(_ context.Context, item queue.Item, _ time.Time, _ queue.EnqueueOpts) error {
	q.enqueued = append(q.enqueued, item)
	return q.enqueueErr
}

// waitForSignalLifecycleRecorder records OnWaitForSignal calls via a buffered
// channel so tests can assert whether the hook fired without racing the
// goroutine the handler fires it from.
type waitForSignalLifecycleRecorder struct {
	execution.NoopLifecyceListener
	fired chan struct{}
}

func newWaitForSignalLifecycleRecorder() *waitForSignalLifecycleRecorder {
	return &waitForSignalLifecycleRecorder{fired: make(chan struct{}, 1)}
}

func (r *waitForSignalLifecycleRecorder) OnWaitForSignal(
	context.Context,
	sv2.Metadata,
	queue.Item,
	state.GeneratorOpcode,
	state.Pause,
) {
	r.fired <- struct{}{}
}

func (r *waitForSignalLifecycleRecorder) didFire(t *testing.T) bool {
	t.Helper()
	select {
	case <-r.fired:
		return true
	case <-time.After(time.Second):
		return false
	}
}

func newWaitForSignalRunContext() *mockRunContext {
	return &mockRunContext{
		md: sv2.Metadata{
			ID:     sv2.ID{RunID: ulid.MustNew(ulid.Now(), nil), FunctionID: uuid.New()},
			Config: *sv2.InitConfig(&sv2.Config{}),
		},
	}
}

// TestWaitForSignal_Conflict_AlreadyExists_QueueExists pins the three error
// dispositions handleGeneratorWaitForSignal takes when writing the signal
// pause and enqueueing its timeout job: a signal conflict fails the step, an
// already-existing pause continues on to enqueue, and an already-enqueued
// timeout job is treated as a no-op replay.
func TestWaitForSignal_Conflict_AlreadyExists_QueueExists(t *testing.T) {
	edge := queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: "wait-for-signal"}}
	gen := state.GeneratorOpcode{
		Op:   enums.OpcodeWaitForSignal,
		ID:   "wait-for-signal",
		Opts: map[string]any{"signal": "some-signal"},
	}

	t.Run("signal conflict fails the step", func(t *testing.T) {
		q := &waitForSignalStubQueue{}
		recorder := newWaitForSignalLifecycleRecorder()
		e := &executor{
			pm:             &waitForSignalStubPauseMgr{writeErr: state.ErrSignalConflict},
			queue:          q,
			log:            logger.From(context.Background()),
			tracerProvider: tracing.NewNoopTracerProvider(),
			lifecycles:     []execution.LifecycleListener{recorder},
		}

		err := e.handleGeneratorWaitForSignal(context.Background(), newWaitForSignalRunContext(), gen, edge, OpcodeGroup{})
		require.Error(t, err)
		require.ErrorContains(t, err, "Signal conflict")
		require.Empty(t, q.enqueued, "a conflicting signal wait must not enqueue a timeout job")
		require.False(t, recorder.didFire(t), "a failed step must not fire OnWaitForSignal")
	})

	t.Run("pause already exists continues to enqueue", func(t *testing.T) {
		q := &waitForSignalStubQueue{}
		recorder := newWaitForSignalLifecycleRecorder()
		e := &executor{
			pm:             &waitForSignalStubPauseMgr{writeErr: state.ErrPauseAlreadyExists},
			queue:          q,
			log:            logger.From(context.Background()),
			tracerProvider: tracing.NewNoopTracerProvider(),
			lifecycles:     []execution.LifecycleListener{recorder},
		}

		err := e.handleGeneratorWaitForSignal(context.Background(), newWaitForSignalRunContext(), gen, edge, OpcodeGroup{})
		require.NoError(t, err)
		require.Len(t, q.enqueued, 1, "an already-existing pause must still enqueue the timeout job")
		require.True(t, recorder.didFire(t), "a successful enqueue must fire OnWaitForSignal")
	})

	t.Run("queue item already exists is a no-op replay", func(t *testing.T) {
		q := &waitForSignalStubQueue{enqueueErr: queue.ErrQueueItemExists}
		recorder := newWaitForSignalLifecycleRecorder()
		e := &executor{
			pm:             &waitForSignalStubPauseMgr{},
			queue:          q,
			log:            logger.From(context.Background()),
			tracerProvider: tracing.NewNoopTracerProvider(),
			lifecycles:     []execution.LifecycleListener{recorder},
		}

		err := e.handleGeneratorWaitForSignal(context.Background(), newWaitForSignalRunContext(), gen, edge, OpcodeGroup{})
		require.NoError(t, err, "a replayed timeout enqueue must not surface an error")
		require.False(t, recorder.didFire(t), "a replayed enqueue must not re-fire OnWaitForSignal")
	})
}
