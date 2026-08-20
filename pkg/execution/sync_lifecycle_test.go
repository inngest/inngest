package execution

import (
	"context"
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
