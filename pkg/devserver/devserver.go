package devserver

import (
	"context"

	"github.com/inngest/inngestctl/pkg/api"
	"github.com/inngest/inngestctl/pkg/engine"
	"github.com/inngest/inngestctl/pkg/logger"
	"github.com/inngest/inngestctl/pkg/logger/stdoutlogger"
)

type Options struct {
	Port         string
	PrettyOutput bool
	Dir          string
}

type DevServer struct {
	Logger logger.Logger
	API    api.API
	Engine *engine.Engine
	Dir    string
}

// Create and start a new dev server (API, Exectutor, State, Logger, etc.)
func NewDevServer(o Options) (DevServer, error) {
	l := stdoutlogger.NewLogger(logger.Options{
		Pretty: o.PrettyOutput,
	})
	eng := engine.New(engine.Options{
		Logger: l,
	})
	api, err := api.NewAPI(api.Options{
		Port:         o.Port,
		EventHandler: eng.HandleEvent,
		Logger:       l,
	})
	if err != nil {
		return DevServer{}, err
	}

	d := DevServer{
		Logger: l,
		API:    api,
		Engine: eng,
		Dir:    o.Dir,
	}

	return d, err
}

func (d DevServer) Start(ctx context.Context) error {
	err := d.Engine.Load(ctx, d.Dir)
	if err != nil {
		d.Logger.Log(logger.Message{
			Object: "ERROR",
			Msg:    err.Error(),
		})
		return err
	}
	err = d.API.Start(ctx)
	if err != nil {
		d.Logger.Log(logger.Message{
			Object: "ERROR",
			Msg:    err.Error(),
		})
	}
	return err
}
