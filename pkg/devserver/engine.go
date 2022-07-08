package devserver

/*

import (
	"context"
	"fmt"
	"os"

	"github.com/inngest/inngest-cli/pkg/cli"
	"github.com/inngest/inngest-cli/pkg/coredata"
	"github.com/inngest/inngest-cli/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/runner"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/inngest/inngest-cli/pkg/function"
	cron "github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// Engine bundles together a runner, in-memory state manager, and local functions
// for use within a local dev server, for development only.
type Engine struct {
	Functions []*function.Function

	// EventTriggers stores a map of event triggers to the function that
	// it triggers for lookups when receiving an event from the dev server.
	EventTriggers map[string][]function.Function

	// cronmanager stores a reference to a cron manager for invoking scheduled
	// functions.  we store a reference so that we can terminate the crons when
	// recreating the engine.
	cronmanager *cron.Cron

	log *zerolog.Logger

	// al is n action loader which stores actions in memory.
	al coredata.ExecutionActionLoader

	// runner coordinates scheduling of steps between the executor and state queue.
	runner *runner.InMemoryRunner

	// exec is the executor used to run steps and calculate expression data
	// for expressions.
	exec executor.Executor

	// sm stores the in-memory state and queue for our functions.  this is used
	// within the runner.
	sm inmemory.Queue
}

// Start starts the runner, blocking until the context is done.
func (eng *Engine) Start(ctx context.Context) error {
	return eng.runner.Start(ctx)
}

func (eng *Engine) SetFunctions(ctx context.Context, functions []*function.Function) error {
	// xxx: handled in coredata.

	// Build all function images.
	if err := eng.buildImages(ctx); err != nil {
		return fmt.Errorf("error building images: %w", err)
	}

	return nil
}

// buildImages builds all images hosted within the engine.  This iterates through all
// functions discovered during Load.
func (eng Engine) buildImages(ctx context.Context) error {
	opts := []dockerdriver.BuildOpts{}

	for _, fn := range eng.Functions {
		steps, err := dockerdriver.FnBuildOpts(ctx, *fn)
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
*/
