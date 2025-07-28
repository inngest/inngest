package debugapi

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"google.golang.org/grpc"
)

var (
	errNotImplemented = fmt.Errorf("not implemented")
)

func NewDebugAPI(o Opts) service.Service {
	return &debugAPI{
		Router: chi.NewRouter(),
		Server: grpc.NewServer(),
		Opts:   o,
	}
}

type Opts struct {
	Log   logger.Logger
	Queue redis_state.QueueManager

	ShardSelector redis_state.ShardSelector
}

type debugAPI struct {
	chi.Router
	*grpc.Server
	Opts
}

func (d *debugAPI) Name() string {
	return "debug-api"
}

func (d *debugAPI) Pre(ctx context.Context) error {
	d.Use(middleware.AllowContentType("application/json"))
	d.Use(headers.ContentTypeJsonResponse())

	d.Route("/queue", func(r chi.Router) {
		r.Get("/partitions/{id}", d.partitionByID)
	})

	return nil
}

func (d *debugAPI) Run(ctx context.Context) error {
	return errNotImplemented
}

func (d *debugAPI) Stop(ctx context.Context) error {
	return errNotImplemented
}
