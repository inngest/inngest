package v0

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/headers"
	"net/url"
)

type Opts struct {
	//ConnectManager          state.ConnectionManager
	//GroupManager            state.WorkerGroupManager
	Signer                  auth.SessionTokenSigner
	RequestAuther           RequestAuther
	ConnectGatewayRetriever ConnectGatewayRetriever

	Dev bool
}

type RequestAuther interface {
	AuthenticateRequest(ctx context.Context, hashedSigningKey string) (*auth.Response, error)
}

type ConnectGatewayRetriever interface {
	// RetrieveGateway retrieves a gateway to use for a new worker connection.
	//
	// Callers can optionally pass exclude with a slice of gateway group names to ignore, in case the worker
	// attempts to reconnect to a different gateway group to avoid repeated connection failures. This may
	// be used, but is not mandatory. For example, if no other gateway groups are available, the implementation
	// may still return a gateway from an excluded group.
	//
	// On a successful request, the gateway group name and URL are returned.
	RetrieveGateway(ctx context.Context, accountId uuid.UUID, envId uuid.UUID, exclude []string) (string, *url.URL, error)
}

type router struct {
	chi.Router
	Opts
}

// New creates a v0 connect REST API, which exposes connection states, history, and more.
// This does not include the actual connect endpoint, nor does it include internal operations
// for rolling out the connect gateway service.
func New(r chi.Router, opts Opts) *router {
	api := &router{
		Router: r,
		Opts:   opts,
	}
	api.setup()
	return api
}

func (a *router) setup() {
	// Connect API
	a.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)
		r.Use(headers.ContentTypeJsonResponse())

		// TODO Implement connection history routes
		//r.Get("/envs/{envID}/conns", a.showConnections)
		//r.Get("/envs/{envID}/groups/{groupID}", a.showWorkerGroup)

	})

	// Worker API
	a.Group(func(r chi.Router) {
		r.Post("/start", a.start)
	})
}
