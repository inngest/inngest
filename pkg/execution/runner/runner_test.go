package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/actionloader"
	"github.com/inngest/inngest-cli/pkg/execution/driver"
	"github.com/inngest/inngest-cli/pkg/execution/driver/mockdriver"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/stretchr/testify/require"
)

// stateManager wrpas the inmemory.Queue implementation, storing everything that
// is enqueued within a slice.
type stateManager struct {
	inmemory.Queue

	// store a slice of everything that has or will be enqueued via the in-memory
	// queue.  This allows us to assert that the runner correctly enqueues items
	// without having to worry about timings and channels.
	queue []enqueued
}

type enqueued struct {
	item inmemory.QueueItem
	at   time.Time
}

func (m *stateManager) Enqueue(item inmemory.QueueItem, at time.Time) {
	m.queue = append(m.queue, enqueued{item: item, at: at})
	m.Queue.Enqueue(item, at)
}

func newRunner(t *testing.T, sm inmemory.Queue, d *mockdriver.Mock) *InMemoryRunner {
	t.Helper()

	al := actionloader.NewMemoryLoader()
	al.Add(inngest.ActionVersion{
		DSN: "step-a",
		Runtime: inngest.RuntimeWrapper{
			Runtime: &mockdriver.Mock{},
		},
	})
	al.Add(inngest.ActionVersion{
		DSN: "step-b",
		Runtime: inngest.RuntimeWrapper{
			Runtime: &mockdriver.Mock{},
		},
	})

	if d == nil {
		d = &mockdriver.Mock{}
	}

	exec, err := executor.NewExecutor(
		executor.WithStateManager(sm),
		executor.WithActionLoader(al),
		executor.WithRuntimeDrivers(d),
	)
	require.NoError(t, err)

	return NewInMemoryRunner(sm, exec)
}

func TestRunner_new(t *testing.T) {
	ctx := context.Background()
	sm := &stateManager{Queue: inmemory.NewStateManager()}
	r := newRunner(t, sm, nil)

	f := inngest.Workflow{}
	data := map[string]interface{}{
		"yea": "yea",
	}
	id, err := r.NewRun(ctx, f, data)
	require.NoError(t, err)
	require.NotNil(t, id)

	// TODO

	// Ensure that the ID and data exists within our state store.
	state, err := sm.Load(ctx, *id)
	require.NoError(t, err)
	require.NotNil(t, state)
	evt := state.Event()
	require.EqualValues(t, evt, data)

	// Ensure that we've enqueued a run to start from the source edge
	// of the dag.
	item := <-sm.Channel()
	require.NotNil(t, item)
	require.EqualValues(t, inmemory.QueueItem{
		ID:   *id,
		Edge: inngest.SourceEdge,
	}, item)
}

func TestRunner_run_source(t *testing.T) {
	driver := &mockdriver.Mock{
		Responses: map[string]driver.Response{
			"first": {Output: map[string]interface{}{"ok": true}},
		},
	}

	ctx := context.Background()
	sm := &stateManager{Queue: inmemory.NewStateManager()}
	r := newRunner(t, sm, driver)

	f := inngest.Workflow{
		Steps: []inngest.Step{
			{
				ClientID: "first",
				DSN:      "step-a",
			},
		},
		Edges: []inngest.Edge{
			{
				Outgoing: inngest.TriggerName,
				Incoming: "first",
			},
		},
	}
	data := map[string]interface{}{
		"yea": "yea",
	}
	id, err := r.NewRun(ctx, f, data)
	require.NoError(t, err)

	// Run the trigger.
	item := <-sm.Channel()
	err = r.run(ctx, item)
	require.NoError(t, err)

	// This should have done nothing but enqueue new items.
	item = <-sm.Channel()
	require.EqualValues(t, inmemory.QueueItem{
		ID: *id,
		Edge: inngest.Edge{
			Outgoing: inngest.TriggerName,
			Incoming: "first",
		},
	}, item)

	// Run this item.
	err = r.run(ctx, item)
	require.NoError(t, err)

	// There should be nothing else left enqueued.
	select {
	case <-time.After(time.Millisecond):
	case <-sm.Channel():
		t.Fail()
	}
}

func TestRunner_run_retry(t *testing.T) {
	driver := &mockdriver.Mock{
		Responses: map[string]driver.Response{
			"first": {Err: fmt.Errorf("some error here")},
		},
	}

	ctx := context.Background()
	sm := &stateManager{Queue: inmemory.NewStateManager()}
	r := newRunner(t, sm, driver)

	f := inngest.Workflow{
		Steps: []inngest.Step{
			{
				ClientID: "first",
				DSN:      "step-a",
			},
		},
		Edges: []inngest.Edge{
			{
				Outgoing: inngest.TriggerName,
				Incoming: "first",
			},
		},
	}
	data := map[string]interface{}{
		"yea": "yea",
	}
	id, err := r.NewRun(ctx, f, data)
	require.NoError(t, err)

	// When making a new run, there's always a trigger in the queue.
	require.Equal(t, 1, len(sm.queue))
	AssertLastEnqueued(t, sm, inmemory.QueueItem{
		ID:   *id,
		Edge: inngest.SourceEdge,
	}, time.Now())

	// Run the trigger.
	item := <-sm.Channel()
	err = r.run(ctx, item)
	require.NoError(t, err)

	// Assert that the first step is in the queue.
	require.Equal(t, 2, len(sm.queue))
	AssertLastEnqueued(t, sm, inmemory.QueueItem{
		ID: *id,
		Edge: inngest.Edge{
			Outgoing: inngest.TriggerName,
			Incoming: "first",
		},
	}, time.Now())

	// This should have done nothing but enqueue new items.
	item = <-sm.Channel()
	require.EqualValues(t, inmemory.QueueItem{
		ID: *id,
		Edge: inngest.Edge{
			Outgoing: inngest.TriggerName,
			Incoming: "first",
		},
	}, item)

	// Run this item.
	err = r.run(ctx, item)
	require.Error(t, err, driver.Responses["first"].Err.Error())

	// Assert that the item was re-enqueued correctly.
	require.Equal(t, 3, len(sm.queue))
	AssertLastEnqueued(t, sm, inmemory.QueueItem{
		ID:         *id,
		ErrorCount: 1,
		Edge: inngest.Edge{
			Outgoing: inngest.TriggerName,
			Incoming: "first",
		},
	}, time.Now().Add(10*time.Second))
}

func AssertLastEnqueued(t *testing.T, sm *stateManager, i inmemory.QueueItem, at time.Time) {
	n := len(sm.queue) - 1
	require.EqualValues(t, sm.queue[n].item, i)
	// And that it should be ran immediately.
	require.WithinDuration(t, at, sm.queue[n].at, time.Second)
}
