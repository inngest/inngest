package golang

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/loop"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLoop(t *testing.T) {
	ctx := context.Background()
	c := client.New(t)

	appID := "loop-" + ulid.MustNew(ulid.Now(), nil).String()
	inngestClient, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	const (
		trigger   = "test/loop"
		tickEvent = "inngest/loop.tick"
		wakeEvent = "inngest/loop.wake"
		tick      = 3 * time.Second
		wakeAfter = 2 * time.Second
	)

	rec := &recorder{}

	_, err := inngestgo.CreateLoop[loopState, loopEvent](
		inngestClient,
		inngestgo.LoopOpts[loopState]{
			FunctionOpts: inngestgo.FunctionOpts{
				ID:      "loop",
				Retries: new(0),
			},
			Key:     "event.data.key",
			Tick:    tick,
			Initial: loopState{Age: 0},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, in inngestgo.LoopInput[loopEvent], state *loopState) error {
			rec.add(seen{
				Kind:      string(in.Loop.Kind),
				ID:        in.Loop.ID,
				Key:       in.Loop.Key,
				Version:   in.Loop.Version,
				AgeBefore: state.Age,
				RunID:     in.InputCtx.RunID,
				Event:     in.Event.Name,
				At:        time.Now(),
			})
			state.Age++

			switch in.Event.Data.Action {
			case "wake":
				return loop.Wake(ctx, "w", wakeAfter)
			case "end":
				return loop.End(ctx)
			}
			return nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	// send sends one test/loop event for key and returns the send time.
	send := func(t *testing.T, key, action string) time.Time {
		t.Helper()
		_, err := inngestClient.Send(ctx, inngestgo.Event{
			Name: trigger,
			Data: map[string]any{
				"key":    key,
				"action": action,
			},
		})
		require.NoError(t, err)
		return time.Now()
	}

	isEvent := func(key string) func(seen) bool {
		return func(s seen) bool {
			return s.Kind == string(inngestgo.LoopInputEvent) && s.Key == key && s.Event == trigger
		}
	}
	isKey := func(key string) func(seen) bool {
		return func(s seen) bool {
			return s.Key == key
		}
	}

	// the sections run in order and share the instance for key "a".  each
	// section leaves the instance in the shape the next one expects.

	t.Run("first input uses the initial state", func(t *testing.T) {
		// the first event for a key starts an instance from Initial.  the
		// handler sees age 0 at version 0.  the run output is the state the
		// run committed, which is age 1.
		send(t, "a", "")

		var first *seen
		waitFor(t, 20*time.Second, func() bool {
			first = rec.find(func(s seen) bool {
				return isEvent("a")(s) && s.AgeBefore == 0 && s.Version == 0
			})
			return first != nil
		})

		run := c.WaitForRunStatus(ctx, t, "COMPLETED", first.RunID)
		require.JSONEq(t, `{"name":"","age":1}`, run.Output)
	})

	t.Run("state carries across inputs", func(t *testing.T) {
		// the second event for the same key loads the committed state.  a
		// tick can run between the first run and this send, so the handler
		// sees an age and version of at least 1.  the run commits that age
		// plus one.
		before := len(rec.filter(isEvent("a")))
		send(t, "a", "")

		var second *seen
		waitFor(t, 20*time.Second, func() bool {
			events := rec.filter(isEvent("a"))
			if len(events) <= before {
				return false
			}
			second = &events[len(events)-1]
			return true
		})
		require.GreaterOrEqual(t, second.AgeBefore, 1)
		require.GreaterOrEqual(t, second.Version, int64(1))

		run := c.WaitForRunStatus(ctx, t, "COMPLETED", second.RunID)
		require.JSONEq(t, fmt.Sprintf(`{"name":"","age":%d}`, second.AgeBefore+1), run.Output)
	})

	t.Run("ticks arrive while the instance exists", func(t *testing.T) {
		// a tick is scheduled one interval after each run finishes.  the
		// instance for key "a" exists, so ticks keep arriving.  each tick
		// loads the state the previous run committed, so age and version
		// both increase across ticks.
		isTick := func(s seen) bool {
			return s.Kind == string(inngestgo.LoopInputTick) && s.Key == "a" && s.Event == tickEvent
		}

		var ticks []seen
		waitFor(t, 10*time.Second, func() bool {
			ticks = rec.filter(isTick)
			return len(ticks) >= 2
		})

		require.Empty(t, ticks[0].ID)
		require.Greater(t, ticks[1].AgeBefore, ticks[0].AgeBefore)
		require.Greater(t, ticks[1].Version, ticks[0].Version)
	})

	t.Run("a wake arrives once at the requested time", func(t *testing.T) {
		// loop.Wake schedules one input with the caller's id.  it arrives no
		// earlier than the requested duration after the run that asked for
		// it, and it arrives exactly once.
		isWake := func(s seen) bool {
			return s.Kind == string(inngestgo.LoopInputWake) && s.ID == "w" && s.Key == "a" && s.Event == wakeEvent
		}

		sent := send(t, "a", "wake")

		var wake *seen
		waitFor(t, 6*time.Second, func() bool {
			wake = rec.find(isWake)
			return wake != nil
		})
		require.GreaterOrEqual(t, wake.At.Sub(sent), wakeAfter)

		<-time.After(3 * time.Second)
		require.Len(t, rec.filter(isWake), 1)
	})

	t.Run("keys are isolated", func(t *testing.T) {
		// a new key starts its own instance from Initial.  the instance for
		// key "a" keeps its state.
		send(t, "b", "")

		waitFor(t, 20*time.Second, func() bool {
			return rec.find(func(s seen) bool {
				return isEvent("b")(s) && s.AgeBefore == 0 && s.Version == 0
			}) != nil
		})

		latestA := rec.last(isKey("a"))
		require.NotNil(t, latestA)
		require.Greater(t, latestA.AgeBefore, 0)
	})

	t.Run("end deletes the instance", func(t *testing.T) {
		// loop.End deletes the state and the pending tick after the run
		// commits.  no tick arrives for the key after that.  the next event
		// for the key starts a fresh instance from Initial.
		before := len(rec.filter(isEvent("a")))
		send(t, "a", "end")

		var end *seen
		waitFor(t, 20*time.Second, func() bool {
			events := rec.filter(isEvent("a"))
			if len(events) <= before {
				return false
			}
			end = &events[len(events)-1]
			return true
		})
		c.WaitForRunStatus(ctx, t, "COMPLETED", end.RunID)

		// two tick intervals pass with no input for the key.
		quiet := len(rec.filter(isKey("a")))
		<-time.After(2*tick + time.Second)
		require.Len(t, rec.filter(isKey("a")), quiet)

		send(t, "a", "")

		var fresh *seen
		waitFor(t, 20*time.Second, func() bool {
			events := rec.filter(isEvent("a"))
			if len(events) <= before+1 {
				return false
			}
			fresh = &events[len(events)-1]
			return true
		})
		require.Equal(t, 0, fresh.AgeBefore)
		require.Equal(t, int64(0), fresh.Version)
	})
}

// loopState is the per-key state of the loop under test.  the committed
// state is the output of the run that committed it.
type loopState struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// loopEvent is the data of the test/loop trigger.  Key picks the instance.
// Action picks what the handler returns.
type loopEvent struct {
	Key    string `json:"key"`
	Action string `json:"action"`
}

// seen is one handler invocation.  AgeBefore is the state age the run
// loaded, before the handler increments it.
type seen struct {
	Kind      string
	ID        string
	Key       string
	Version   int64
	AgeBefore int
	RunID     string
	Event     string
	At        time.Time
}

// recorder collects every invocation of the loop handler.  the handler
// runs on server goroutines, so every read copies under the lock.
type recorder struct {
	mu      sync.Mutex
	entries []seen
}

func (r *recorder) add(s seen) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, s)
}

// all returns a copy of every entry in arrival order.
func (r *recorder) all() []seen {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]seen, len(r.entries))
	copy(out, r.entries)
	return out
}

// find returns the first entry that matches pred, or nil.
func (r *recorder) find(pred func(seen) bool) *seen {
	for _, s := range r.all() {
		if pred(s) {
			return &s
		}
	}
	return nil
}

// last returns the newest entry that matches pred, or nil.
func (r *recorder) last(pred func(seen) bool) *seen {
	all := r.all()
	for i := len(all) - 1; i >= 0; i-- {
		if pred(all[i]) {
			return &all[i]
		}
	}
	return nil
}

// filter returns every entry that matches pred in arrival order.
func (r *recorder) filter(pred func(seen) bool) []seen {
	var out []seen
	for _, s := range r.all() {
		if pred(s) {
			out = append(out, s)
		}
	}
	return out
}

// waitFor polls cond until it returns true.  the test fails after timeout.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	require.Eventually(t, cond, timeout, 50*time.Millisecond)
}
