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
