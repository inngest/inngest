package engine

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/event"
	"github.com/inngest/inngestctl/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/inngest/inngestctl/pkg/logger"
	cron "github.com/robfig/cron/v3"
)

type Options struct {
	Logger logger.Logger
}

type Engine struct {
	Logger    logger.Logger
	Functions []*function.Function

	// EventTriggers stores a map of event triggers to the function that
	// it triggers for lookups when receiving an event from the dev server.
	EventTriggers map[string][]function.Function

	// cronmanager stores a reference to a cron manager for invoking scheduled
	// functions.  we store a reference so that we can terminate the crons when
	// recreating the engine.
	cronmanager *cron.Cron
}

func New(o Options) *Engine {
	eng := &Engine{
		Logger:        o.Logger,
		EventTriggers: map[string][]function.Function{},
	}
	return eng
}

func (eng *Engine) Load(ctx context.Context, dir string) error {
	eng.Logger.Log(logger.Message{
		Object: "ENGINE",
		Msg:    fmt.Sprintf("Recursively loading functions from %s", dir),
	})

	functions, err := function.LoadRecursive(ctx, dir)
	if err != nil {
		return err
	}

	eng.Logger.Log(logger.Message{
		Object: "ENGINE",
		Msg:    fmt.Sprintf("Found %d functions", len(eng.Functions)),
	})

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
	eng.Functions = functions
	for _, f := range functions {
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
				eng.execute(ctx, f, &event.Event{})
			})
			if err != nil {
				return err
			}
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
	functions, err := eng.findFunction(context.Background(), evt)
	if err != nil {
		return nil
	}
	if len(functions) == 0 {
		eng.Logger.Log(logger.Message{
			Object: "ENGINE",
			Msg:    fmt.Sprintf("No matching function triggers for %s", evt.Name),
		})
		return nil
	}
	for _, fn := range functions {
		err := eng.execute(context.Background(), &fn, evt)
		if err != nil {
			eng.Logger.Log(logger.Message{
				Object:  "FUNCTION",
				Action:  "FAILED",
				Msg:     fn.Name,
				Context: err,
			})
		}
	}
	return nil
}

func (eng *Engine) findFunction(ctx context.Context, evt *event.Event) ([]function.Function, error) {
	funcs, _ := eng.EventTriggers[evt.Name]
	// TODO: Execute expressions here.
	return funcs, nil
}

func (eng Engine) execute(ctx context.Context, fn *function.Function, evt *event.Event) error {
	eventMap, err := evt.Map()
	if err != nil {
		return err
	}
	ui, err := cli.NewRunUI(ctx, cli.RunUIOpts{
		Function: *fn,
		Event:    eventMap,
	})
	if err != nil {
		return err
	}

	eng.Logger.Log(logger.Message{
		Object: "FUNCTION",
		Action: "STARTED",
		Msg:    fn.Name,
	})

	if err := tea.NewProgram(ui).Start(); err != nil {
		return err
	}
	return ui.Error()
}
