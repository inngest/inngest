package devserver

import (
	"context"

	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
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
	loader, err := inmemory.New(ctx)
	if err != nil {
		return err
	}
	return start(ctx, opts, loader)
}

func start(ctx context.Context, opts StartOpts, loader *inmemory.ReadWriter) error {

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

	sm, err := opts.Config.State.Service.Concrete.Manager(ctx)
	if err != nil {
		return err
	}

	runner := runner.NewService(
		opts.Config,
		runner.WithExecutionLoader(loader),
		runner.WithEventManager(event.NewManager()),
		runner.WithStateManager(sm),
	)

	// The devserver embeds the event API.
	ds := newService(opts, loader, runner)
	exec := executor.NewService(
		opts.Config,
		executor.WithExecutionLoader(loader),
		executor.WithEnvReader(envreader),
		executor.WithState(sm),
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
