package devserver

import (
	"context"

	"github.com/inngest/inngestctl/pkg/api"
	"github.com/inngest/inngestctl/pkg/engine"
	"github.com/rs/zerolog"
)

type Options struct {
	Port string
	Log  *zerolog.Logger
	Dir  string
}

type DevServer struct {
	API    api.API
	Engine *engine.Engine

	dir string
	log *zerolog.Logger
}

// Create and start a new dev server (API, Exectutor, State, Logger, etc.)
func NewDevServer(o Options) (DevServer, error) {
	eng, err := engine.New(engine.Options{
		Logger: o.Log,
	})
	if err != nil {
		return DevServer{}, err
	}
	api, err := api.NewAPI(api.Options{
		Port:         o.Port,
		EventHandler: eng.HandleEvent,
		Logger:       o.Log,
	})
	if err != nil {
		return DevServer{}, err
	}

	d := DevServer{
		API:    api,
		Engine: eng,

		dir: o.Dir,
		log: o.Log,
	}

	return d, err
}

func (d DevServer) Start(ctx context.Context) error {
	err := d.Engine.Load(ctx, d.dir)
	if err != nil {
		d.log.Error().Msg(err.Error())
		return err
	}
	err = d.API.Start(ctx)
	if err != nil {
		d.log.Error().Msg(err.Error())
	}
	return err
}
