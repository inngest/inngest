package executor

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

type handleResponseTestCtxKey string

const handleResponseTestCtxValueKey handleResponseTestCtxKey = "test-value"

// timingRecordingSyncListener records everything OnStepFinished needs to
// prove the sync-dispatch timing/context contract.
type timingRecordingSyncListener struct {
	execution.NoopSyncLifecycleListener
	called      bool
	gotCtx      context.Context
	gotReqStart time.Time
	gotNow      time.Time
}

func (l *timingRecordingSyncListener) OnStepFinished(ctx context.Context, _ sv2.Metadata, _ queue.Item, _ inngest.Edge, _ *state.DriverResponse, _ error, reqStart time.Time, now time.Time) {
	l.called = true
	l.gotCtx = ctx
	l.gotReqStart = reqStart
	l.gotNow = now
}

// timingRecordingLegacyListener records the context OnStepFinished was
// invoked with on the legacy (async, goroutine-dispatched) path.
type timingRecordingLegacyListener struct {
	execution.NoopLifecyceListener
	gotCtx context.Context
	done   chan struct{}
}

func (l *timingRecordingLegacyListener) OnStepFinished(ctx context.Context, _ sv2.Metadata, _ queue.Item, _ inngest.Edge, _ *state.DriverResponse, _ error) {
	l.gotCtx = ctx
	close(l.done)
}

// TestHandleResponse_OnStepFinishedTimingAndContextContract proves two
// properties HandleResponse's OnStepFinished dispatch relies on but no
// existing test protected:
//
//  1. reqStart (captured before the SDK request was dispatched) and now (the
//     completion timestamp passed to sync listeners) are distinct instants,
//     with now strictly after reqStart -- not derived from one another.
//  2. Sync listeners are invoked with the original ctx (so they observe
//     cancellation and any values on it), while legacy/async listeners are
//     invoked with context.WithoutCancel(ctx) (so they intentionally do not
//     observe cancellation, though values still propagate).
func TestHandleResponse_OnStepFinishedTimingAndContextContract(t *testing.T) {
	fakeClock := clockwork.NewFakeClock()
	reqStart := fakeClock.Now()
	// Simulate time passing while the (fake) SDK request was in flight.
	fakeClock.Advance(5 * time.Second)

	sync := &timingRecordingSyncListener{}
	legacy := &timingRecordingLegacyListener{done: make(chan struct{})}

	e := &executor{
		log:            logger.VoidLogger(),
		clock:          fakeClock,
		tracerProvider: tracing.NewNoopTracerProvider(),
		lifecycles:     []execution.LifecycleListener{legacy},
		syncLifecycles: []execution.SyncLifecycleListener{sync},
	}

	// A driver error response with no NoRetry/final flag is Retryable(),
	// routing HandleResponse through the branch that dispatches
	// OnStepFinished without requiring further executor/state-store
	// machinery (HandleGeneratorResponse, Finalize, etc.).
	errMsg := "boom"
	inst := &runInstance{
		md:   sv2.Metadata{},
		resp: &state.DriverResponse{Err: &errMsg},
		// reqStart is normally set once, before dispatching the SDK request;
		// simulate that here directly on the instance.
	}
	inst.reqStart = reqStart

	ctx := context.WithValue(context.Background(), handleResponseTestCtxValueKey, "present")
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	err := e.HandleResponse(ctx, inst)
	require.NoError(t, err)

	// Timing contract: reqStart and now are distinct, with now strictly
	// after reqStart once the clock advances.
	require.True(t, sync.called)
	require.Equal(t, reqStart, sync.gotReqStart)
	require.Equal(t, fakeClock.Now(), sync.gotNow)
	require.True(t, sync.gotNow.After(sync.gotReqStart))

	// Context contract: the sync listener sees the original ctx -- the
	// value survives and cancellation is observable.
	require.Equal(t, "present", sync.gotCtx.Value(handleResponseTestCtxValueKey))
	require.Error(t, sync.gotCtx.Err())

	select {
	case <-legacy.done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for legacy listener")
	}
	// Context contract: the legacy listener sees context.WithoutCancel(ctx)
	// -- the value still survives, but cancellation is NOT observable.
	require.Equal(t, "present", legacy.gotCtx.Value(handleResponseTestCtxValueKey))
	require.NoError(t, legacy.gotCtx.Err())
}
