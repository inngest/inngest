package sdkloop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/pkg/checkpoint"
	"github.com/inngest/inngestgo/step"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// loopRequest builds the executor request for a loop run of the test
// counter functions.  id is the event key and is sent as the loop key.  a
// nil loop block is sent as no loop key.
func loopRequest(t *testing.T, id string, loop *driver.SDKLoopRequest) []byte {
	t.Helper()

	if loop != nil {
		keyed := *loop
		keyed.Key = id
		loop = &keyed
	}
	return request(t, map[string]any{
		"name": "loop/tick",
		"data": map[string]any{
			"id": id,
		},
	}, loop, nil)
}

// request builds the executor request with the given event, loop block,
// and memoized step results.  a nil loop block is sent as no loop key.
func request(t *testing.T, evt map[string]any, loop *driver.SDKLoopRequest, steps map[string]any) []byte {
	t.Helper()

	if steps == nil {
		steps = map[string]any{}
	}
	req := driver.SDKRequest{
		Event:   evt,
		Actions: steps,
		Context: &driver.SDKRequestContext{
			FunctionID: uuid.New(),
			RunID:      ulid.Make(),
			Attempt:    0,
			Stack: &driver.FunctionStack{
				Stack:   []string{},
				Current: 0,
			},
		},
		Loop: loop,
	}
	byt, err := json.Marshal(req)
	require.NoError(t, err)
	return byt
}

// post sends the request body to the handler for the given function and
// returns the recorded response.
func post(t *testing.T, c inngestgo.Client, fnID string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://localhost/api/inngest?fnId="+c.AppID()+"-"+fnID, bytes.NewReader(body))
	c.Serve().ServeHTTP(rec, req)
	return rec
}

// ops decodes a 206 body.
func ops(t *testing.T, rec *httptest.ResponseRecorder) []state.GeneratorOpcode {
	t.Helper()

	require.Equal(t, http.StatusPartialContent, rec.Code, rec.Body.String())
	var out []state.GeneratorOpcode
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

// loopReturn decodes a 206 body that holds exactly one LoopReturn.
func loopReturn(t *testing.T, rec *httptest.ResponseRecorder) *state.LoopReturn {
	t.Helper()

	out := ops(t, rec)
	require.Len(t, out, 1)
	require.Equal(t, enums.OpcodeLoopReturn, out[0].Op)
	require.Equal(t, "loop", out[0].Name)
	require.NotEmpty(t, out[0].ID)
	ret, err := out[0].LoopReturn()
	require.NoError(t, err)
	return ret
}

func newLoop(t *testing.T, id string, initial counter, f inngestgo.LoopFunction[counter, tickData]) inngestgo.Client {
	t.Helper()

	c, _ := captureSync(t)
	_, err := inngestgo.CreateLoop(
		c,
		inngestgo.LoopOpts[counter]{
			FunctionOpts: inngestgo.FunctionOpts{
				ID: id,
			},
			Key:     "event.data.id",
			Initial: initial,
		},
		inngestgo.EventTrigger("loop/tick", nil),
		f,
	)
	require.NoError(t, err)
	return c
}

func TestInvokeNullStateUsesInitial(t *testing.T) {
	initial := counter{
		Count: 5,
		Last:  "seed",
	}

	var seen counter
	var seenLoop inngestgo.LoopCtx
	c := newLoop(t, "null-state", initial, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
		seen = *state
		seenLoop = in.Loop
		return nil
	})

	wake := driver.SDKLoopInput{
		Kind: "wake",
		ID:   "soak",
	}

	t.Run("state is null", func(t *testing.T) {
		seen = counter{}
		rec := post(t, c, "null-state", loopRequest(t, "a", &driver.SDKLoopRequest{
			State:   json.RawMessage("null"),
			Version: 7,
			Input:   wake,
		}))
		ret := loopReturn(t, rec)

		require.Equal(t, initial, seen)
		require.Equal(t, inngestgo.LoopCtx{
			Kind:    inngestgo.LoopInputWake,
			ID:      "soak",
			Key:     "a",
			Version: 7,
		}, seenLoop)
		require.JSONEq(t, `{"count":5,"last":"seed"}`, string(ret.State))
		require.Equal(t, int64(7), ret.Version)
		require.False(t, ret.End)
		require.Nil(t, ret.Wake)
	})

	t.Run("state is absent", func(t *testing.T) {
		seen = counter{}
		body := loopRequest(t, "a", &driver.SDKLoopRequest{
			Version: 3,
			Input:   wake,
		})

		// drop the state key from the loop block.
		var raw map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(body, &raw))
		var loop map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw["loop"], &loop))
		delete(loop, "state")
		loopByt, err := json.Marshal(loop)
		require.NoError(t, err)
		raw["loop"] = loopByt
		body, err = json.Marshal(raw)
		require.NoError(t, err)

		rec := post(t, c, "null-state", body)
		ret := loopReturn(t, rec)

		require.Equal(t, initial, seen)
		require.Equal(t, int64(3), seenLoop.Version)
		require.JSONEq(t, `{"count":5,"last":"seed"}`, string(ret.State))
		require.Equal(t, int64(3), ret.Version)
	})

	t.Run("version is absent", func(t *testing.T) {
		seen = counter{}
		body := loopRequest(t, "a", &driver.SDKLoopRequest{
			State:   json.RawMessage("null"),
			Version: 9,
			Input:   wake,
		})

		// drop the version key from the loop block.
		var raw map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(body, &raw))
		var loop map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw["loop"], &loop))
		delete(loop, "version")
		loopByt, err := json.Marshal(loop)
		require.NoError(t, err)
		raw["loop"] = loopByt
		body, err = json.Marshal(raw)
		require.NoError(t, err)

		rec := post(t, c, "null-state", body)
		ret := loopReturn(t, rec)

		require.Equal(t, initial, seen)
		require.Equal(t, int64(0), seenLoop.Version)
		require.JSONEq(t, `{"count":5,"last":"seed"}`, string(ret.State))
		require.Equal(t, int64(0), ret.Version)
	})
}

func TestInvokeReturnsMutatedState(t *testing.T) {
	c := newLoop(t, "mutate", counter{}, handler)

	rec := post(t, c, "mutate", loopRequest(t, "b", &driver.SDKLoopRequest{
		State:   json.RawMessage(`{"count":2,"last":"a"}`),
		Version: 1,
		Input: driver.SDKLoopInput{
			Kind: "event",
		},
	}))
	ret := loopReturn(t, rec)

	require.JSONEq(t, `{"count":3,"last":"b"}`, string(ret.State))
	require.Equal(t, int64(1), ret.Version)
	require.False(t, ret.End)
	require.Nil(t, ret.Wake)
	require.Empty(t, ret.Subscribe)
	require.Empty(t, ret.Unsubscribe)
}

func TestInvokePointerSwapIsLost(t *testing.T) {
	t.Run("new pointer is lost", func(t *testing.T) {
		c := newLoop(t, "swap", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			other := counter{
				Count: state.Count + 97,
				Last:  "other",
			}
			state = &other
			state.Count++
			return nil
		})

		rec := post(t, c, "swap", loopRequest(t, "a", &driver.SDKLoopRequest{
			State:   json.RawMessage(`{"count":2,"last":"a"}`),
			Version: 1,
			Input: driver.SDKLoopInput{
				Kind: "event",
			},
		}))
		ret := loopReturn(t, rec)

		require.JSONEq(t, `{"count":2,"last":"a"}`, string(ret.State))
	})

	t.Run("assignment through the pointer is committed", func(t *testing.T) {
		c := newLoop(t, "assign", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			*state = counter{
				Count: 99,
				Last:  "other",
			}
			return nil
		})

		rec := post(t, c, "assign", loopRequest(t, "a", &driver.SDKLoopRequest{
			State:   json.RawMessage(`{"count":2,"last":"a"}`),
			Version: 1,
			Input: driver.SDKLoopInput{
				Kind: "event",
			},
		}))
		ret := loopReturn(t, rec)

		require.JSONEq(t, `{"count":99,"last":"other"}`, string(ret.State))
	})
}

func TestInvokeHandlerError(t *testing.T) {
	c := newLoop(t, "fails", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
		state.Count++
		return errors.New("boom")
	})

	rec := post(t, c, "fails", loopRequest(t, "c", &driver.SDKLoopRequest{
		State: json.RawMessage(`{"count":0,"last":""}`),
		Input: driver.SDKLoopInput{
			Kind: "event",
		},
	}))

	require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "boom")
	require.NotContains(t, rec.Body.String(), "LoopReturn")
	require.Empty(t, rec.Header().Get("X-Inngest-No-Retry"))
}

func TestInvokeStepThenReturn(t *testing.T) {
	c := newLoop(t, "stepped", counter{}, func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
		n, err := step.Run(ctx, "count", func(ctx context.Context) (int, error) {
			return 10, nil
		})
		if err != nil {
			return err
		}
		state.Count += n
		state.Last = in.Event.Data.ID
		return nil
	})

	loop := &driver.SDKLoopRequest{
		State:   json.RawMessage(`{"count":1,"last":""}`),
		Version: 2,
		Input: driver.SDKLoopInput{
			Kind: "tick",
		},
	}

	// the first request hijacks on the step and sends no LoopReturn.
	first := ops(t, post(t, c, "stepped", loopRequest(t, "d", loop)))
	require.Len(t, first, 1)
	require.Equal(t, enums.OpcodeStepRun, first[0].Op)
	require.NotEmpty(t, first[0].ID)

	// the second request replays the step from memory and returns the
	// LoopReturn.
	body := loopRequest(t, "d", loop)
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &raw))
	steps, err := json.Marshal(map[string]json.RawMessage{
		first[0].ID: first[0].Data,
	})
	require.NoError(t, err)
	raw["steps"] = steps
	body, err = json.Marshal(raw)
	require.NoError(t, err)

	ret := loopReturn(t, post(t, c, "stepped", body))
	require.JSONEq(t, `{"count":11,"last":"d"}`, string(ret.State))
	require.Equal(t, loop.Version, ret.Version)
	require.False(t, ret.End)
}

func TestInvokeWithoutLoopBlock(t *testing.T) {
	c := newLoop(t, "no-block", counter{}, handler)

	rec := post(t, c, "no-block", loopRequest(t, "e", nil))

	require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
	require.Equal(t, "true", rec.Header().Get("X-Inngest-No-Retry"))
	require.Contains(t, rec.Body.String(), "without a loop block")
}

// marked uses pointer receivers for both JSON methods.  the wire form adds
// a key that the struct does not hold, which shows the methods ran.
type marked struct {
	N int
}

func (m *marked) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"n":      m.N,
		"marked": true,
	})
}

func (m *marked) UnmarshalJSON(byt []byte) error {
	var raw struct {
		N      int  `json:"n"`
		Marked bool `json:"marked"`
	}
	if err := json.Unmarshal(byt, &raw); err != nil {
		return err
	}
	if !raw.Marked {
		return errors.New("state was not marshalled through marked.MarshalJSON")
	}
	m.N = raw.N
	return nil
}

func TestInvokePointerReceiverMarshalJSON(t *testing.T) {
	c, _ := captureSync(t)
	fn, err := inngestgo.CreateLoop(
		c,
		inngestgo.LoopOpts[marked]{
			FunctionOpts: inngestgo.FunctionOpts{
				ID: "marked",
			},
			Key: "event.data.id",
			Initial: marked{
				N: 1,
			},
		},
		inngestgo.EventTrigger("loop/tick", nil),
		func(ctx context.Context, in inngestgo.LoopInput[tickData], state *marked) error {
			state.N++
			return nil
		},
	)
	require.NoError(t, err)

	initial := fn.(interface{ InitialState() json.RawMessage }).InitialState()
	require.JSONEq(t, `{"n":1,"marked":true}`, string(initial))

	t.Run("initial state", func(t *testing.T) {
		rec := post(t, c, "marked", loopRequest(t, "f", &driver.SDKLoopRequest{
			State: json.RawMessage("null"),
			Input: driver.SDKLoopInput{
				Kind: "event",
			},
		}))
		ret := loopReturn(t, rec)
		require.JSONEq(t, `{"n":2,"marked":true}`, string(ret.State))
		require.Equal(t, int64(0), ret.Version)
	})

	t.Run("committed state", func(t *testing.T) {
		rec := post(t, c, "marked", loopRequest(t, "f", &driver.SDKLoopRequest{
			State:   json.RawMessage(`{"n":5,"marked":true}`),
			Version: 1,
			Input: driver.SDKLoopInput{
				Kind: "event",
			},
		}))
		ret := loopReturn(t, rec)
		require.JSONEq(t, `{"n":6,"marked":true}`, string(ret.State))
	})

	t.Run("state that skipped MarshalJSON is rejected", func(t *testing.T) {
		rec := post(t, c, "marked", loopRequest(t, "f", &driver.SDKLoopRequest{
			State: json.RawMessage(`{"n":5}`),
			Input: driver.SDKLoopInput{
				Kind: "event",
			},
		}))
		require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
		require.Equal(t, "true", rec.Header().Get("X-Inngest-No-Retry"))
	})
}

// checkpointServer returns a server that records each checkpoint POST body
// and answers with the given status.
func checkpointServer(t *testing.T, status int) (*httptest.Server, func() []map[string]any) {
	t.Helper()

	var mu sync.Mutex
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		w.WriteHeader(status)
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(ts.Close)

	return ts, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(bodies)
	}
}

// newCheckpointLoop builds a loop that runs one step and adds its result to
// the count.  the client sends checkpoints to apiURL.
func newCheckpointLoop(t *testing.T, apiURL string, cfg *checkpoint.Config) inngestgo.Client {
	t.Helper()

	c, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:      "loops",
		Dev:        new(true),
		APIBaseURL: new(apiURL),
	})
	require.NoError(t, err)

	_, err = inngestgo.CreateLoop(
		c,
		inngestgo.LoopOpts[counter]{
			FunctionOpts: inngestgo.FunctionOpts{
				ID:         "checkpointed",
				Checkpoint: cfg,
			},
			Key: "event.data.id",
		},
		inngestgo.EventTrigger("loop/tick", nil),
		func(ctx context.Context, in inngestgo.LoopInput[tickData], state *counter) error {
			n, err := step.Run(ctx, "count", func(ctx context.Context) (int, error) {
				return 10, nil
			})
			if err != nil {
				return err
			}
			state.Count += n
			state.Last = in.Event.Data.ID
			return nil
		},
	)
	require.NoError(t, err)
	return c
}

func TestInvokeCheckpointedStepThenReturn(t *testing.T) {
	ts, received := checkpointServer(t, http.StatusOK)
	c := newCheckpointLoop(t, ts.URL, checkpoint.ConfigSafe)

	loop := &driver.SDKLoopRequest{
		State:   json.RawMessage(`{"count":1,"last":""}`),
		Version: 2,
		Input: driver.SDKLoopInput{
			Kind: "tick",
		},
	}

	// the step checkpoints in the request, so the response is one LoopReturn
	// with no StepRun next to it.
	ret := loopReturn(t, post(t, c, "checkpointed", loopRequest(t, "d", loop)))
	require.JSONEq(t, `{"count":11,"last":"d"}`, string(ret.State))
	require.Equal(t, loop.Version, ret.Version)

	bodies := received()
	require.Len(t, bodies, 1)
	steps, ok := bodies[0]["steps"].([]any)
	require.True(t, ok)
	require.Len(t, steps, 1)
	require.Equal(t, enums.OpcodeStepRun.String(), steps[0].(map[string]any)["op"])
}

func TestInvokeCheckpointFailureYields(t *testing.T) {
	ts, received := checkpointServer(t, http.StatusInternalServerError)
	c := newCheckpointLoop(t, ts.URL, checkpoint.ConfigSafe)

	loop := &driver.SDKLoopRequest{
		State:   json.RawMessage(`{"count":1,"last":""}`),
		Version: 2,
		Input: driver.SDKLoopInput{
			Kind: "tick",
		},
	}

	// the checkpoint call fails, so the step goes to the executor in the
	// response and no LoopReturn is sent.
	out := ops(t, post(t, c, "checkpointed", loopRequest(t, "d", loop)))
	require.Len(t, out, 1)
	require.Equal(t, enums.OpcodeStepRun, out[0].Op)
	require.NotEmpty(t, received())
}

func TestInvokeBatchedCheckpointReturnsStepThenLoopReturn(t *testing.T) {
	ts, received := checkpointServer(t, http.StatusOK)
	c := newCheckpointLoop(t, ts.URL, &checkpoint.Config{
		BatchSteps: 5,
	})

	loop := &driver.SDKLoopRequest{
		State:   json.RawMessage(`{"count":1,"last":""}`),
		Version: 2,
		Input: driver.SDKLoopInput{
			Kind: "tick",
		},
	}

	// the batch is not full when the handler returns, so the step is still
	// in the response and the LoopReturn comes after it.
	out := ops(t, post(t, c, "checkpointed", loopRequest(t, "d", loop)))
	require.Len(t, out, 2)
	require.Equal(t, enums.OpcodeStepRun, out[0].Op)
	require.Equal(t, "count", out[0].Name)
	require.Equal(t, enums.OpcodeLoopReturn, out[1].Op)

	ret, err := out[1].LoopReturn()
	require.NoError(t, err)
	require.JSONEq(t, `{"count":11,"last":"d"}`, string(ret.State))
	require.Equal(t, loop.Version, ret.Version)
	require.Empty(t, received())
}
