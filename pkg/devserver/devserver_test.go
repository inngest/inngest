package devserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/config/registration"
	"github.com/inngest/inngest-cli/pkg/coredata"
	"github.com/inngest/inngest-cli/pkg/event"
	"github.com/inngest/inngest-cli/pkg/execution/driver/mockdriver"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/inngest/inngest-cli/pkg/function"
	"github.com/stretchr/testify/require"
)

// TestEngine_async asserst that the engine coordinates events between the runner, executor, and
// state manager to successfully pause workflows until specific events are received.
func TestEngine_async(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &mockdriver.Config{
		Responses: map[string]state.DriverResponse{
			"first":        {Output: map[string]interface{}{"ok": true}},
			"wait-for-evt": {Output: map[string]interface{}{"ok": true}},
		},
	}

	conf, err := config.Default(ctx)
	require.NoError(t, err)
	// Update config to use our mocking driver.
	conf.Execution.Drivers = map[string]registration.DriverConfig{
		"mock": mock,
	}
	conf.EventAPI.Port = "47192"

	// Fetch the in-memory state store singleton.
	sm := inmemory.NewSingletonStateManager()

	el := &coredata.MemoryExecutionLoader{}
	err = el.SetFunctions(ctx, []*function.Function{
		{
			Name: "test fn",
			ID:   "test-fn",
			Triggers: []function.Trigger{
				{
					EventTrigger: &function.EventTrigger{
						Event: "test/new.event",
					},
				},
			},
			Steps: map[string]function.Step{
				"first": {
					ID:   "first",
					Name: "Basic step",
					Runtime: inngest.RuntimeWrapper{
						Runtime: &mockdriver.Mock{},
					},
					After: []function.After{
						{
							Step: inngest.TriggerName,
						},
					},
				},
				"wait-for-evt": {
					ID:   "wait-for-evt",
					Name: "A step with a wait",
					Runtime: inngest.RuntimeWrapper{
						Runtime: &mockdriver.Mock{},
					},
					After: []function.After{
						{
							Step: inngest.TriggerName,
							Async: &inngest.AsyncEdgeMetadata{
								Event: "test/continue",
								TTL:   "5s",
								Match: strptr("async.data.continue == 'yes'"),
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	go func() {
		// Start the engine.
		err = newDevServer(ctx, *conf, el)
		require.NoError(t, err)
	}()

	handleEvent := func(ctx context.Context, evt *event.Event) error {
		byt, err := json.Marshal(evt)
		require.NoError(t, err)
		buf := bytes.NewBuffer(byt)
		resp, err := http.Post(
			fmt.Sprintf("http://127.0.0.1:%s/e/key", conf.EventAPI.Port),
			"application/json",
			buf,
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, 200, resp.StatusCode)
		return err
	}

	<-time.After(2 * time.Second)

	// Fetch the driver that the mock driver created.
	d, err := mock.NewDriver()
	require.NoError(t, err)
	driver := d.(*mockdriver.Mock)

	// 1.
	// Send an event that does nothing, and assert nothing runs.
	err = handleEvent(ctx, &event.Event{
		Name: "test/random.walk.down.the.street",
		Data: map[string]interface{}{
			"test": true,
		},
	})
	require.NoError(t, err)
	<-time.After(50 * time.Millisecond)
	require.EqualValues(t, 0, len(driver.Executed))

	// 2.
	// HandleEvent should create a new execution when an event matches
	// the trigger.
	err = handleEvent(ctx, &event.Event{
		Name: "test/new.event",
		Data: map[string]interface{}{
			"test": true,
		},
	})
	require.NoError(t, err)

	// Eventually the first step should execute.
	require.Eventually(t, func() bool {
		return driver.ExecutedLen() == 1
	}, time.Second, 10*time.Millisecond)
	// Assert that the first step ran.
	require.Equal(t, "Basic step", driver.Executed["first"].Name)

	// And we should have a pause.
	require.Eventually(t, func() bool {
		n := 0
		iter, err := sm.PausesByEvent(ctx, "test/continue")
		require.NoError(t, err)
		for iter.Next(ctx) {
			n++
		}
		return n == 1
	}, 50*time.Millisecond, 10*time.Millisecond)

	// 3.
	// Once we have the pause, we can send another event.  This shouldn't continue
	// the stopped function as the expression doesn't match.
	err = handleEvent(ctx, &event.Event{
		Name: "test/continue",
		Data: map[string]interface{}{
			"continue": "no",
		},
	})
	require.NoError(t, err)
	<-time.After(50 * time.Millisecond)
	require.EqualValues(t, 1, len(driver.Executed))
	require.Eventually(t, func() bool {
		n := 0
		iter, err := sm.PausesByEvent(ctx, "test/continue")
		require.NoError(t, err)
		for iter.Next(ctx) {
			n++
		}
		return n == 1
	}, 50*time.Millisecond, 10*time.Millisecond)

	// 4.
	// Finally, assert that sending an event which matches the pause conditions
	// starts the workflow from the stopped edge.
	err = handleEvent(ctx, &event.Event{
		Name: "test/continue",
		Data: map[string]interface{}{
			"continue": "yes",
		},
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return driver.ExecutedLen() == 2
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, "A step with a wait", driver.Executed["wait-for-evt"].Name)
}

func strptr(s string) *string {
	return &s
}
