package devserver

import (
	"github.com/inngest/inngestctl/pkg/api"
	"github.com/inngest/inngestctl/pkg/event"
	"github.com/inngest/inngestctl/pkg/logger"
	"github.com/inngest/inngestctl/pkg/logger/stdoutlogger"
)

type Options struct {
	Port         string
	PrettyOutput bool
}

// Create and start a new dev server (API, Exectutor, State, Logger, etc.)
func NewDevServer(o Options) error {
	l := stdoutlogger.NewLogger(logger.Options{
		Pretty: o.PrettyOutput,
	})

	// TODO - Init "Loader" to load all functions into memory and pass to Executor
	// TODO - ExecutorRegistry? - something that loads functions from disk and builds a registry of them to then execute

	// TODO - Create Executor instance / load inngest.json files, pass logger to Executor
	// executor, err := executor.NewExecutor(executor.Options{}
	registry := NewExecutorRegistry(EROptions{
		logger: l,
	})

	err := api.NewAPI(api.Options{
		Port:         o.Port,
		EventHandler: registry.Handler,
		Logger:       l,
	})

	return err
}

type EROptions struct {
	logger logger.Logger
}

type ExecutorRegistry struct {
	logger logger.Logger
}

func NewExecutorRegistry(o EROptions) ExecutorRegistry {
	r := ExecutorRegistry{
		logger: o.logger,
	}
	r.Load()
	return r
}

func (r *ExecutorRegistry) Load() error {
	r.logger.Log(logger.Message{
		Object: "REGISTRY",
		Action: "LOAD",
		Msg:    "Loaded 6 functions",
	})
	return nil
}

func (r *ExecutorRegistry) Handler(e *event.Event) error {
	r.logger.Log(logger.Message{
		Object: "FUNCTION",
		Action: "STARTED",
		Msg:    "myFunctionName",
	})
	r.logger.Log(logger.Message{
		Object:  "FUNCTION",
		Action:  "COMPLETED",
		Msg:     "myFunctionName",
		Context: "{ \"status\": \"200\" }",
	})
	return nil
}
