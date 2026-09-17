package runner

import (
	"context"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/stretchr/testify/require"
)

type recordingSyncLifecycle struct {
	execution.NoopSyncLifecycleListener
	received []event.TrackedEvent
}

func (r *recordingSyncLifecycle) OnEventReceived(_ context.Context, evt event.TrackedEvent) {
	r.received = append(r.received, evt)
}

func TestSvcNotifiesSyncLifecycleOnEventReceived(t *testing.T) {
	sync := &recordingSyncLifecycle{}
	s := &svc{syncLifecycles: []execution.SyncLifecycleListener{sync}}

	tracked := event.NewBaseTrackedEvent(event.Event{Name: "test/event"}, nil)
	s.notifySyncLifecyclesEventReceived(context.Background(), tracked)

	require.Len(t, sync.received, 1)
	require.Equal(t, "test/event", sync.received[0].GetEvent().Name)
}
