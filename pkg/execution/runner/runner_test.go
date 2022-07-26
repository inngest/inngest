package runner

/*
import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coredata"
	"github.com/inngest/inngest/pkg/execution/driver/mockdriver"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/inmemory"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/service"
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
	item queue.Item
	at   time.Time
}

func (m *stateManager) Enqueue(ctx context.Context, item queue.Item, at time.Time) error {
	m.queue = append(m.queue, enqueued{item: item, at: at})
	return m.Queue.Enqueue(ctx, item, at)
}

func newRunner(t *testing.T, ctx context.Context, conf *config.Config, f function.Function) {
	t.Helper()

	el := &coredata.MemoryExecutionLoader{}
	err := el.SetFunctions(ctx, []*function.Function{&f})
	require.NoError(t, err)

	go func() {
		// Start the engine.
		api := api.NewService(*conf)
		runner := NewService(*conf, WithExecutionLoader(el))
		exec := executor.NewService(*conf, executor.WithExecutionLoader(el))
		err = service.StartAll(ctx, api, runner, exec)
		require.NoError(t, err)
	}()
}

func TestRunner_new(t *testing.T) {
	ctx := context.Background()
	conf, err := config.Default(ctx)
	require.NoError(t, err)

	newRunner(t, ctx, conf, function.Function{
	})

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
	require.EqualValues(t, queue.Item{
		Identifier: *id,
		Payload: queue.PayloadEdge{
			Edge: inngest.SourceEdge,
		},
	}, item)
}

func TestRunner_run_source(t *testing.T) {
	driver := &mockdriver.Mock{
		Responses: map[string]state.DriverResponse{
			"first": {Output: map[string]interface{}{"ok": true}},
		},
	}

	ctx := context.Background()
	sm := &stateManager{Queue: inmemory.NewStateManager()}
	r := newRunner(t, sm, driver)

	f := inngest.Workflow{
		Steps: []inngest.Step{
			{
				ID:  "first",
				DSN: "step-a",
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
	require.EqualValues(t, queue.Item{
		Identifier: *id,
		Payload: queue.PayloadEdge{
			Edge: inngest.Edge{
				Outgoing: inngest.TriggerName,
				Incoming: "first",
			},
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
		Responses: map[string]state.DriverResponse{
			"first": {Err: fmt.Errorf("some error here")},
		},
	}

	ctx := context.Background()
	sm := &stateManager{Queue: inmemory.NewStateManager()}
	r := newRunner(t, sm, driver)

	f := inngest.Workflow{
		Steps: []inngest.Step{
			{
				ID:  "first",
				DSN: "step-a",
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
	AssertLastEnqueued(t, sm, queue.Item{
		Identifier: *id,
		Payload: queue.PayloadEdge{
			Edge: inngest.SourceEdge,
		},
	}, time.Now())

	// Run the trigger.
	item := <-sm.Channel()
	err = r.run(ctx, item)
	require.NoError(t, err)

	// Assert that the first step is in the queue.
	require.Equal(t, 2, len(sm.queue))
	AssertLastEnqueued(t, sm, queue.Item{
		Identifier: *id,
		Payload: queue.PayloadEdge{
			Edge: inngest.Edge{
				Outgoing: inngest.TriggerName,
				Incoming: "first",
			},
		},
	}, time.Now())

	// This should have done nothing but enqueue new items.
	item = <-sm.Channel()
	require.EqualValues(t, queue.Item{
		Identifier: *id,
		Payload: queue.PayloadEdge{
			Edge: inngest.Edge{
				Outgoing: inngest.TriggerName,
				Incoming: "first",
			},
		},
	}, item)

	// Run this item.
	err = r.run(ctx, item)
	require.Error(t, err, driver.Responses["first"].Err.Error())

	// Assert that the item was re-enqueued correctly.
	require.Equal(t, 3, len(sm.queue))
	AssertLastEnqueued(t, sm, queue.Item{
		Identifier: *id,
		ErrorCount: 1,
		Payload: queue.PayloadEdge{
			Edge: inngest.Edge{
				Outgoing: inngest.TriggerName,
				Incoming: "first",
			},
		},
	}, time.Now().Add(10*time.Second))
}

// This test asserts that the scheduled and finalized counts are increased
// correctly throughout the runner.
func TestRunStateModification(t *testing.T) {
	f := inngest.Workflow{
		Steps: []inngest.Step{
			{
				ID:   "a",
				Name: "a",
				DSN:  "test",
			},
			{
				ID:   "b",
				Name: "b",
				DSN:  "test",
			},
			{
				ID:   "a-run",
				Name: "a-run",
				DSN:  "test",
			},
			{
				ID:   "a-ignore",
				Name: "a-ignore",
				DSN:  "test",
			},
			{
				ID:   "b-a",
				Name: "b-a",
				DSN:  "test",
			},
		},
		Edges: []inngest.Edge{
			{
				Outgoing: inngest.TriggerName,
				Incoming: "a",
			},
			{
				Outgoing: inngest.TriggerName,
				Incoming: "b",
			},
			{
				Outgoing: "a",
				Incoming: "a-run",
				Metadata: &inngest.EdgeMetadata{
					If: "true == true",
				},
			},
			{
				Outgoing: "a",
				Incoming: "a-ignore",
				Metadata: &inngest.EdgeMetadata{
					If: "true == false",
				},
			},
			{
				Outgoing: "b",
				Incoming: "b-a",
			},
		},
	}

	driver := &mockdriver.Mock{
		Responses: map[string]state.DriverResponse{
			"a":        {Output: map[string]interface{}{"status": 200, "body": "a"}},
			"b":        {Output: map[string]interface{}{"status": 200, "body": "b"}},
			"b-run":    {Output: map[string]interface{}{"status": 200, "body": "b-run"}},
			"a-run":    {Output: map[string]interface{}{"status": 200, "body": "a-run"}},
			"b-a":      {Output: map[string]interface{}{"status": 200, "body": "b-a"}},
			"a-ignore": {Err: fmt.Errorf("this should never run")},
		},
	}

	ctx := context.Background()
	sm := &stateManager{Queue: inmemory.NewStateManager()}
	r := newRunner(t, sm, driver)

	id, err := r.NewRun(ctx, f, map[string]interface{}{"input": true})
	require.NoError(t, err)
	require.NotNil(t, id)

	s, err := r.sm.Load(ctx, *id)
	require.NoError(t, err)
	require.NotNil(t, s)

	// We should have one pending: the trigger.
	require.Equal(t, 1, s.Metadata().Pending)

	// Run the queue.
	go func() {
		err := r.Start(ctx)
		require.NoError(t, err)
	}()

	err = r.Wait(ctx, s.Identifier())
	require.NoError(t, err)

	s, err = r.sm.Load(ctx, *id)
	require.NoError(t, err)
	require.NotNil(t, s)
	require.Equal(t, 0, s.Metadata().Pending)
}

func AssertLastEnqueued(t *testing.T, sm *stateManager, i queue.Item, at time.Time) {
	n := len(sm.queue) - 1
	require.EqualValues(t, sm.queue[n].item, i)
	// And that it should be ran immediately.
	require.WithinDuration(t, at, sm.queue[n].at, time.Second)
}
*/
