package sdkloop

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

type counter struct {
	Count int    `json:"count"`
	Last  string `json:"last"`
}

type tickData struct {
	ID string `json:"id"`
}

func handler(ctx context.Context, input inngestgo.LoopInput[tickData], state *counter) error {
	state.Count++
	state.Last = input.Event.Data.ID
	return nil
}

// captureSync returns a client in dev mode whose out of band sync posts to
// a local server, and the register body that server received.
func captureSync(t *testing.T) (inngestgo.Client, func() []byte) {
	t.Helper()

	var mu sync.Mutex
	var body []byte
	var readErr error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		byt, err := io.ReadAll(r.Body)
		mu.Lock()
		body = byt
		readErr = err
		mu.Unlock()
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(ts.Close)

	c, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:       "loops",
		Dev:         new(true),
		RegisterURL: new(ts.URL),
	})
	require.NoError(t, err)

	return c, func() []byte {
		mu.Lock()
		defer mu.Unlock()
		require.NoError(t, readErr)
		return body
	}
}

func syncPayload(t *testing.T, c inngestgo.Client, captured func() []byte) (sdk.RegisterRequest, map[string]any) {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "http://localhost/api/inngest", nil)
	c.Serve().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body := captured()
	require.NotEmpty(t, body)

	var typed sdk.RegisterRequest
	require.NoError(t, json.Unmarshal(body, &typed))

	var raw map[string]any
	require.NoError(t, json.Unmarshal(body, &raw))

	return typed, raw
}

func TestSyncCarriesLoopConfig(t *testing.T) {
	ctx := context.Background()
	c, captured := captureSync(t)

	initial := counter{
		Count: 5,
		Last:  "seed",
	}

	loopFn, err := inngestgo.CreateLoop(
		c,
		inngestgo.LoopOpts[counter]{
			FunctionOpts: inngestgo.FunctionOpts{
				ID: "counter-loop",
			},
			Key:     "event.data.id",
			Tick:    30 * time.Second,
			IdleTTL: 10 * time.Minute,
			Initial: initial,
		},
		inngestgo.EventTrigger("loop/tick", nil),
		handler,
	)
	require.NoError(t, err)

	fsmFn, err := inngestgo.CreateFSM(
		c,
		inngestgo.LoopOpts[counter]{
			FunctionOpts: inngestgo.FunctionOpts{
				ID: "counter-fsm",
			},
			Key:  "event.data.id",
			Tick: 30 * time.Second,
		},
		inngestgo.EventTrigger("fsm/tick", nil),
		handler,
	)
	require.NoError(t, err)

	want, err := json.Marshal(initial)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(loopFn.(interface{ InitialState() json.RawMessage }).InitialState()))
	require.JSONEq(t, `{"count":0,"last":""}`, string(fsmFn.(interface{ InitialState() json.RawMessage }).InitialState()))

	// a loop gets the same checkpoint default as a normal function.
	plainClient, _ := captureSync(t)
	plainFn, err := inngestgo.CreateFunction(
		plainClient,
		inngestgo.FunctionOpts{
			ID: "plain",
		},
		inngestgo.EventTrigger("plain/tick", nil),
		func(ctx context.Context, input inngestgo.Input[tickData]) (any, error) {
			return nil, nil
		},
	)
	require.NoError(t, err)
	require.NotNil(t, plainFn.Config().Checkpoint)
	require.NotNil(t, loopFn.Config().Checkpoint)
	require.False(t, loopFn.Config().Checkpoint.Disable)
	require.Equal(t, plainFn.Config().Checkpoint, loopFn.Config().Checkpoint)
	require.Equal(t, plainFn.Config().Checkpoint, fsmFn.Config().Checkpoint)

	typed, raw := syncPayload(t, c, captured)

	require.Equal(t, sdk.LoopV1, typed.Capabilities.Loop)
	require.Len(t, typed.Functions, 2)

	byID := map[string]sdk.SDKFunction{}
	for _, f := range typed.Functions {
		byID[f.Slug] = f
	}

	loop := byID["loops-counter-loop"]
	require.NotNil(t, loop.Loop)
	require.Equal(t, "loop", loop.Loop.Kind)
	require.Equal(t, "event.data.id", loop.Loop.Key)
	require.Equal(t, 30*time.Second, loop.Loop.Tick)
	require.Equal(t, 10*time.Minute, loop.Loop.IdleTTL)
	require.NotNil(t, loop.Checkpoint)
	require.NotNil(t, byID["loops-counter-fsm"].Checkpoint)

	fsm := byID["loops-counter-fsm"]
	require.NotNil(t, fsm.Loop)
	require.Equal(t, "fsm", fsm.Loop.Kind)
	require.Equal(t, "event.data.id", fsm.Loop.Key)
	require.Equal(t, 30*time.Second, fsm.Loop.Tick)
	require.Equal(t, consts.DefaultLoopIdleTTL, fsm.Loop.IdleTTL)

	for _, f := range typed.Functions {
		built, err := f.Function()
		require.NoError(t, err, f.Slug)
		require.NotNil(t, built.Loop, f.Slug)
		require.NoError(t, built.Validate(ctx), f.Slug)
	}

	// the wire form uses str2duration strings and drops zero durations.
	rawByID := map[string]map[string]any{}
	for _, f := range raw["functions"].([]any) {
		m := f.(map[string]any)
		rawByID[m["id"].(string)] = m["loop"].(map[string]any)
	}
	require.Equal(t, "30s", rawByID["loops-counter-loop"]["tick"])
	require.Equal(t, "10m", rawByID["loops-counter-loop"]["idle_ttl"])
	require.Equal(t, "30s", rawByID["loops-counter-fsm"]["tick"])
	_, hasIdle := rawByID["loops-counter-fsm"]["idle_ttl"]
	require.False(t, hasIdle)
}

func TestCreateLoopRejectsOptions(t *testing.T) {
	c, _ := captureSync(t)

	create := func(opts inngestgo.LoopOpts[counter]) error {
		_, err := inngestgo.CreateLoop(c, opts, inngestgo.EventTrigger("loop/tick", nil), handler)
		return err
	}

	base := func() inngestgo.LoopOpts[counter] {
		return inngestgo.LoopOpts[counter]{
			FunctionOpts: inngestgo.FunctionOpts{
				ID: "rejected",
			},
			Key: "event.data.id",
		}
	}

	t.Run("empty key", func(t *testing.T) {
		opts := base()
		opts.Key = ""
		require.ErrorContains(t, create(opts), "key is required")
	})

	t.Run("singleton", func(t *testing.T) {
		opts := base()
		opts.Singleton = &inngestgo.ConfigSingleton{
			Mode: enums.SingletonModeSkip,
		}
		require.ErrorContains(t, create(opts), "singleton")
	})

	t.Run("batch events", func(t *testing.T) {
		opts := base()
		opts.BatchEvents = &inngestgo.ConfigBatchEvents{
			MaxSize: 2,
			Timeout: time.Second,
		}
		require.ErrorContains(t, create(opts), "batching")
	})

	t.Run("debounce", func(t *testing.T) {
		opts := base()
		opts.Debounce = &inngestgo.ConfigDebounce{
			Period: time.Second,
		}
		require.ErrorContains(t, create(opts), "debouncing")
	})

	t.Run("keyed step concurrency", func(t *testing.T) {
		opts := base()
		opts.Concurrency = &inngestgo.ConfigConcurrency{
			Step: []inngestgo.ConfigStepConcurrency{
				{
					Limit: 1,
					Key:   new("event.data.id"),
				},
			},
		}
		require.ErrorContains(t, create(opts), "keyed step concurrency")
	})

	t.Run("function concurrency", func(t *testing.T) {
		opts := base()
		opts.Concurrency = &inngestgo.ConfigConcurrency{
			Fn: []inngestgo.ConfigFnConcurrency{
				{
					Limit: 1,
				},
			},
		}
		require.ErrorContains(t, create(opts), "function concurrency")
	})

	t.Run("unkeyed step concurrency is allowed", func(t *testing.T) {
		opts := base()
		opts.Concurrency = &inngestgo.ConfigConcurrency{
			Step: []inngestgo.ConfigStepConcurrency{
				{
					Limit: 1,
				},
			},
		}
		require.NoError(t, create(opts))
	})

	t.Run("negative tick", func(t *testing.T) {
		opts := base()
		opts.Tick = -time.Second
		require.ErrorContains(t, create(opts), "tick must not be negative")
	})

	t.Run("negative idle ttl", func(t *testing.T) {
		opts := base()
		opts.IdleTTL = -time.Second
		require.ErrorContains(t, create(opts), "idle ttl must not be negative")
	})
}

func mapHandler(ctx context.Context, input inngestgo.LoopInput[tickData], state *map[string]int) error {
	(*state)[input.Event.Data.ID]++
	return nil
}

func TestCreateLoopRejectsPointerState(t *testing.T) {
	c, _ := captureSync(t)

	opts := inngestgo.LoopOpts[*counter]{
		FunctionOpts: inngestgo.FunctionOpts{
			ID: "pointer-state",
		},
		Key:     "event.data.id",
		Initial: &counter{},
	}
	f := func(ctx context.Context, in inngestgo.LoopInput[tickData], state **counter) error {
		return nil
	}

	t.Run("loop", func(t *testing.T) {
		_, err := inngestgo.CreateLoop(c, opts, inngestgo.EventTrigger("loop/tick", nil), f)
		require.ErrorContains(t, err, "STATE must not be a pointer type")
	})

	t.Run("fsm", func(t *testing.T) {
		_, err := inngestgo.CreateFSM(c, opts, inngestgo.EventTrigger("loop/tick", nil), f)
		require.ErrorContains(t, err, "STATE must not be a pointer type")
	})
}

func TestCreateLoopRejectsNullInitial(t *testing.T) {
	c, _ := captureSync(t)

	create := func(opts inngestgo.LoopOpts[map[string]int]) error {
		opts.ID = "map-loop"
		opts.Key = "event.data.id"
		_, err := inngestgo.CreateLoop(c, opts, inngestgo.EventTrigger("loop/tick", nil), mapHandler)
		return err
	}

	t.Run("nil map initial", func(t *testing.T) {
		require.ErrorContains(t, create(inngestgo.LoopOpts[map[string]int]{}), "Initial marshals to null")
	})

	t.Run("empty map initial", func(t *testing.T) {
		require.NoError(t, create(inngestgo.LoopOpts[map[string]int]{
			Initial: map[string]int{},
		}))
	})
}
