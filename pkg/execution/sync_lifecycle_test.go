package execution

import (
	"context"
	"reflect"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
)

func TestNoopSyncLifecycleListenerSatisfiesInterface(t *testing.T) {
	var l SyncLifecycleListener = NoopSyncLifecycleListener{}
	require.NotPanics(t, func() {
		l.OnEventReceived(context.Background(), event.NewBaseTrackedEvent(event.Event{Name: "test"}, nil))
	})
}

// TestSyncLifecycleListenerSignaturesMatchAsync pins the invariant the spec's
// Architecture section relies on: every hook SyncLifecycleListener shares a
// name with on the async LifecycleListener must take exactly the same
// arguments. The executor dispatches both from the same call site with the
// same values, so a signature that drifts here is a call site that silently
// stops compiling — or worse, one that compiles against a subtly different
// argument list.
func TestSyncLifecycleListenerSignaturesMatchAsync(t *testing.T) {
	syncType := reflect.TypeOf((*SyncLifecycleListener)(nil)).Elem()
	asyncType := reflect.TypeOf((*LifecycleListener)(nil)).Elem()

	shared := 0
	for i := 0; i < syncType.NumMethod(); i++ {
		m := syncType.Method(i)
		am, ok := asyncType.MethodByName(m.Name)
		if !ok {
			// OnEventReceived has no async counterpart by design — the async
			// EventLifecycleListener's hooks describe per-match scheduling
			// decisions instead.
			require.Equal(t, "OnEventReceived", m.Name, "unexpected sync-only hook %s", m.Name)
			continue
		}
		shared++
		require.Equal(t, am.Type.String(), m.Type.String(), "signature drift on %s", m.Name)
	}

	// The 7 hooks from the original interface plus the 8 generator-opcode
	// hooks the spec's Architecture section calls for (sleep, wait for
	// event/signal and their resumes, invoke and its resume, and the step
	// gateway request).
	require.Equal(t, 15, shared)
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
