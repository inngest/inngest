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
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
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
				return "hello 1", nil
			})
			require.NoError(t, err)

			_, err = step.Run(ctx, "2", func(ctx context.Context) (string, error) {
				fmt.Println("2")
				atomic.AddInt32(&counter, 1)
				return "test", nil
			})
			require.NoError(t, err)

			step.Sleep(ctx, "delay", 2*time.Second)

			_, err = step.WaitForEvent[any](ctx, "wait1", step.WaitForEventOpts{
				Event:   "api/new.event",
				Timeout: time.Minute,
			})
			if err == step.ErrEventNotReceived {
				panic("no event found")
			}

			// Wait for an event with an expression
			_, err = step.WaitForEvent[any](ctx, "wait2", step.WaitForEventOpts{
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
		run := c.WaitForRunTraces(ctx, t, &runID, models.FunctionStatusCompleted)

		require.False(t, run.IsBatch)
		require.Nil(t, run.BatchCreatedAt)

		require.NotNil(t, run.Trace)
		require.True(t, run.Trace.IsRoot)
		require.Equal(t, 5, len(run.Trace.ChildSpans))
		require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

		// output test
		require.NotNil(t, run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		require.NotNil(t, output)
		require.NotNil(t, output.Data)
		require.Contains(t, *output.Data, "true")

		rootSpanID := run.Trace.SpanID

		t.Run("step 1", func(t *testing.T) {
			one := run.Trace.ChildSpans[0]
			assert.Equal(t, "1", one.Name)
			assert.Equal(t, 0, one.Attempts)
			assert.False(t, one.IsRoot)
			assert.Equal(t, rootSpanID, one.ParentSpanID)
			assert.Equal(t, models.StepOpRun.String(), one.StepOp)
			assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), one.Status)
			// output test
			assert.NotNil(t, one.OutputID)
			oneOutput := c.RunSpanOutput(ctx, *one.OutputID)
			c.ExpectSpanOutput(t, "hello 1", oneOutput)
		})

		t.Run("step 2", func(t *testing.T) {
			sec := run.Trace.ChildSpans[1]
			assert.Equal(t, "2", sec.Name)
			assert.Equal(t, 0, sec.Attempts)
			assert.False(t, sec.IsRoot)
			assert.Equal(t, rootSpanID, sec.ParentSpanID)
			assert.Equal(t, models.StepOpRun.String(), sec.StepOp)
			assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), sec.Status)
			// output test
			assert.NotNil(t, sec.OutputID)
			secOutput := c.RunSpanOutput(ctx, *sec.OutputID)
			c.ExpectSpanOutput(t, "test", secOutput)
		})

		// third step
		t.Run("step sleep", func(t *testing.T) {
			thr := run.Trace.ChildSpans[2]
			assert.Equal(t, "delay", thr.Name)
			assert.Equal(t, 0, thr.Attempts)
			assert.False(t, thr.IsRoot)
			assert.Equal(t, rootSpanID, thr.ParentSpanID)
			assert.Equal(t, models.StepOpSleep.String(), thr.StepOp)
			assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), thr.Status)
			assert.NotNil(t, thr.StartedAt)
			assert.NotNil(t, thr.EndedAt)
			assert.Nil(t, thr.OutputID)
			// check sleep duration
			expectedDur := (2 * time.Second).Milliseconds()
			assert.InDelta(t, expectedDur, thr.Duration, 200)
		})

		// forth
		t.Run("wait step", func(t *testing.T) {
			forth := run.Trace.ChildSpans[3]
			assert.Equal(t, "wait1", forth.Name)
			assert.Equal(t, 0, forth.Attempts)
			assert.False(t, forth.IsRoot)
			assert.Equal(t, rootSpanID, forth.ParentSpanID)
			assert.Equal(t, models.StepOpWaitForEvent.String(), forth.StepOp)
			assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), forth.Status)
			assert.NotNil(t, forth.StartedAt)
			assert.NotNil(t, forth.EndedAt)

			var stepInfo models.WaitForEventStepInfo
			byt, err := json.Marshal(forth.StepInfo)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(byt, &stepInfo))

			assert.False(t, *stepInfo.TimedOut)
			assert.NotNil(t, stepInfo.FoundEventID)

			// output test
			assert.NotNil(t, forth.OutputID)
			forthOutput := c.RunSpanOutput(ctx, *forth.OutputID)
			c.ExpectSpanOutput(t, "api/new.event", forthOutput)
		})

		t.Run("trigger", func(t *testing.T) {
			// check trigger
			trigger := c.RunTrigger(ctx, runID)
			assert.NotNil(t, trigger)
			assert.NotNil(t, trigger.EventName)
			assert.Equal(t, "test/sdk-steps", *trigger.EventName)
			assert.Equal(t, 1, len(trigger.IDs))
			assert.False(t, trigger.Timestamp.IsZero())
			assert.False(t, trigger.IsBatch)
			assert.Nil(t, trigger.BatchID)
			assert.Nil(t, trigger.Cron)

			rid := ulid.MustParse(runID)
			assert.True(t, trigger.Timestamp.Before(ulid.Time(rid.Time())))
		})
	})
}
