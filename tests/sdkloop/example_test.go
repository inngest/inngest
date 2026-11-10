package sdkloop

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/loop"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

// this file is the Go port of the connect rollout fsm in loop.md.  state
// holds intent.  the compute list is the truth about what runs.  every
// input ends in the same reconcile.

// rolloutState is the state of one app's rollout.  Phase is stable,
// canary, or rolling.  HealthyAt is the event timestamp in unix
// milliseconds of the run that found the canary or the current batch
// healthy.  zero means not yet.
type rolloutState struct {
	Phase        string `json:"phase"`
	Build        string `json:"build"`
	Desired      int    `json:"desired"`
	BatchPercent int    `json:"batch_percent"`
	HealthyAt    int64  `json:"healthy_at"`
}

// rolloutEvent is the data of both trigger events.  deploy.created sets
// every field.  workload.terminated sets AppID only.  a tick and a wake
// carry none of them.
type rolloutEvent struct {
	AppID    string `json:"app_id"`
	BuildID  string `json:"build_id"`
	Replicas int    `json:"replicas"`
}

// replica is one running workload.
type replica struct {
	ID      string `json:"id"`
	Build   string `json:"build"`
	Started int64  `json:"started"`
}

const (
	canarySoak = 2 * time.Minute
	batchSoak  = 30 * time.Second
)

// newRollout registers the connect rollout fsm.  fake stands in for the
// compute api.
func newRollout(t *testing.T, c inngestgo.Client, fake *compute) {
	t.Helper()

	_, err := inngestgo.CreateFSM(
		c,
		inngestgo.LoopOpts[rolloutState]{
			FunctionOpts: inngestgo.FunctionOpts{
				ID: "connect-app",
			},
			Key:  "event.data.app_id",
			Tick: 30 * time.Second,
			Initial: rolloutState{
				Phase:        "stable",
				BatchPercent: 25,
			},
		},
		inngestgo.MultipleTriggers{
			inngestgo.EventTrigger("compute/deploy.created", nil),
			inngestgo.EventTrigger("compute/workload.terminated", nil),
		},
		func(ctx context.Context, in inngestgo.LoopInput[rolloutEvent], state *rolloutState) error {
			// a tick and a wake carry no app id in their data.  the loop
			// key is the app id for every input.
			appID := in.Loop.Key
			// now is the input's timestamp.  the handler never reads the
			// clock, so a replay sees the same time.
			now := in.Event.Timestamp

			// a new deploy sets the intent.  a terminate, a tick, and a
			// wake change nothing here.
			if in.Loop.Kind == inngestgo.LoopInputEvent && in.Event.Name == "compute/deploy.created" {
				state.Build = in.Event.Data.BuildID
				state.Desired = in.Event.Data.Replicas
				state.Phase = "canary"
				state.HealthyAt = 0
			}

			// every input ends here.
			running, err := step.Run(ctx, "list", func(ctx context.Context) ([]replica, error) {
				return fake.listReplicas(appID), nil
			})
			if err != nil {
				return err
			}
			var current, old []replica
			for _, r := range running {
				if r.Build == state.Build {
					current = append(current, r)
				} else {
					old = append(old, r)
				}
			}

			// keep the count.  canary wants one replica of the new build.
			// rolling and stable want the full desired count, so each old
			// replica that terminates is replaced on the new build until
			// the count is reached.
			want := state.Desired
			if state.Phase == "canary" {
				want = min(1, state.Desired)
			}
			if short := want - len(current); short > 0 {
				// start stands in for a step.Invoke of the start function.
				_, err := step.Run(ctx, "start", func(ctx context.Context) (int, error) {
					return fake.start(appID, state.Build, short), nil
				})
				return err
			}

			if state.Phase == "stable" {
				if extra := len(running) - state.Desired; extra > 0 {
					_, err := step.Run(ctx, "stop extra", func(ctx context.Context) (int, error) {
						return fake.stop(appID, oldest(running, extra)), nil
					})
					return err
				}
				return nil
			}

			health, err := step.Run(ctx, "check", func(ctx context.Context) (string, error) {
				return fake.checkConnected(current), nil
			})
			if err != nil {
				return err
			}
			switch health {
			case "unhealthy":
				_, err := step.Run(ctx, "stop failed", func(ctx context.Context) (int, error) {
					return fake.stop(appID, current), nil
				})
				if err != nil {
					return err
				}
				state.Phase = "stable"
				if len(old) > 0 {
					state.Build = old[0].Build
				}
				return nil
			case "starting":
				return nil
			}

			// healthy.  the first healthy run records its timestamp and
			// asks for a wake when the soak is over.  every later input
			// compares its own timestamp with HealthyAt.  the wake only
			// makes a run happen at the right time.  a wake that arrives
			// early changes nothing.
			soak := batchSoak
			if state.Phase == "canary" {
				soak = canarySoak
			}
			if state.HealthyAt == 0 {
				state.HealthyAt = now
				return loop.Wake(ctx, "soak", soak)
			}
			if now-state.HealthyAt < soak.Milliseconds() {
				return nil
			}

			if len(old) == 0 {
				state.Phase = "stable"
				return nil
			}
			// soaked.  kill the next batch.  the terminate events come back
			// through this function and start replacements above.
			state.Phase = "rolling"
			state.HealthyAt = 0
			batch := max(1, state.Desired*state.BatchPercent/100)
			_, err = step.Run(ctx, "stop batch", func(ctx context.Context) (int, error) {
				return fake.stop(appID, oldest(old, batch)), nil
			})
			return err
		},
	)
	require.NoError(t, err)
}

// compute is an in-memory fake of the compute api.  replicas and stopped
// are keyed by app id, so a call with the wrong app id sees nothing.
// health is what checkConnected reports for any set of replicas.
type compute struct {
	mu       sync.Mutex
	next     int
	replicas map[string][]replica
	stopped  map[string][]replica
	health   string
}

// newCompute returns a fake with n replicas of build under appID that
// report starting.
func newCompute(appID string, build string, n int) *compute {
	c := &compute{
		replicas: map[string][]replica{},
		stopped:  map[string][]replica{},
		health:   "starting",
	}
	c.start(appID, build, n)
	return c
}

func (c *compute) listReplicas(appID string) []replica {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]replica{}, c.replicas[appID]...)
}

func (c *compute) start(appID string, build string, n int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 0; i < n; i++ {
		c.next++
		c.replicas[appID] = append(c.replicas[appID], replica{
			ID:      fmt.Sprintf("r%d", c.next),
			Build:   build,
			Started: int64(c.next),
		})
	}
	return n
}

func (c *compute) checkConnected(current []replica) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.health
}

func (c *compute) setHealth(health string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.health = health
}

func (c *compute) stop(appID string, rs []replica) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	gone := map[string]bool{}
	for _, r := range rs {
		gone[r.ID] = true
		c.stopped[appID] = append(c.stopped[appID], r)
	}
	kept := c.replicas[appID][:0]
	for _, r := range c.replicas[appID] {
		if !gone[r.ID] {
			kept = append(kept, r)
		}
	}
	c.replicas[appID] = kept
	return len(rs)
}

// builds returns the build of each running replica of the app in start
// order.
func (c *compute) builds(appID string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.replicas[appID]))
	for _, r := range c.replicas[appID] {
		out = append(out, r.Build)
	}
	return out
}

// oldest returns the n replicas that started first.
func oldest(rs []replica, n int) []replica {
	sorted := append([]replica{}, rs...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Started < sorted[j].Started
	})
	return sorted[:min(n, len(sorted))]
}

// rolloutRun is one input to the fsm, built the way the executor sends
// it.  name and data are the event.  ts is the event timestamp in unix
// milliseconds.
type rolloutRun struct {
	name  string
	data  any
	ts    int64
	input driver.SDKLoopInput
}

// eventAt is a trigger event for the app.
func eventAt(name string, data rolloutEvent, ts int64) rolloutRun {
	return rolloutRun{
		name: name,
		data: data,
		ts:   ts,
		input: driver.SDKLoopInput{
			Kind: "event",
		},
	}
}

// tickAt is the synthesized tick event.  its data is empty.
func tickAt(ts int64) rolloutRun {
	return rolloutRun{
		name: "inngest/loop.tick",
		data: map[string]any{},
		ts:   ts,
		input: driver.SDKLoopInput{
			Kind: "tick",
		},
	}
}

// wakeAt is the synthesized wake event.  its data holds the wake id only.
func wakeAt(id string, ts int64) rolloutRun {
	return rolloutRun{
		name: "inngest/loop.wake",
		data: map[string]any{
			"id": id,
		},
		ts: ts,
		input: driver.SDKLoopInput{
			Kind: "wake",
			ID:   id,
		},
	}
}

// drive posts one input for the app and replays each step until the
// LoopReturn, the way the executor does.
func drive(t *testing.T, c inngestgo.Client, appID string, st rolloutState, run rolloutRun) (*state.LoopReturn, rolloutState) {
	t.Helper()

	stateByt, err := json.Marshal(st)
	require.NoError(t, err)
	evt := map[string]any{
		"name": run.name,
		"data": run.data,
		"ts":   run.ts,
	}
	loopReq := &driver.SDKLoopRequest{
		State: stateByt,
		Key:   appID,
		Input: run.input,
	}

	steps := map[string]any{}
	for i := 0; i < 10; i++ {
		out := ops(t, post(t, c, "connect-app", request(t, evt, loopReq, steps)))
		require.Len(t, out, 1)
		if out[0].Op == enums.OpcodeLoopReturn {
			ret, err := out[0].LoopReturn()
			require.NoError(t, err)
			var next rolloutState
			require.NoError(t, json.Unmarshal(ret.State, &next))
			return ret, next
		}
		require.Equal(t, enums.OpcodeStepRun, out[0].Op)
		steps[out[0].ID] = out[0].Data
	}
	t.Fatal("run did not end in a LoopReturn")
	return nil, rolloutState{}
}

func TestRolloutExample(t *testing.T) {
	const appID = "app"
	c, _ := captureSync(t)
	fake := newCompute(appID, "b1", 4)
	newRollout(t, c, fake)

	// at is the event timestamp d after the deploy.
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(d time.Duration) int64 {
		return base.Add(d).UnixMilli()
	}

	st := rolloutState{
		Phase:        "stable",
		BatchPercent: 25,
	}

	// a deploy moves the app to canary and starts one replica of the
	// new build.
	ret, st := drive(t, c, appID, st, eventAt("compute/deploy.created", rolloutEvent{
		AppID:    appID,
		BuildID:  "b2",
		Replicas: 4,
	}, at(0)))
	require.Equal(t, rolloutState{
		Phase:        "canary",
		Build:        "b2",
		Desired:      4,
		BatchPercent: 25,
	}, st)
	require.Nil(t, ret.Wake)
	require.Equal(t, []string{"b1", "b1", "b1", "b1", "b2"}, fake.builds(appID))

	// a tick while the canary starts changes nothing.
	ret, next := drive(t, c, appID, st, tickAt(at(30*time.Second)))
	require.Equal(t, st, next)
	require.Nil(t, ret.Wake)

	// the first healthy tick records its timestamp and asks for a wake
	// at the end of the canary soak.
	fake.setHealth("healthy")
	ret, st = drive(t, c, appID, st, tickAt(at(60*time.Second)))
	require.Equal(t, "canary", st.Phase)
	require.Equal(t, at(60*time.Second), st.HealthyAt)
	require.Equal(t, &state.LoopWake{
		ID:    "soak",
		After: "2m",
	}, ret.Wake)

	// a tick inside the soak window changes nothing.
	ret, next = drive(t, c, appID, st, tickAt(at(90*time.Second)))
	require.Equal(t, st, next)
	require.Nil(t, ret.Wake)
	require.Equal(t, []string{"b1", "b1", "b1", "b1", "b2"}, fake.builds(appID))

	// a wake that arrives before the soak window has passed changes
	// nothing.  the wake id alone does not end the soak.
	ret, next = drive(t, c, appID, st, wakeAt("soak", at(120*time.Second)))
	require.Equal(t, st, next)
	require.Nil(t, ret.Wake)
	require.Equal(t, []string{"b1", "b1", "b1", "b1", "b2"}, fake.builds(appID))
	require.Empty(t, fake.stopped[appID])

	// the wake at the end of the soak window ends it.  the app moves to
	// rolling and the oldest quarter of the old build is stopped.
	ret, st = drive(t, c, appID, st, wakeAt("soak", at(180*time.Second)))
	require.Equal(t, rolloutState{
		Phase:        "rolling",
		Build:        "b2",
		Desired:      4,
		BatchPercent: 25,
	}, st)
	require.Nil(t, ret.Wake)
	require.Equal(t, []string{"b1", "b1", "b1", "b2"}, fake.builds(appID))
	require.Equal(t, "r1", fake.stopped[appID][0].ID)

	// the terminate event comes back through the fsm.  rolling wants the
	// full count, so the replacements start on the new build.
	_, st = drive(t, c, appID, st, eventAt("compute/workload.terminated", rolloutEvent{
		AppID: appID,
	}, at(181*time.Second)))
	require.Equal(t, "rolling", st.Phase)
	require.Equal(t, []string{"b1", "b1", "b1", "b2", "b2", "b2", "b2"}, fake.builds(appID))
}
