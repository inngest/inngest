package engine

import (
	"context"
	"fmt"

	"github.com/inngest/inngestctl/pkg/event"
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
	for _, f := range functions {
		eng.Functions = append(eng.Functions, f)
		eng.Logger.Log(logger.Message{
			Object: "FUNCTION",
			Action: "LOADED",
			Msg:    f.Name,
		})
	}

	eng.Logger.Log(logger.Message{
		Object: "ENGINE",
		Msg:    fmt.Sprintf("Loaded %d functions", len(eng.Functions)),
	})
	return nil
}

func (eng *Engine) HandleEvent(e *event.Event) error {
	functions, err := eng.FindFunctionsByEvent(context.Background(), e)
	if err != nil {
		return nil
	}
	if len(functions) == 0 {
		eng.Logger.Log(logger.Message{
			Object: "ENGINE",
			Msg:    fmt.Sprintf("No matching function triggers for %s", e.Name),
		})
		return nil
	}
	for _, f := range functions {
		eng.ExecuteFunction(context.Background(), f, e)
	}
	return nil
}

func (eng *Engine) FindFunctionsByEvent(ctx context.Context, e *event.Event) ([]*function.Function, error) {
	var functions []*function.Function
	for _, f := range eng.Functions {
		for _, t := range f.Triggers {
			if t.Event == e.Name {
				functions = append(functions, f)
			}
		}
	}
	return functions, nil
}

func (eng Engine) ExecuteFunction(ctx context.Context, f *function.Function, e *event.Event) error {
	eng.Logger.Log(logger.Message{
		Object: "FUNCTION",
		Action: "STARTED",
		Msg:    f.Name,
	})
	// TODO - Execute function
	eng.Logger.Log(logger.Message{
		Object:  "FUNCTION",
		Action:  "COMPLETED",
		Msg:     f.Name,
		Context: "{ \"status\": \"200\" }",
	})
	return nil
}
