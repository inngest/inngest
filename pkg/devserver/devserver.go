package devserver

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
	inmemorydatastore "github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/function/env"
	"github.com/inngest/inngest/pkg/service"
)

// StartOpts configures the dev server
type StartOpts struct {
	Config       config.Config `json:"-"`
	RootDir      string        `json:"dir"`
	URLs         []string      `json:"urls"`
	Autodiscover bool          `json:"autodiscover"`
	Docker       bool          `json:"docker"`
}

// Create and start a new dev server.  The dev server is used during (surprise surprise)
// development.
//
// It runs all available services from `inngest serve`, plus:
//
// - Builds locally defined docker-based functions using Buildx
// - Adds development-specific APIs for communicating with the SDK.
func New(ctx context.Context, opts StartOpts) error {
	// The dev server _always_ logs output for development.
	if !opts.Config.Execution.LogOutput {
		opts.Config.Execution.LogOutput = true
	}

	// we run this before initializing the devserver serivce (even though it has Pre)
	// because building images should happen and error early, prior to any other
	// service starting.
	el, _ := inmemorydatastore.NewEmptyFSLoader(ctx, opts.RootDir)
	if opts.Docker {
		var err error
		if el, err = prepareDockerImages(ctx, opts.RootDir); err != nil {
			return err
		}
	}

	return start(ctx, opts, el)
}

func start(ctx context.Context, opts StartOpts, loader *inmemorydatastore.FSLoader) error {
	inmemory.NewInMemoryAPIReadWriter()

	funcs, err := loader.Functions(ctx)
	if err != nil {
		return err
	}

	// create a new env reader which will load .env files from functions directly, each
	// time the executor runs.
	envreader, err := env.NewReader(funcs)
	if err != nil {
		return err
	}

	rc, err := createInmemoryRedis()
	if err != nil {
		return err
	}

	var sm state.Manager
	t := runner.NewTracker()
	sm, err = redis_state.New(
		ctx,
		redis_state.WithRedisClient(rc),
		redis_state.WithFunctionCallbacks(func(ctx context.Context, i state.Identifier, rs enums.RunStatus) {
			switch rs {
			case enums.RunStatusRunning:
				// Add this to the in-memory tracker.
				// XXX: Switch to sqlite.
				s, err := sm.Load(ctx, i.RunID)
				if err != nil {
					return
				}
				t.Add(s.Event()["id"].(string), s.Metadata())
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
		redis_state.WithPartitionConcurrencyKeyGenerator(func(ctx context.Context, p redis_state.QueuePartition) (string, int) {
			// Ensure that we return the correct concurrency values per
			// partition.
			funcs, _ := loader.Functions(ctx)
			for _, f := range funcs {
				if f.ID == p.WorkflowID.String() {
					return p.Queue(), f.Concurrency
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
	exec := executor.NewService(
		opts.Config,
		executor.WithExecutionLoader(loader),
		executor.WithEnvReader(envreader),
		executor.WithState(sm),
		executor.WithQueue(queue),
	)

	return service.StartAll(ctx, ds, runner, exec)
}

func createInmemoryRedis() (redis.UniversalClient, error) {
	r := miniredis.NewMiniRedis()
	_ = r.Start()
	rc := redis.NewClient(&redis.Options{
		Addr:     r.Addr(),
		PoolSize: 100,
	})
	go func() {
		for range time.Tick(time.Second) {
			r.FastForward(time.Second)
		}
	}()
	return rc, rc.Ping(context.Background()).Err()
}

func prepareDockerImages(ctx context.Context, dir string) (*inmemorydatastore.FSLoader, error) {
	// Create a new filesystem loader.
	el, err := inmemorydatastore.NewFSLoader(ctx, dir)
	if err != nil {
		return nil, err
	}

	funcs, err := el.Functions(ctx)
	if err != nil {
		return nil, err
	}

	// For each function, build the image.
	if err := buildImages(ctx, funcs); err != nil {
		return nil, err
	}

	return el.(*inmemorydatastore.FSLoader), nil
}

// buildImages builds all images hosted within the engine.  This iterates through all
// functions discovered during Load.
func buildImages(ctx context.Context, funcs []function.Function) error {
	opts := []dockerdriver.BuildOpts{}

	for _, fn := range funcs {
		steps, err := dockerdriver.FnBuildOpts(ctx, fn)
		if err != nil {
			return err
		}
		opts = append(opts, steps...)
	}

	if len(opts) == 0 {
		return nil
	}

	ui, err := cli.NewBuilder(ctx, cli.BuilderUIOpts{
		QuitOnComplete: true,
		BuildOpts:      opts,
	})
	if err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
	// calling Start on our UI instance invokes either a pretty TTY output
	// via tea, or renders output as JSON directly depending on the global
	// JSON flag.
	if err := ui.Start(ctx); err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}

	return ui.Error()
}
