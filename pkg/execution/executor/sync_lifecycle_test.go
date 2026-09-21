package executor

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
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

// TestSyncLifecycleRegistrationIsStaticOptionOnly locks the registration
// contract documented on execution.SyncLifecycleListener: sync listeners are
// supplied once at construction, via WithSyncLifecycleListeners, and there is
// deliberately no post-construction mutator. A dynamic AddSyncLifecycleListener
// (the counterpart AddLifecycleListener provides for async listeners) would let
// a consumer register with some dispatchers but not others, leaving a listener
// observing only part of a run's lifecycle -- so its absence is part of the
// contract, not an oversight.
func TestSyncLifecycleRegistrationIsStaticOptionOnly(t *testing.T) {
	// The option is a real registration path: a listener supplied through it
	// lands in the same slice the dispatchers fan out over.
	sync := &recordingSyncLifecycle{}
	e := &executor{log: logger.VoidLogger()}
	require.NoError(t, WithSyncLifecycleListeners(sync)(e))

	e.RunFunctionFinishedLifecycle(context.Background(), sv2.Metadata{}, queue.Item{}, nil, statev1.DriverResponse{})
	require.Equal(t, 1, sync.finishedCalls)

	// ...and it is the *only* registration path. Neither the concrete
	// *executor nor the execution.Executor interface may expose a mutator for
	// sync listeners.
	for _, typ := range []reflect.Type{
		reflect.TypeOf(&executor{}),
		reflect.TypeOf((*execution.Executor)(nil)).Elem(),
	} {
		for i := 0; i < typ.NumMethod(); i++ {
			name := typ.Method(i).Name
			if !strings.Contains(name, "Sync") {
				continue
			}
			require.False(t,
				strings.HasPrefix(name, "Add") || strings.HasPrefix(name, "Set") || strings.HasPrefix(name, "Register"),
				"%s.%s: sync lifecycle listeners are registered statically via WithSyncLifecycleListeners; adding a runtime mutator breaks that contract",
				typ, name,
			)
		}
	}
}
