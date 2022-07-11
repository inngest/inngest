package coreapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/logger"
	"github.com/inngest/inngest-cli/pkg/service"
)

func NewService(c config.Config) service.Service {
	return &coreApiServer{
		config: c,
	}
}

type coreApiServer struct {
	config config.Config
	api    *CoreAPI
}

func (a *coreApiServer) Name() string {
	return "coreapi"
}

func (a *coreApiServer) Pre(ctx context.Context) error {
	// TODO - Connect to coredata database
	// TODO - Configure API with correct ports, etc., set up routes
	var err error

	a.api, err = NewCoreApi(Options{
		Config: a.config,
		Logger: logger.From(ctx),
	})

	if err != nil {
		return err
	}
	return nil
}

func (a *coreApiServer) Run(ctx context.Context) error {
	// TODO - Start API server
	err := a.api.Start(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func (a *coreApiServer) Stop(ctx context.Context) error {
	// TODO - Gracefully shut down server, remove connection to coredata database
	return a.api.Stop(ctx)
}
