package v0

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/headers"
)

type GatewayMaintenanceActions interface {
	GetState() (*state.Gateway, error)
	DrainGateway() error
	ActivateGateway() error
}

type Opts struct {
	ConnectManager     state.ConnectionManager
	GroupManager       state.WorkerGroupManager
	GatewayMaintenance GatewayMaintenanceActions

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

		r.Get("/envs/{envID}/conns", a.showConnections)
		r.Get("/envs/{envID}/groups/{groupID}", a.showWorkerGroup)
	})

	a.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)
		r.Use(headers.ContentTypeJsonResponse())

		r.Get("/gateway", a.getGatewayState)
		r.Post("/gateway/drain", a.drainGateway)
		r.Post("/gateway/activate", a.activateGateway)
	})
}
