package engine

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/event"
	"github.com/inngest/inngestctl/pkg/execution/actionloader"
	"github.com/inngest/inngestctl/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngestctl/pkg/execution/executor"
	"github.com/inngest/inngestctl/pkg/execution/runner"
	"github.com/inngest/inngestctl/pkg/execution/state/inmemory"
	"github.com/inngest/inngestctl/pkg/expressions"
	"github.com/inngest/inngestctl/pkg/function"
	cron "github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type Options struct {
	Logger *zerolog.Logger
}

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

	exec executor.Executor

	// sm stores the in-memory state and queue for our functions.  this is used
	// within the runner.
	sm inmemory.Queue
}

func New(o Options) (*Engine, error) {
	logger := o.Logger.With().Str("caller", "engine").Logger()

	eng := &Engine{
		log:           &logger,
		EventTriggers: map[string][]function.Function{},
		al:            actionloader.NewMemoryLoader(),
		sm:            NewLoggingQueue(o.Logger),
	}

	// Create our drivers.
	dd, err := dockerdriver.New()
	if err != nil {
		return nil, err
	}

	// Create an executor with the state manager and drivers.
	eng.exec, err = executor.NewExecutor(
		executor.WithStateManager(eng.sm),
		executor.WithActionLoader(eng.al),
		executor.WithRuntimeDrivers(
			dd,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating executor: %w", err)
	}

	return eng, nil
}

func (eng *Engine) Load(ctx context.Context, dir string) error {
	var err error
	eng.log.Info().
		Str("dir", dir).
		Msgf("Recursively loading functions from %s", dir)

	if eng.Functions, err = function.LoadRecursive(ctx, dir); err != nil {
		return err
	}

	eng.log.Info().Int("len", len(eng.Functions)).Msgf("Found functions")

	// Build all function images.
	if err := eng.buildImages(ctx); err != nil {
		return err
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

	// XXX: Depending on the type of output here, we want to either
	// create an interactive UI for building images or show JSON output
	// as we build.
	ui, err := cli.NewBuilder(ctx, cli.BuilderUIOpts{
		QuitOnComplete: true,
		BuildOpts:      opts,
	})
	if err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
	if err := tea.NewProgram(ui).Start(); err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}

	return ui.Error()
}

func (eng *Engine) HandleEvent(evt *event.Event) error {
	functions, err := eng.findFunctions(context.Background(), evt)
	if err != nil {
		return err
	}
	if len(functions) == 0 {
		return nil
	}
	for _, fn := range functions {
		go func(fn function.Function) {
			_ = eng.execute(context.Background(), &fn, evt)
		}(fn)
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

	runlog := eng.log.With().Str("caller", "runner").Logger()
	runner := runner.NewInMemoryRunner(
		eng.sm,
		NewLoggingExecutor(eng.exec, &runlog),
	)

	id, err := runner.NewRun(ctx, *flow, data)
	if err != nil {
		return fmt.Errorf("error initializing execution: %w", err)
	}

	log := eng.log.With().
		Str("function", fn.Name).
		Str("event", evt.Name).
		Str("run_id", id.RunID.String()).
		Logger()

	log.Info().Msg("executing function")
	if err = runner.Execute(ctx, *id); err != nil {
		log.Error().Err(err).Msg("executed function")
		return err
	}
	log.Info().Msg("executed function")
	return nil
}
