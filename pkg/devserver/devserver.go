package devserver

import (
	"context"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/config"
	_ "github.com/inngest/inngest/pkg/config/defaults"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/service"
	"github.com/rueian/rueidis"
)

// StartOpts configures the dev server
type StartOpts struct {
	Config       config.Config `json:"-"`
	RootDir      string        `json:"dir"`
	URLs         []string      `json:"urls"`
	Autodiscover bool          `json:"autodiscover"`
}

// Create and start a new dev server.  The dev server is used during (surprise surprise)
// development.
//
// It runs all available services from `inngest serve`, plus:
// - Adds development-specific APIs for communicating with the SDK.
func New(ctx context.Context, opts StartOpts) error {
	// The dev server _always_ logs output for development.
	if !opts.Config.Execution.LogOutput {
		opts.Config.Execution.LogOutput = true
	}
	loader, err := inmemory.New(ctx)
	if err != nil {
		return err
	}
	return start(ctx, opts, loader)
}

func start(ctx context.Context, opts StartOpts, loader *inmemory.ReadWriter) error {
	rc, err := createInmemoryRedis(ctx)
	if err != nil {
		return err
	}

	var sm state.Manager
	t := runner.NewTracker()
	sm, err = redis_state.New(
		ctx,
		redis_state.WithFunctionLoader(loader),
		redis_state.WithRedisClient(rc),
		redis_state.WithKeyGenerator(redis_state.DefaultKeyFunc{
			Prefix: "{state}",
		}),
		redis_state.WithFunctionCallbacks(func(ctx context.Context, i state.Identifier, rs enums.RunStatus) {
			switch rs {
			case enums.RunStatusRunning:
				// Add this to the in-memory tracker.
				// XXX: Switch to sqlite.
				s, err := sm.Load(ctx, i.RunID)
				if err != nil {
					return
				}
				t.Add(s.Event()["id"].(string), i)
			}
		}),
	)
	if err != nil {
		return err
	}

	queue := redis_state.NewQueue(
		rc,
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithNumWorkers(100),
		redis_state.WithPollTick(250*time.Millisecond),
		redis_state.WithQueueKeyGenerator(&redis_state.DefaultQueueKeyGenerator{
			Prefix: "{queue}",
		}),
		redis_state.WithPartitionConcurrencyKeyGenerator(func(ctx context.Context, p redis_state.QueuePartition) (string, int) {
			// Ensure that we return the correct concurrency values per
			// partition.
			funcs, _ := loader.Functions(ctx)
			for _, f := range funcs {
				if f.ID == uuid.Nil {
					f.ID = inngest.DeterministicUUID(f)
				}
				if f.ID == p.WorkflowID && f.ConcurrencyLimit() > 0 {
					return p.Queue(), f.ConcurrencyLimit()
				}
			}
			return p.Queue(), 10_000
		}),
	)

	runner := runner.NewService(
		opts.Config,
		runner.WithExecutionLoader(loader),
		runner.WithEventManager(event.NewManager()),
		runner.WithStateManager(sm),
		runner.WithQueue(queue),
		runner.WithTracker(t),
	)

	// The devserver embeds the event API.
	ds := newService(opts, loader, runner)
	// embed the tracker
	ds.tracker = t
	ds.state = sm

	// Create an executor.
	exec := executor.NewService(
		opts.Config,
		executor.WithExecutionLoader(loader),
		executor.WithState(sm),
		executor.WithQueue(queue),
		executor.WithExecutorOpts(
			executor.WithFunctionLoader(loader),
		),
	)

	// Add notifications to the state manager so that we can store new function runs
	// in the core API service.
	if notify, ok := sm.(state.FunctionNotifier); ok {
		notify.OnFunctionStatus(func(ctx context.Context, id state.Identifier, rs enums.RunStatus) {
			switch rs {
			case enums.RunStatusRunning:
				// A new function was added, so add this to the core API
				// for listing functions by XYZ.
			}
		})
	}

	return service.StartAll(ctx, ds, runner, exec)
}

func createInmemoryRedis(ctx context.Context) (rueidis.Client, error) {
	r := miniredis.NewMiniRedis()
	_ = r.Start()
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	if err != nil {
		return nil, err
	}
	go func() {
		for range time.Tick(time.Second) {
			r.FastForward(time.Second)
		}
	}()
	return rc, nil
}
