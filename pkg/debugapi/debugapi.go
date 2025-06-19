package debugapi

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
)

func NewDebugAPI(o Opts) (*debugAPI, error) {
	api := &debugAPI{
		Router: chi.NewRouter(),
		Opts:   o,
	}

	api.Use(middleware.AllowContentType("application/json"))
	api.Use(headers.ContentTypeJsonResponse())

	// API groupings
	api.Route("/queue", func(r chi.Router) {
		r.Get("/partitions/{id}", api.partitionByID)
	})

	return api, nil
}

type Opts struct {
	Log   logger.Logger
	Queue redis_state.QueueManager

	ShardSelector redis_state.ShardSelector
}

type debugAPI struct {
	chi.Router
	Opts
}
