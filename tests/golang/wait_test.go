package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/testapi"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

func TestWait(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWait" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will invoke the other function
	runID := ""
	evtName := "wait-event"
	waitEvtName := "resume"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					Name:    "dummy",
					Event:   waitEvtName,
					Timeout: 30 * time.Second,
				},
			)

			return "DONE", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	// Trigger the main function
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	t.Run("in progress wait", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusRunning,
			ChildSpanCount: 1,
		})

		require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
		require.Nil(t, run.Trace.OutputID)

		rootSpanID := run.Trace.SpanID

		t.Run("wait step", func(t *testing.T) {
			span := run.Trace.ChildSpans[0]
			assert.Equal(t, "dummy", span.Name)
			assert.Equal(t, 0, span.Attempts)
			assert.Equal(t, rootSpanID, span.ParentSpanID)
			assert.False(t, span.IsRoot)
			assert.Equal(t, 0, len(span.ChildSpans)) // NOTE: should have no child
			assert.Equal(t, models.RunTraceSpanStatusWaiting.String(), span.Status)
			assert.Equal(t, models.StepOpWaitForEvent.String(), span.StepOp)
			assert.Nil(t, span.EndedAt)
			assert.Nil(t, span.OutputID)

			var stepInfo models.WaitForEventStepInfo
			byt, err := json.Marshal(span.StepInfo)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(byt, &stepInfo))

			assert.Equal(t, waitEvtName, stepInfo.EventName)
			assert.Nil(t, stepInfo.TimedOut)
			assert.Nil(t, stepInfo.FoundEventID)
		})
	})

	<-time.After(10 * time.Second)
	// Trigger the main function
	_, err = inngestClient.Send(ctx, &event.Event{Name: waitEvtName})
	r.NoError(err)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusCompleted,
			ChildSpanCount: 1,
		})

		require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

		// output test
		require.NotNil(t, run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanOutput(t, "DONE", output)

		rootSpanID := run.Trace.SpanID

		t.Run("wait step", func(t *testing.T) {
			span := run.Trace.ChildSpans[0]
			assert.Equal(t, "dummy", span.Name)
			assert.Equal(t, 0, span.Attempts)
			assert.Equal(t, rootSpanID, span.ParentSpanID)
			assert.False(t, span.IsRoot)
			assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), span.Status)
			assert.Equal(t, models.StepOpWaitForEvent.String(), span.StepOp)

			// output test
			assert.NotNil(t, span.OutputID)
			spanOutput := c.RunSpanOutput(ctx, *span.OutputID)
			c.ExpectSpanOutput(t, "resume", spanOutput)

			var stepInfo models.WaitForEventStepInfo
			byt, err := json.Marshal(span.StepInfo)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(byt, &stepInfo))

			assert.Equal(t, waitEvtName, stepInfo.EventName)
			assert.NotNil(t, stepInfo.TimedOut)
			assert.False(t, *stepInfo.TimedOut)
			assert.NotNil(t, stepInfo.FoundEventID)
			assert.Nil(t, stepInfo.Expression)
		})
	})
}

func TestWaitGroup(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWaitGroup" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	var started int32

	runID := ""
	evtName := "wait-group"
	waitEvtName := "resume-group"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			if atomic.LoadInt32(&started) == 0 {
				atomic.AddInt32(&started, 1)
				return nil, inngestgo.RetryAtError(fmt.Errorf("initial error"), time.Now().Add(5*time.Second))
			}

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					Name:    "dummy",
					Event:   waitEvtName,
					Timeout: 30 * time.Second,
				},
			)

			return "DONE", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	// Trigger the main function
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	t.Run("in progress wait", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusRunning,
			ChildSpanCount: 1,
		})

		require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
		require.Nil(t, run.Trace.OutputID)

		rootSpanID := run.Trace.SpanID

		span := run.Trace.ChildSpans[0]
		assert.Equal(t, consts.OtelExecPlaceholder, span.Name)
		assert.Equal(t, 0, span.Attempts)
		assert.Equal(t, rootSpanID, span.ParentSpanID)
		assert.False(t, span.IsRoot)
		assert.Equal(t, 2, len(span.ChildSpans)) // include queued retry span
		assert.Equal(t, models.RunTraceSpanStatusRunning.String(), span.Status)
		assert.Equal(t, "", span.StepOp)
		assert.Nil(t, span.OutputID)

		t.Run("failed", func(t *testing.T) {
			exec := span.ChildSpans[0]
			assert.Equal(t, "Attempt 0", exec.Name)
			assert.Equal(t, models.RunTraceSpanStatusFailed.String(), exec.Status)
			assert.NotNil(t, exec.OutputID)

			execOutput := c.RunSpanOutput(ctx, *exec.OutputID)
			assert.NotNil(t, execOutput)
			c.ExpectSpanErrorOutput(t, "initial error", "", execOutput)
		})

		// Wait for the WaitForEvent to appear in history
		r.EventuallyWithT(func(ct *assert.CollectT) {
			a := assert.New(ct)
			run, err := c.RunTraces(ctx, runID)
			a.NoError(err)
			a.Len(run.Trace.ChildSpans, 1)

			isWaiting := false
			for _, s := range run.Trace.ChildSpans[0].ChildSpans {
				if s.StepOp == models.StepOpWaitForEvent.String() {
					isWaiting = true
					break
				}
			}
			a.True(isWaiting)
		}, 5*time.Second, 500*time.Millisecond)
	})

	// Trigger the main function
	_, err = inngestClient.Send(ctx, &event.Event{Name: waitEvtName})
	r.NoError(err)

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusCompleted,
			ChildSpanCount: 1,
		})

		require.Equal(t, models.RunTraceSpanStatusCompleted.String(), run.Trace.Status)

		// output test
		require.NotNil(t, run.Trace.OutputID)
		output := c.RunSpanOutput(ctx, *run.Trace.OutputID)
		c.ExpectSpanOutput(t, "DONE", output)

		rootSpanID := run.Trace.SpanID

		t.Run("wait step", func(t *testing.T) {
			span := run.Trace.ChildSpans[0]
			assert.Equal(t, "dummy", span.Name)
			assert.Equal(t, 0, span.Attempts)
			assert.Equal(t, rootSpanID, span.ParentSpanID)
			assert.False(t, span.IsRoot)
			assert.Equal(t, 2, len(span.ChildSpans))
			assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), span.Status)
			assert.Equal(t, models.StepOpWaitForEvent.String(), span.StepOp)

			// output test
			assert.NotNil(t, span.OutputID)
			spanOutput := c.RunSpanOutput(ctx, *span.OutputID)
			c.ExpectSpanOutput(t, waitEvtName, spanOutput)

			var stepInfo models.WaitForEventStepInfo
			byt, err := json.Marshal(span.StepInfo)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(byt, &stepInfo))

			assert.Equal(t, waitEvtName, stepInfo.EventName)
			assert.NotNil(t, stepInfo.TimedOut)
			assert.False(t, *stepInfo.TimedOut)
			assert.NotNil(t, stepInfo.FoundEventID)
			assert.Nil(t, stepInfo.Expression)
		})
	})
}

func TestWaitInvalidExpression(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWaitInvalidExpression" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					If:      inngestgo.StrPtr("invalid"),
					Name:    "dummy",
					Timeout: 30 * time.Second,
				},
			)

			return nil, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "FAILED", &runID)
}

func TestWaitInvalidExpressionSyntaxError(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestWaitInvalidExpression" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will invoke the other function
	runID := ""
	evtName := "my-event"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					If:      inngestgo.StrPtr("event.data.userId === async.data.userId"),
					Name:    "test/continue",
					Timeout: 30 * time.Second,
				},
			)

			return nil, nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	run := c.WaitForRunStatus(ctx, t, "FAILED", &runID)
	assert.Equal(t,
		`{"error":{"error":"InvalidExpression: Wait for event If expression failed to compile","name":"InvalidExpression","message":"Wait for event If expression failed to compile","stack":"error compiling expression: ERROR: \u003cinput\u003e:1:21: Syntax error: token recognition error at: '= '\n | event.data.userId === async.data.userId\n | ....................^"}}`,
		run.Output,
	)
}

func TestManyWaitInvalidExpressions(t *testing.T) {
	// This test ensures that valid expressions can match even when there are
	// many invalid expressions.
	//
	// We created this test because aggregate pause processing did not support
	// partial failures: if any expression was invalid then the valid
	// expressions were skipped.

	ctx := context.Background()
	r := require.New(t)

	appID := ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	type eventData struct {
		Bad bool `json:"bad"`
	}

	var counter int32
	var done bool
	evtName := "my-event"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(
			ctx context.Context,
			input inngestgo.Input[eventData],
		) (any, error) {
			atomic.AddInt32(&counter, 1)

			exp := "async.data.name == 'Alice'"
			if input.Event.Data.Bad {
				exp = "invalid"
			}

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					If:      inngestgo.StrPtr(exp),
					Name:    "match-event",
					Timeout: time.Minute,
				},
			)

			done = true
			return nil, nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Trigger enough function runs to cause us to use the "aggregate pauses"
	// code path.
	var badEvents []any
	for i := 0; i < consts.AggregatePauseThreshold+1; i++ {
		badEvents = append(badEvents, event.Event{
			Data: map[string]any{"bad": true},
			Name: evtName,
		})
	}
	_, err = inngestClient.SendMany(ctx, badEvents)
	r.NoError(err)
	r.EventuallyWithT(func(ct *assert.CollectT) {
		a := assert.New(ct)
		a.EqualValues(len(badEvents), atomic.LoadInt32(&counter))
	}, 20*time.Second, 100*time.Millisecond)

	// Trigger a function run with a valid expression that should match.
	_, err = inngestClient.Send(ctx, &event.Event{
		Data: map[string]any{"bad": false},
		Name: evtName,
	})
	r.NoError(err)
	r.EventuallyWithT(func(ct *assert.CollectT) {
		a := assert.New(ct)
		a.EqualValues(len(badEvents)+1, atomic.LoadInt32(&counter))
	}, 10*time.Second, 100*time.Millisecond)

	// Arbitrary sleep to ensure all the waitForEvents are processed.
	<-time.After(time.Second)

	// Send an event that should match the valid expression.
	_, err = inngestClient.Send(ctx, &event.Event{
		Data: map[string]any{"name": "Alice"},
		Name: "match-event",
	})
	r.NoError(err)

	// Ensure we made it past the waitForEvent in the valid expression function
	// run.
	r.EventuallyWithT(func(ct *assert.CollectT) {
		a := assert.New(ct)
		a.True(done)
	}, 5*time.Second, 100*time.Millisecond)
}

func TestWaitForEvent_Timeout(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	c.ResetAll(t)

	appID := "TestWaitTimeout" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	// This function will invoke the other function
	runID := ""
	evtName := "wait-event-timeout"
	waitEvtName := "resume-but-dont-actually-cuz-lol-timeouts"
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "main-fn",
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID

			_, _ = step.WaitForEvent[any](
				ctx,
				"wait",
				step.WaitForEventOpts{
					Name:    "dummy",
					Event:   waitEvtName,
					Timeout: 5 * time.Second,
				},
			)

			return "DONE", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	fnId := ""
	require.Eventually(t, func() bool {
		functions, err := c.Functions(ctx)
		if err != nil {
			return false
		}
		for _, function := range functions {
			if function.App.ExternalID != appID {
				continue
			}

			fnId = function.ID
			return true
		}
		return false
	}, 10*time.Second, 250*time.Millisecond)

	getActiveCount := func(accountId uuid.UUID, fnId uuid.UUID) testapi.TestActiveSets {
		reqUrl, err := url.Parse(c.APIHost + "/test/queue/active-count")
		require.NoError(t, err)

		fv := reqUrl.Query()
		fv.Add("accountId", consts.DevServerAccountID.String())
		fv.Add("fnId", fnId.String())

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl.String()+"?"+fv.Encode(), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		byt, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		r := testapi.TestActiveSets{}
		err = json.Unmarshal(byt, &r)
		require.NoError(t, err, "Test API may not be enabled! Error unmarshalling %s", byt)

		return r
	}

	// Trigger the main function
	_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)

	run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
		Status:         models.FunctionStatusRunning,
		ChildSpanCount: 1,
	})

	require.Equal(t, models.RunTraceSpanStatusRunning.String(), run.Trace.Status)
	require.Nil(t, run.Trace.OutputID)

	rootSpanID := run.Trace.SpanID

	time.Sleep(5 * time.Second)

	t.Run("on timeout", func(t *testing.T) {
		span := run.Trace.ChildSpans[0]
		assert.Equal(t, "dummy", span.Name)
		assert.Equal(t, 0, span.Attempts)
		assert.Equal(t, rootSpanID, span.ParentSpanID)
		assert.False(t, span.IsRoot)
		assert.Equal(t, 0, len(span.ChildSpans)) // NOTE: should have no child
		assert.Equal(t, models.RunTraceSpanStatusWaiting.String(), span.Status)
		assert.Equal(t, models.StepOpWaitForEvent.String(), span.StepOp)
		assert.Nil(t, span.EndedAt)
		assert.Nil(t, span.OutputID)

		var stepInfo models.WaitForEventStepInfo
		byt, err := json.Marshal(span.StepInfo)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(byt, &stepInfo))

		assert.Equal(t, waitEvtName, stepInfo.EventName)
		assert.Nil(t, stepInfo.TimedOut)
		assert.Nil(t, stepInfo.FoundEventID)

		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			count := getActiveCount(consts.DevServerAccountID, uuid.MustParse(fnId))
			assert.Equal(collect, testapi.TestActiveSets{
				ActiveAccount:      0,
				ActiveFunction:     0,
				ActiveRunsAccount:  0,
				ActiveRunsFunction: 0,
			}, count)
		}, 15*time.Second, 25*time.Millisecond)
	})
}
