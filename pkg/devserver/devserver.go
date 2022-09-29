package devserver

import (
	"context"
	"fmt"
	"os"

	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/config"
	inmemorydatastore "github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/function/env"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
)

// Create and start a new dev server.  The dev server is used during (surprise surprise)
// development.
//
// It runs all available services from `inngest serve`, plus:
//
// - Builds locally defined docker-based functions using Buildx
// - Adds development-specific APIs for communicating with the SDK.
func NewDevServer(ctx context.Context, c config.Config, dir string) error {
	// The dev server _always_ logs output for development.
	if !c.Execution.LogOutput {
		logger.From(ctx).Info().Msg("overriding config to log step output within dev server")
		c.Execution.LogOutput = true
	}

	// we run this before initializing the devserver serivce (even though it has Pre)
	// because building images should happen and error early, prior to any other
	// service starting.
	el, err := prepareDockerImages(ctx, dir)
	if err != nil {
		return err
	}

	return start(ctx, c, el, dir)
}

func start(ctx context.Context, c config.Config, loader *inmemorydatastore.FSLoader, dir string) error {
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

	ds := &devserver{
		rootDir: dir,
		loader:  loader,
	}

	api := api.NewService(c)
	runner := runner.NewService(c, runner.WithExecutionLoader(loader))
	exec := executor.NewService(
		c,
		executor.WithExecutionLoader(loader),
		executor.WithEnvReader(envreader),
	)

	return service.StartAll(ctx, ds, api, runner, exec)
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
