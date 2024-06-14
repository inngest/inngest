package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

func TestFunctionSteps(t *testing.T) {
	ctx := context.Background()

	c := client.New(t)
	h, server, registerFuncs := NewSDKHandler(t, "my-app")
	defer server.Close()

	var (
		counter int32
		runID   string
	)

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "test sdk"},
		inngestgo.EventTrigger("test/sdk-steps", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			runID = input.InputCtx.RunID

			_, err := step.Run(ctx, "1", func(ctx context.Context) (any, error) {
				fmt.Println("1")
				atomic.AddInt32(&counter, 1)
				return input.Event, nil
			})
			require.NoError(t, err)

			_, err = step.Run(ctx, "2", func(ctx context.Context) (string, error) {
				fmt.Println("2")
				atomic.AddInt32(&counter, 1)
				return "test", nil
			})
			require.NoError(t, err)

			step.Sleep(ctx, "delay", 2*time.Second)

			_, err = step.WaitForEvent[any](ctx, "wait", step.WaitForEventOpts{
				Name:    "step name",
				Event:   "api/new.event",
				Timeout: time.Minute,
			})
			if err == step.ErrEventNotReceived {
				panic("no event found")
			}

			// Wait for an event with an expression
			_, err = step.WaitForEvent[any](ctx, "wait", step.WaitForEventOpts{
				Name:    "step name",
				Event:   "api/new.event",
				If:      inngestgo.StrPtr(`async.data.ok == "yes" && async.data.id == event.data.id`),
				Timeout: time.Minute,
			})
			if err == step.ErrEventNotReceived {
				panic("no event found")
			}

			fmt.Println("3")
			atomic.AddInt32(&counter, 1)
			return true, nil
		},
	)
	h.Register(a)
	registerFuncs()

	evt := inngestgo.Event{
		Name: "test/sdk-steps",
		Data: map[string]any{
			"test": true,
			"id":   "1",
		},
	}
	_, err := inngestgo.Send(ctx, evt)
	require.NoError(t, err)

	<-time.After(3 * time.Second)

	// While we're waiting, ensure that the batch API works.  We must do this while the function is
	// in-progress as the state is cleared up.
	t.Run("Check batch API", func(t *testing.T) {
		// Fetch event data and step data from the V0 APIs;  it should exist.
		resp, err := http.Get(fmt.Sprintf("%s/v0/runs/%s/batch", DEV_URL, runID))
		require.NoError(t, err)
		require.EqualValues(t, 200, resp.StatusCode)

		body := []event.Event{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		_ = resp.Body.Close()
		require.NoError(t, err)

		require.Equal(t, 1, len(body))
		require.EqualValues(t, evt.Data, body[0].Data)
	})

	t.Run("Check batch API", func(t *testing.T) {
		// Fetch event data and step data from the V0 APIs;  it should exist.
		resp, err := http.Get(fmt.Sprintf("%s/v0/runs/%s/actions", DEV_URL, runID))
		require.NoError(t, err)
		require.EqualValues(t, 200, resp.StatusCode)

		body := map[string]any{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		_ = resp.Body.Close()
		require.NoError(t, err)

		// 3 step so far: 2 steps, 1 wait
		require.Equal(t, 3, len(body))
	})

	t.Run("waitForEvents succeed", func(t *testing.T) {
		// Send the first event to trigger the wait.
		_, err = inngestgo.Send(ctx, inngestgo.Event{
			Name: "api/new.event",
			Data: map[string]any{
				"test": true,
			},
		})
		require.NoError(t, err)

		<-time.After(time.Second)

		// And the second event to trigger the next wait.
		_, err = inngestgo.Send(ctx, inngestgo.Event{
			Name: "api/new.event",
			Data: map[string]any{
				"ok": "yes",
				"id": "1",
			},
		})
		require.NoError(t, err)

		<-time.After(time.Second)

		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&counter) == 3
		}, 15*time.Second, time.Second)
	})

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		<-time.After(3 * time.Second)

		require.Eventually(t, func() bool {
			run := c.RunTraces(ctx, runID)
			require.NotNil(t, run)
			require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Status)
			require.False(t, run.IsBatch)
			require.Nil(t, run.BatchCreatedAt)

			// TODO: add traces

			return true
		}, 10*time.Second, 2*time.Second)
	})
}
