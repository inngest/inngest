package engine

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/event"
	"github.com/inngest/inngestctl/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/inngest/inngestctl/pkg/logger"
)

type Options struct {
	Logger logger.Logger
}

type Engine struct {
	Logger    logger.Logger
	Functions []*function.Function
}

func NewFunctionEngine(o Options) *Engine {
	eng := &Engine{
		Logger: o.Logger,
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
	for _, fn := range functions {
		eng.Functions = append(eng.Functions, fn)
		// TODO - Build images, ideally in the background w/ some sort of readiness message via the logger
		// err := eng.BuildImages(ctx, fn)
		// if err != nil {
		// 	return err
		// }
		eng.Logger.Log(logger.Message{
			Object: "FUNCTION",
			Action: "LOADED",
			Msg:    fn.Name,
		})
	}

	eng.Logger.Log(logger.Message{
		Object: "ENGINE",
		Msg:    fmt.Sprintf("Loaded %d functions", len(eng.Functions)),
	})
	return nil
}

func (eng Engine) BuildImages(ctx context.Context, fn *function.Function) error {
	a, _, _ := fn.Actions(ctx)
	if a[0].Runtime.RuntimeType() != inngest.RuntimeTypeDocker {
		return nil
	}

	ui, err := cli.NewBuilder(ctx, cli.BuilderUIOpts{
		QuitOnComplete: true,
		BuildOpts: dockerdriver.BuildOpts{
			Path: ".",
			Tag:  a[0].DSN,
		},
	})
	if err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
	if err := tea.NewProgram(ui).Start(); err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
	return ui.Builder.Error()
}

func (eng *Engine) HandleEvent(evt *event.Event) error {
	functions, err := eng.FindFunctionsByEvent(context.Background(), evt)
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
		err := eng.ExecuteFunction(context.Background(), fn, evt)
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

func (eng *Engine) FindFunctionsByEvent(ctx context.Context, evt *event.Event) ([]*function.Function, error) {
	var functions []*function.Function
	for _, fn := range eng.Functions {
		for _, t := range fn.Triggers {
			if t.Event == evt.Name {
				functions = append(functions, fn)
			}
		}
	}
	return functions, nil
}

func (eng Engine) ExecuteFunction(ctx context.Context, fn *function.Function, evt *event.Event) error {
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
