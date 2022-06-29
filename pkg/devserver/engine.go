package devserver

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/cli"
	"github.com/inngest/inngest-cli/pkg/event"
	"github.com/inngest/inngest-cli/pkg/execution/actionloader"
	"github.com/inngest/inngest-cli/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/queue"
	"github.com/inngest/inngest-cli/pkg/execution/runner"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/inngest/inngest-cli/pkg/expressions"
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
	al *actionloader.MemoryLoader

	// runner coordinates scheduling of steps between the executor and state queue.
	runner *runner.InMemoryRunner

	// exec is the executor used to run steps and calculate expression data
	// for expressions.
	exec executor.Executor

	// sm stores the in-memory state and queue for our functions.  this is used
	// within the runner.
	sm inmemory.Queue
}

func NewEngine(l *zerolog.Logger) (*Engine, error) {
	logger := l.With().Str("caller", "engine").Logger()

	engineLogger := logger.Output(os.Stderr)
	queueLogger := logger.Output(os.Stderr)

	eng := &Engine{
		log:           &engineLogger,
		EventTriggers: map[string][]function.Function{},
		al:            actionloader.NewMemoryLoader(),
		sm:            NewLoggingQueue(&queueLogger),
	}

	// Create our drivers.
	dd, err := dockerdriver.New()
	if err != nil {
		return nil, err
	}

	// Create an executor with the state manager and drivers.
	exec, err := executor.NewExecutor(
		executor.WithStateManager(eng.sm),
		executor.WithActionLoader(eng.al),
		executor.WithRuntimeDrivers(
			dd,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating executor: %w", err)
	}

	eng.exec = NewLoggingExecutor(exec, l)
	eng.runner = runner.NewInMemoryRunner(eng.sm, eng.exec)

	return eng, nil
}

func (eng *Engine) setExecutor(e executor.Executor) {
	eng.exec = e
	eng.runner = runner.NewInMemoryRunner(eng.sm, eng.exec)
}

// Start starts the runner, blocking until the context is done.
func (eng *Engine) Start(ctx context.Context) error {
	return eng.runner.Start(ctx)
}

// Load loads all functions and their steps from the given directory into the engine.
//
// This replaces all action versions previously loaded, and replaces all function definitions
// previously available.  This allows for hot reloading during development.
func (eng *Engine) Load(ctx context.Context, dir string) error {
	var err error
	eng.log.Info().
		Str("dir", dir).
		Msgf("Recursively loading functions from %s", dir)

	funcs, err := function.LoadRecursive(ctx, dir)
	if err != nil {
		return err
	}

	if len(funcs) == 0 {
		return fmt.Errorf("No functions found in your current directory.  You can create a function by running `inngest init`.")
	}

	eng.log.Info().Int("len", len(funcs)).Msgf("Found functions")

	return eng.SetFunctions(ctx, funcs)
}

func (eng *Engine) SetFunctions(ctx context.Context, functions []*function.Function) error {
	for _, f := range functions {
		if err := f.Validate(ctx); err != nil {
			return fmt.Errorf("error setting functions: %w", err)
		}
	}

	eng.Functions = functions

	// Build all function images.
	if err := eng.buildImages(ctx); err != nil {
		return fmt.Errorf("error building images: %w", err)
	}

	// If a previous cron manager exists, cancel it.
	if eng.cronmanager != nil {
		eng.cronmanager.Stop()
	}

	eng.cronmanager = cron.New(
		cron.WithParser(
			cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		),
	)

	// Set the functions within the engine, then iterate through each function's
	// triggers so that we can easily invoke them.  We also need to immediately
	// set up cron timers to invoke functions on a schedule.
	for _, f := range eng.Functions {
		for _, t := range f.Triggers {
			if t.EventTrigger != nil {
				if _, ok := eng.EventTriggers[t.Event]; !ok {
					eng.EventTriggers[t.Event] = []function.Function{}
				}
				eng.EventTriggers[t.Event] = append(eng.EventTriggers[t.Event], *f)
				continue
			}

			// Set up a cron schedule for the current function.
			_, err := eng.cronmanager.AddFunc(t.Cron, func() {
				_ = eng.execute(ctx, f, &event.Event{})
			})
			if err != nil {
				return err
			}
			eng.log.Info().Str("function", f.Name).Msg("creating scheduled function")
		}

		// For each function, add each action to our actionloader.
		avs, _, _ := f.Actions(ctx)
		for _, a := range avs {
			eng.log.Info().Str("action", a.Name).Msg("making action version available to executor")
			eng.al.Add(a)
		}
	}

	eng.cronmanager.Start()
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

func (eng *Engine) HandleEvent(ctx context.Context, evt *event.Event) error {
	// See if we have any pauses that must be triggered by the event.
	functions, err := eng.findFunctions(context.Background(), evt)
	if err != nil {
		return err
	}

	for _, fn := range functions {
		go func(fn function.Function) {
			_ = eng.execute(context.Background(), &fn, evt)
		}(fn)
	}

	if err := eng.handlePauses(ctx, evt); err != nil {
		return err
	}
	return nil
}

func (eng *Engine) handlePauses(ctx context.Context, evt *event.Event) error {
	if evt == nil {
		return nil
	}
	it, err := eng.sm.PausesByEvent(ctx, evt.Name)
	if err != nil {
		return err
	}

	for it.Next(ctx) {
		pause := it.Val(ctx)

		if pause.Expression != nil {
			s, err := eng.sm.Load(ctx, pause.Identifier)
			if err != nil {
				return err
			}

			// Get expression data from the executor for the given run ID.
			data := state.EdgeExpressionData(ctx, s, pause.Outgoing)
			// Add the async event data to the expression
			data["async"] = evt.Map()

			// Compile and run the expression.
			ok, _, err := expressions.Evaluate(ctx, *pause.Expression, data)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
		}

		// Lease this pause so that only this thread can schedule the execution.
		if err := eng.sm.LeasePause(ctx, pause.ID); err != nil {
			return err
		}

		// Schedule an execution from the pause's entrypoint.
		if err := eng.runner.Enqueue(ctx, queue.Item{
			Identifier: pause.Identifier,
			Payload: queue.PayloadEdge{
				Edge: inngest.Edge{
					Incoming: pause.Incoming,
				},
			},
		}, time.Now()); err != nil {
			return err
		}

		if err := eng.sm.ConsumePause(ctx, pause.ID); err != nil {
			return err
		}
	}

	return nil
}

func (eng *Engine) findFunctions(ctx context.Context, evt *event.Event) ([]function.Function, error) {
	var err error

	funcs, ok := eng.EventTriggers[evt.Name]
	if !ok {
		return nil, nil
	}

	triggered := []function.Function{}
	for _, f := range funcs {
		for _, t := range f.Triggers {
			if t.EventTrigger == nil || t.Event != evt.Name {
				continue
			}

			if t.Expression == nil {
				triggered = append(triggered, f)
				continue
			}

			// Execute expressions here, ensuring that each function is only triggered
			// under the correct conditions.
			ok, _, evalerr := expressions.Evaluate(ctx, *t.Expression, evt.Map())
			if evalerr != nil {
				err = multierror.Append(err, evalerr)
				continue
			}
			if ok {
				triggered = append(triggered, f)
			}
		}
	}

	return triggered, nil
}

func (eng Engine) execute(ctx context.Context, fn *function.Function, evt *event.Event) error {
	data := evt.Map()

	// XXX: We could/should memoize this, though as this is a development engine it's not
	// necessarily a big deal.
	flow, err := fn.Workflow(ctx)
	if err != nil {
		return err
	}

	// Locally, we want to ensure that each function has its own deterministic
	// UUID for managing state.
	//
	// Using a remote API, this UUID may be a surrogate primary key.
	flow.UUID = function.DeterministicUUID(*fn)

	id, err := eng.runner.NewRun(ctx, *flow, data)
	if err != nil {
		return fmt.Errorf("error initializing execution: %w", err)
	}

	log := eng.log.With().
		Str("function", fn.Name).
		Str("event", evt.Name).
		Str("run_id", id.RunID.String()).
		Logger()

	log.Info().Msg("executing function")
	if err = eng.runner.Wait(ctx, *id); err != nil {
		log.Error().Err(err).Msg("executed function")
		return err
	}

	log.Info().Msg("executed function")
	return nil
}
