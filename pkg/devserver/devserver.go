package devserver

import (
	"context"

	"github.com/inngest/inngest-cli/pkg/api"
	"github.com/rs/zerolog"
)

type Options struct {
	Port string
	Log  *zerolog.Logger
	Dir  string
}

type DevServer struct {
	API    api.API
	Engine *Engine

	dir string
	log *zerolog.Logger
}

// Create and start a new dev server (API, Exectutor, State, Logger, etc.)
func NewDevServer(o Options) (DevServer, error) {
	eng, err := NewEngine(o.Log)
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
	go func() {
		if err := d.Engine.Start(ctx); err != nil {
			panic("unable to start runner")
		}
	}()

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
