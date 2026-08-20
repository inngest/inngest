package executor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/require"
)

type recordingSyncLifecycle struct {
	execution.NoopSyncLifecycleListener
	finishedCalls int
}

func (r *recordingSyncLifecycle) OnFunctionFinished(context.Context, sv2.Metadata, queue.Item, []json.RawMessage, statev1.DriverResponse, time.Time) {
	r.finishedCalls++
}

func TestRunFunctionFinishedLifecycleCallsSyncListenerInline(t *testing.T) {
	sync := &recordingSyncLifecycle{}
	e := &executor{
		syncLifecycles: []execution.SyncLifecycleListener{sync},
	}

	e.RunFunctionFinishedLifecycle(context.Background(), sv2.Metadata{}, queue.Item{}, nil, statev1.DriverResponse{})

	// No channel/timeout wait needed: if this passes without any
	// synchronization, the call happened inline before
	// RunFunctionFinishedLifecycle returned — proving synchronous dispatch.
	require.Equal(t, 1, sync.finishedCalls)
}

type asyncOnlyLifecycle struct {
	execution.NoopLifecyceListener
	done chan struct{}
}

func (a *asyncOnlyLifecycle) OnFunctionFinished(context.Context, sv2.Metadata, queue.Item, []json.RawMessage, statev1.DriverResponse) {
	close(a.done)
}

func TestRunFunctionFinishedLifecycleSyncListenerDoesNotAffectAsyncListeners(t *testing.T) {
	async := &asyncOnlyLifecycle{done: make(chan struct{})}
	sync := &recordingSyncLifecycle{}
	e := &executor{
		lifecycles:     []execution.LifecycleListener{async},
		syncLifecycles: []execution.SyncLifecycleListener{sync},
	}

	e.RunFunctionFinishedLifecycle(context.Background(), sv2.Metadata{}, queue.Item{}, nil, statev1.DriverResponse{})

	// The sync listener already ran inline by the time we get here.
	require.Equal(t, 1, sync.finishedCalls)

	// The async listener runs on its own goroutine — it must eventually run,
	// but is not required to have run yet.
	select {
	case <-async.done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async listener")
	}
}
