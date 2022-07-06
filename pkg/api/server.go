package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/logger"
	"github.com/inngest/inngest-cli/pkg/server"
)

func NewServer(c *config.Config) server.Server {
	return &apiServer{
		config: c,
	}
}

type apiServer struct {
	config *config.Config
	api    *API
}

func (a *apiServer) Name() string {
	return "api"
}

func (a *apiServer) Pre(ctx context.Context) error {
	var err error

	a.api, err = NewAPI(Options{
		Hostname: a.config.EventAPI.Addr,
		Port:     a.config.EventAPI.Port,
		Logger:   logger.From(ctx),
	})

	if err != nil {
		return err
	}

	return nil
}

func (a *apiServer) Run(ctx context.Context) error {
	err := a.api.Start(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *apiServer) Stop(ctx context.Context) error {
	return a.api.Stop(ctx)
}
