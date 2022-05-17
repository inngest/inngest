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
	for _, fn := range functions {
		eng.Functions = append(eng.Functions, fn)
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
		eng.ExecuteFunction(context.Background(), fn, evt)
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
	eng.Logger.Log(logger.Message{
		Object: "FUNCTION",
		Action: "STARTED",
		Msg:    fn.Name,
	})
	// TODO - Execute function
	eng.Logger.Log(logger.Message{
		Object:  "FUNCTION",
		Action:  "COMPLETED",
		Msg:     fn.Name,
		Context: "{ \"status\": \"200\" }",
	})
	return nil
}
