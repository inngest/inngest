package api

import (
	"context"

	"github.com/inngest/inngest-cli/pkg/server"
)

func NewServer() server.Server {
	return &apiServer{}
}

type apiServer struct {
	api *API
}

func (a *apiServer) Name() string {
	return "api"
}

func (a *apiServer) Pre(ctx context.Context) error {
	var err error

	// TODO: Conenct and load ingest keys.
	a.api, err = NewAPI(Options{
		Hostname: "localhost",
		Port:     "8181",
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *apiServer) Run(ctx context.Context) error {
	return a.api.Start(ctx)
}

func (a *apiServer) Stop(ctx context.Context) error {
	// Gracefully shut down the server.
	return a.api.Stop(ctx)
}
