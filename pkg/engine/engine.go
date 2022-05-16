package engine

import (
	"context"

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

func NewFunctionEngine(o Options) Engine {
	eng := Engine{
		Logger: o.Logger,
	}
	return eng
}

func (eng Engine) Load(ctx context.Context) error {
	eng.Logger.Log(logger.Message{
		Object: "ENGINE",
		Action: "LOAD",
		Msg:    "Loaded 6 functions",
	})

	functions, err := function.LoadRecursive(ctx, "./functions")
	if err != nil {
		return err
	}
	eng.Functions = functions

	for _, f := range eng.Functions {
		eng.Logger.Log(logger.Message{
			Object: "FUNCTION",
			Action: "LOAD",
			Msg:    f.Name,
		})
	}

	return nil
}

func (eng Engine) HandleEvent(e *event.Event) error {

	eng.Logger.Log(logger.Message{
		Object: "FUNCTION",
		Action: "STARTED",
		Msg:    "myFunctionName",
	})
	eng.Logger.Log(logger.Message{
		Object:  "FUNCTION",
		Action:  "COMPLETED",
		Msg:     "myFunctionName",
		Context: "{ \"status\": \"200\" }",
	})
	return nil
}
