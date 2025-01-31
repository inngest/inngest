package v0

import (
	"context"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/headers"
)

type Opts struct {
	ConnectManager          state.ConnectionManager
	GroupManager            state.WorkerGroupManager
	ConnectResponseNotifier pubsub.ResponseNotifier

	Signer                  auth.SessionTokenSigner
	RequestAuther           RequestAuther
	ConnectGatewayRetriever ConnectGatewayRetriever
	ConnectionLimiter       ConnectionLimiter

	Dev bool
}

type RequestAuther interface {
	AuthenticateRequest(ctx context.Context, hashedSigningKey string, env string) (*auth.Response, error)
}

type ConnectionLimiter interface {
	CheckConnectionLimit(ctx context.Context, resp *auth.Response) (bool, error)
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
	RetrieveGateway(ctx context.Context, accountId uuid.UUID, envId uuid.UUID, exclude []string, hostname string) (string, *url.URL, error)
}

type connectApiRouter struct {
	chi.Router
	Opts
}

// New creates a v0 connect REST API, which exposes connection states, history, and more.
// This does not include the actual connect endpoint, nor does it include internal operations
// for rolling out the connect gateway service.
func New(r chi.Router, opts Opts) *connectApiRouter {
	api := &connectApiRouter{
		Router: r,
		Opts:   opts,
	}
	api.setup()
	return api
}

func (a *connectApiRouter) setup() {
	// These routes are testing-only
	if a.Dev {
		a.Group(func(r chi.Router) {
			r.Use(middleware.Recoverer)
			r.Use(headers.ContentTypeJsonResponse())

			r.Get("/envs/{envID}/conns", a.showConnections)
			r.Get("/envs/{envID}/groups/{groupID}", a.showWorkerGroup)
		})
	}

	// Worker API
	a.Group(func(r chi.Router) {
		r.Post("/start", a.start)
		r.Post("/flush", a.flushBuffer)
	})
}
