package sdkloop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/loop"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
	str2duration "github.com/xhit/go-str2duration/v2"
)

// eventLoop is the loop block for a run started by an event.
func eventLoop() *driver.SDKLoopRequest {
	return &driver.SDKLoopRequest{
		State: json.RawMessage(`{"count":0,"last":""}`),
		Input: driver.SDKLoopInput{
			Kind: "event",
		},
	}
}

func newFSM(t *testing.T, id string, f inngestgo.LoopFunction[counter, tickData]) inngestgo.Client {
	t.Helper()

	c, _ := captureSync(t)
	_, err := inngestgo.CreateFSM(
		c,
		inngestgo.LoopOpts[counter]{
			FunctionOpts: inngestgo.FunctionOpts{
				ID: id,
			},
			Key: "event.data.id",
		},
		inngestgo.EventTrigger("loop/tick", nil),
		f,
	)
	require.NoError(t, err)
	return c
}

// requireNoRetry asserts a 500 that the executor must not retry.
func requireNoRetry(t *testing.T, rec *httptest.ResponseRecorder, text string) {
	t.Helper()

	require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
	require.Equal(t, "true", rec.Header().Get("X-Inngest-No-Retry"))
	require.Contains(t, rec.Body.String(), text)
	require.NotContains(t, rec.Body.String(), "LoopReturn")
}

func TestLoopWake(t *testing.T) {
	c := newLoop(t, "wake", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
		state.Count++
		return loop.Wake(ctx, "soak", 30*time.Second)
	})

	ret := loopReturn(t, post(t, c, "wake", loopRequest(t, "a", eventLoop())))
	require.JSONEq(t, `{"count":1,"last":""}`, string(ret.State))
	require.False(t, ret.End)
	require.Equal(t, &state.LoopWake{
		ID:    "soak",
		After: "30s",
	}, ret.Wake)
	require.Empty(t, ret.Subscribe)
	require.Empty(t, ret.Unsubscribe)
}

func TestLoopEndWinsOverWake(t *testing.T) {
	c := newLoop(t, "end", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
		if err := loop.Wake(ctx, "soak", time.Minute); err != nil {
			return err
		}
		return loop.End(ctx)
	})

	ret := loopReturn(t, post(t, c, "end", loopRequest(t, "a", eventLoop())))
	require.True(t, ret.End)
	require.Nil(t, ret.Wake)
}

func TestLoopSubscribe(t *testing.T) {
	t.Run("repeated id keeps the last call", func(t *testing.T) {
		c := newLoop(t, "sub", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			if err := loop.Subscribe(ctx, "pr", loop.SubscribeOpts{
				Event: "github/pr.opened",
			}); err != nil {
				return err
			}
			if err := loop.Subscribe(ctx, "pr", loop.SubscribeOpts{
				Event: "github/pr.reviewed",
				If:    "event.data.pr == 123",
			}); err != nil {
				return err
			}
			return loop.Unsubscribe(ctx, "stale")
		})

		ret := loopReturn(t, post(t, c, "sub", loopRequest(t, "a", eventLoop())))
		require.Equal(t, []state.LoopSubscribe{
			{
				ID:    "pr",
				Event: "github/pr.reviewed",
				If:    "event.data.pr == 123",
			},
		}, ret.Subscribe)
		require.Equal(t, []string{"stale"}, ret.Unsubscribe)
		require.False(t, ret.End)
		require.Nil(t, ret.Wake)
	})

	t.Run("unsubscribe then subscribe sends only the subscribe", func(t *testing.T) {
		c := newLoop(t, "resub", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			if err := loop.Unsubscribe(ctx, "pr"); err != nil {
				return err
			}
			return loop.Subscribe(ctx, "pr", loop.SubscribeOpts{
				Event: "github/pr.reviewed",
			})
		})

		ret := loopReturn(t, post(t, c, "resub", loopRequest(t, "a", eventLoop())))
		require.Equal(t, []state.LoopSubscribe{
			{
				ID:    "pr",
				Event: "github/pr.reviewed",
			},
		}, ret.Subscribe)
		require.Empty(t, ret.Unsubscribe)
	})

	t.Run("ttl", func(t *testing.T) {
		ttl := 7 * 24 * time.Hour
		c := newLoop(t, "ttl", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			return loop.Subscribe(ctx, "pr", loop.SubscribeOpts{
				Event: "github/pr.reviewed",
				TTL:   ttl,
			})
		})

		rec := post(t, c, "ttl", loopRequest(t, "a", eventLoop()))
		ret := loopReturn(t, rec)
		require.Len(t, ret.Subscribe, 1)
		require.Equal(t, str2duration.String(ttl), ret.Subscribe[0].TTL)
		parsed, err := str2duration.ParseDuration(ret.Subscribe[0].TTL)
		require.NoError(t, err)
		require.Equal(t, ttl, parsed)

		// a ttl above zero is sent as a str2duration string.
		require.Contains(t, rec.Body.String(), `"ttl"`)
	})

	t.Run("no ttl omits the key", func(t *testing.T) {
		c := newLoop(t, "no-ttl", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			return loop.Subscribe(ctx, "pr", loop.SubscribeOpts{
				Event: "github/pr.reviewed",
			})
		})

		rec := post(t, c, "no-ttl", loopRequest(t, "a", eventLoop()))
		ret := loopReturn(t, rec)
		require.Len(t, ret.Subscribe, 1)
		require.Empty(t, ret.Subscribe[0].TTL)
		require.NotContains(t, rec.Body.String(), `"ttl"`)
	})
}

func TestLoopRejectsWaitingSteps(t *testing.T) {
	t.Run("step.Sleep", func(t *testing.T) {
		c := newLoop(t, "sleep", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			step.Sleep(ctx, "nap", time.Second)
			return nil
		})
		rec := post(t, c, "sleep", loopRequest(t, "a", eventLoop()))
		requireNoRetry(t, rec, "step.Sleep is not allowed in a loop handler")
		require.Contains(t, rec.Body.String(), "loop.Wake")
	})

	t.Run("step.SleepUntil", func(t *testing.T) {
		c := newLoop(t, "sleep-until", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			step.SleepUntil(ctx, "nap", time.Now().Add(time.Second))
			return nil
		})
		rec := post(t, c, "sleep-until", loopRequest(t, "a", eventLoop()))
		requireNoRetry(t, rec, "step.Sleep is not allowed in a loop handler")
		require.Contains(t, rec.Body.String(), "loop.Wake")
	})

	t.Run("step.WaitForEvent", func(t *testing.T) {
		c := newLoop(t, "wait-event", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			_, err := step.WaitForEvent[any](ctx, "wait", step.WaitForEventOpts{
				Event:   "loop/other",
				Timeout: time.Second,
			})
			return err
		})
		rec := post(t, c, "wait-event", loopRequest(t, "a", eventLoop()))
		requireNoRetry(t, rec, "step.WaitForEvent is not allowed in a loop handler")
		require.Contains(t, rec.Body.String(), "loop.Wake")
	})

	t.Run("step.WaitForSignal", func(t *testing.T) {
		c := newLoop(t, "wait-signal", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			_, err := step.WaitForSignal[any](ctx, "wait", step.WaitForSignalOpts{
				Signal:  "go",
				Timeout: time.Second,
			})
			return err
		})
		rec := post(t, c, "wait-signal", loopRequest(t, "a", eventLoop()))
		requireNoRetry(t, rec, "step.WaitForSignal is not allowed in a loop handler")
		require.Contains(t, rec.Body.String(), "loop.Wake")
	})
}

func TestLoopSubscribeInFSM(t *testing.T) {
	c := newFSM(t, "fsm-sub", func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
		return loop.Subscribe(ctx, "pr", loop.SubscribeOpts{
			Event: "github/pr.reviewed",
		})
	})

	rec := post(t, c, "fsm-sub", loopRequest(t, "a", eventLoop()))
	requireNoRetry(t, rec, "fsm")
}

func TestLoopWakeOutsideLoop(t *testing.T) {
	c, _ := captureSync(t)
	_, err := inngestgo.CreateFunction(
		c,
		inngestgo.FunctionOpts{
			ID: "plain",
		},
		inngestgo.EventTrigger("loop/tick", nil),
		func(ctx context.Context, in inngestgo.Input[tickData]) (any, error) {
			return nil, loop.Wake(ctx, "soak", time.Second)
		},
	)
	require.NoError(t, err)

	rec := post(t, c, "plain", loopRequest(t, "a", nil))
	requireNoRetry(t, rec, "loop.Wake called outside a loop function")
}

func TestLoopHijackWithoutOpFailsRun(t *testing.T) {
	c := newLoop(t, "cancelled", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
		state.Count++
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		_, err := step.Run(ctx, "never", func(ctx context.Context) (int, error) {
			return 1, nil
		})
		return err
	})

	rec := post(t, c, "cancelled", loopRequest(t, "a", eventLoop()))

	// a hijack with no op and no error is a 500 the executor retries.
	require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
	require.Empty(t, rec.Header().Get("X-Inngest-No-Retry"))
	require.Contains(t, rec.Body.String(), "loop run ended without a LoopReturn")
	require.NotContains(t, rec.Body.String(), `"op":"LoopReturn"`)
}

func TestLoopCallInsideStep(t *testing.T) {
	c := newLoop(t, "wake-in-step", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
		_, err := step.Run(ctx, "work", func(ctx context.Context) (int, error) {
			return 1, loop.Wake(ctx, "soak", time.Second)
		})
		return err
	})

	rec := post(t, c, "wake-in-step", loopRequest(t, "a", eventLoop()))
	requireNoRetry(t, rec, "loop.Wake called inside a step.  call it from the handler body")
}

func TestLoopArgumentChecks(t *testing.T) {
	cases := []struct {
		name string
		call func(ctx context.Context) error
		text string
	}{
		{
			name: "empty wake id",
			call: func(ctx context.Context) error {
				return loop.Wake(ctx, "", time.Second)
			},
			text: "loop.Wake requires a non-empty id",
		},
		{
			name: "zero wake duration",
			call: func(ctx context.Context) error {
				return loop.Wake(ctx, "soak", 0)
			},
			text: "loop.Wake requires a duration above zero",
		},
		{
			name: "negative wake duration",
			call: func(ctx context.Context) error {
				return loop.Wake(ctx, "soak", -time.Second)
			},
			text: "loop.Wake requires a duration above zero",
		},
		{
			name: "empty subscribe id",
			call: func(ctx context.Context) error {
				return loop.Subscribe(ctx, "", loop.SubscribeOpts{
					Event: "github/pr.reviewed",
				})
			},
			text: "loop.Subscribe requires a non-empty id",
		},
		{
			name: "empty subscribe event",
			call: func(ctx context.Context) error {
				return loop.Subscribe(ctx, "pr", loop.SubscribeOpts{})
			},
			text: "loop.Subscribe requires a non-empty event",
		},
		{
			name: "negative subscribe ttl",
			call: func(ctx context.Context) error {
				return loop.Subscribe(ctx, "pr", loop.SubscribeOpts{
					Event: "github/pr.reviewed",
					TTL:   -time.Second,
				})
			},
			text: "loop.Subscribe requires a ttl of zero or above",
		},
		{
			name: "empty unsubscribe id",
			call: func(ctx context.Context) error {
				return loop.Unsubscribe(ctx, "")
			},
			text: "loop.Unsubscribe requires a non-empty id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			call := tc.call
			c := newLoop(t, "args", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
				return call(ctx)
			})
			rec := post(t, c, "args", loopRequest(t, "a", eventLoop()))
			requireNoRetry(t, rec, tc.text)
		})
	}
}

func TestLoopAllowsRunAndInvoke(t *testing.T) {
	t.Run("step.Run", func(t *testing.T) {
		c := newLoop(t, "run", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			_, err := step.Run(ctx, "work", func(ctx context.Context) (int, error) {
				return 1, nil
			})
			return err
		})

		out := ops(t, post(t, c, "run", loopRequest(t, "a", eventLoop())))
		require.Len(t, out, 1)
		require.Equal(t, enums.OpcodeStepRun, out[0].Op)
	})

	t.Run("step.Invoke", func(t *testing.T) {
		c := newLoop(t, "invoke", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			_, err := step.Invoke[any](ctx, "start", step.InvokeOpts{
				FunctionId: "other-start",
			})
			return err
		})

		out := ops(t, post(t, c, "invoke", loopRequest(t, "a", eventLoop())))
		require.Len(t, out, 1)
		require.Equal(t, enums.OpcodeInvokeFunction, out[0].Op)
	})
}

// TestWaitingStepsOutsideLoop keeps the step tools a loop rejects working
// in a plain function.
func TestWaitingStepsOutsideLoop(t *testing.T) {
	cases := []struct {
		name string
		op   enums.Opcode
		f    inngestgo.SDKFunction[tickData]
	}{
		{
			name: "step.Sleep",
			op:   enums.OpcodeSleep,
			f: func(ctx context.Context, in inngestgo.Input[tickData]) (any, error) {
				step.Sleep(ctx, "nap", time.Second)
				return nil, nil
			},
		},
		{
			name: "step.WaitForEvent",
			op:   enums.OpcodeWaitForEvent,
			f: func(ctx context.Context, in inngestgo.Input[tickData]) (any, error) {
				return step.WaitForEvent[any](ctx, "wait", step.WaitForEventOpts{
					Event:   "loop/other",
					Timeout: time.Second,
				})
			},
		},
		{
			name: "step.WaitForSignal",
			op:   enums.OpcodeWaitForSignal,
			f: func(ctx context.Context, in inngestgo.Input[tickData]) (any, error) {
				return step.WaitForSignal[any](ctx, "wait", step.WaitForSignalOpts{
					Signal:  "go",
					Timeout: time.Second,
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := captureSync(t)
			_, err := inngestgo.CreateFunction(
				c,
				inngestgo.FunctionOpts{
					ID: "plain",
				},
				inngestgo.EventTrigger("loop/tick", nil),
				tc.f,
			)
			require.NoError(t, err)

			out := ops(t, post(t, c, "plain", loopRequest(t, "a", nil)))
			require.Len(t, out, 1)
			require.Equal(t, tc.op, out[0].Op)
			require.NotEmpty(t, out[0].ID)
		})
	}
}

func TestLoopPanicFailsRun(t *testing.T) {
	c := newLoop(t, "panics", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
		state.Count++
		panic("boom")
	})

	rec := post(t, c, "panics", loopRequest(t, "a", eventLoop()))
	require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "function panicked: boom")

	// the body is an error object.  a LoopReturn would be an opcode array.
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotContains(t, body["message"], `"op":"LoopReturn"`)
}
