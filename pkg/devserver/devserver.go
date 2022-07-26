package devserver

import (
	"context"
	"fmt"
	"os"

	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coredata"
	inmemorydatastore "github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
)

// Create and start a new dev server (API, Exectutor, State, Logger, etc.)
func NewDevServer(ctx context.Context, c config.Config, dir string) error {
	// Create a new filesystem loader.
	el, err := inmemorydatastore.NewFSLoader(ctx, dir)
	if err != nil {
		return err
	}

	funcs, err := el.Functions(ctx)
	if err != nil {
		return err
	}

	// For each function, build the image.
	if err := buildImages(ctx, funcs); err != nil {
		return err
	}

	// The dev server _always_ logs output for development.
	if !c.Execution.LogOutput {
		logger.From(ctx).Info().Msg("overriding config to log step output within dev server")
		c.Execution.LogOutput = true
	}

	return newDevServer(ctx, c, el)
}

func newDevServer(ctx context.Context, c config.Config, el coredata.ExecutionLoader) error {
	api := api.NewService(c)
	runner := runner.NewService(c, runner.WithExecutionLoader(el))
	exec := executor.NewService(c, executor.WithExecutionLoader(el))
	return service.StartAll(ctx, api, runner, exec)
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
