package v0

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/headers"
)

type Opts struct {
	ConnectManager state.ConnectionManager
	GroupManager   state.WorkerGroupManager

	Dev bool
}

type router struct {
	chi.Router
	Opts
}

func New(r chi.Router, opts Opts) *router {
	api := &router{
		Router: r,
		Opts:   opts,
	}
	api.setup()
	return api
}

func (a *router) setup() {
	a.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)
		r.Use(headers.ContentTypeJsonResponse())

		r.Get("/envs/{envID}/conns", a.showConnectionsByEnv)
		r.Get("/envs/{envID}/groups/{groupID}", a.showWorkerGroup)
		r.Get("/apps/{appID}/conns", a.showConnectionsByApp)
	})
}
