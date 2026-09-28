package execution

import (
	"context"
	"reflect"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

// spyLogger records ErrorContext calls without requiring every logger.Logger
// method to be implemented -- only ErrorContext is ever expected to be
// exercised here.
type spyLogger struct {
	logger.Logger
	errorCalls []string
}

func (s *spyLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	s.errorCalls = append(s.errorCalls, msg)
}

func TestNoopSyncLifecycleListenerSatisfiesInterface(t *testing.T) {
	var l SyncLifecycleListener = NoopSyncLifecycleListener{}
	require.NotPanics(t, func() {
		l.OnEventReceived(context.Background(), event.NewBaseTrackedEvent(event.Event{Name: "test"}, nil))
	})
}

// TestSafelyInvokeSyncListeners_RecoversPanic proves a panicking hook is
// recovered and logged, rather than propagating into the caller's critical
// path (execution, checkpointing, defer handling, OTLP ingestion), and that
// later listeners in the same fan-out still run.
func TestSafelyInvokeSyncListeners_RecoversPanic(t *testing.T) {
	spy := &spyLogger{}
	secondRan := false

	require.NotPanics(t, func() {
		SafelyInvokeSyncListeners(context.Background(), spy, []SyncLifecycleListener{
			panicListener{},
			NoopSyncLifecycleListener{},
		}, "OnFunctionFinished", func(l SyncLifecycleListener) {
			if _, ok := l.(panicListener); ok {
				panic("boom")
			}
			secondRan = true
		})
	})

	require.Len(t, spy.errorCalls, 1)
	require.True(t, secondRan)
}

type panicListener struct {
	NoopSyncLifecycleListener
}

// TestSafelyInvokeSyncListeners_RunsFnNormally proves a non-panicking hook
// still runs to completion and doesn't spuriously log an error.
func TestSafelyInvokeSyncListeners_RunsFnNormally(t *testing.T) {
	spy := &spyLogger{}
	ran := false

	SafelyInvokeSyncListeners(context.Background(), spy, []SyncLifecycleListener{NoopSyncLifecycleListener{}}, "OnFunctionFinished", func(l SyncLifecycleListener) {
		ran = true
	})

	require.True(t, ran)
	require.Empty(t, spy.errorCalls)
}

// TestNoopSyncLifecycleListenerImplementsEveryHook makes sure the Noop
// implementation actually stays in step with the interface: embedding it is
// how every implementation (including pkg/execution/dualwrite's) opts out of
// the hooks it does not care about, so a hook missing here is a compile error
// in every embedder rather than a no-op.
func TestNoopSyncLifecycleListenerImplementsEveryHook(t *testing.T) {
	syncType := reflect.TypeOf((*SyncLifecycleListener)(nil)).Elem()
	noopType := reflect.TypeOf(NoopSyncLifecycleListener{})

	for i := 0; i < syncType.NumMethod(); i++ {
		name := syncType.Method(i).Name
		_, ok := noopType.MethodByName(name)
		require.True(t, ok, "NoopSyncLifecycleListener is missing %s", name)
	}
}
