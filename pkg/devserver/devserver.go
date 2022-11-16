package devserver

import (
	"context"
	"fmt"
	"os"

	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coreapi"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
	inmemorydatastore "github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/function/env"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
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
//
// - Builds locally defined docker-based functions using Buildx
// - Adds development-specific APIs for communicating with the SDK.
func New(ctx context.Context, opts StartOpts) error {
	// The dev server _always_ logs output for development.
	if !opts.Config.Execution.LogOutput {
		logger.From(ctx).Info().Msg("overriding config to log step output within dev server")
		opts.Config.Execution.LogOutput = true
	}

	// we run this before initializing the devserver serivce (even though it has Pre)
	// because building images should happen and error early, prior to any other
	// service starting.
	el, err := prepareDockerImages(ctx, opts.RootDir)
	if err != nil {
		return err
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

	runner := runner.NewService(opts.Config, runner.WithExecutionLoader(loader), runner.WithEventManager(event.NewManager()))

	// The devserver embeds the event API.
	ds := newService(opts, loader, runner)
	exec := executor.NewService(
		opts.Config,
		executor.WithExecutionLoader(loader),
		executor.WithEnvReader(envreader),
	)
	coreapi := coreapi.NewService(opts.Config, coreapi.WithRunner(runner))

	return service.StartAll(ctx, ds, runner, exec, coreapi)
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
