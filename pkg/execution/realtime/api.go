package realtime

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
)

type APIOpts struct {
	JWTSecret []byte
	// Broadcaster allows connections to subscribe to topics, picking up events from
	// the system and forwarding them on.
	Broadcaster Broadcaster
	// AuthMiddleware authenticates the incoming API request.
	AuthMiddleware func(http.Handler) http.Handler
	// AuthFinder authenticates the given request, returning the env and account IDs.
	AuthFinder apiv1.AuthFinder
}

func NewAPI(o APIOpts) http.Handler {
	if o.AuthFinder == nil {
		o.AuthFinder = apiv1.NilAuthFinder
	}

	// Create the HTTP implementation, which wraps the handler.  We do ths to code
	// share and split the HTTP concerns from the actual logic, eg. to share to GQL.
	impl := &api{
		Router: chi.NewRouter(),
		opts:   o,
	}

	impl.setup()

	return impl
}

type api struct {
	chi.Router

	opts APIOpts
}

func (a *api) setup() {
	a.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)

		// NOTE: We always use the realtime auth middleware which wraps the standard
		// auth middleware with JWT validation
		r.Use(realtimeAuthMW(a.opts.JWTSecret, a.opts.AuthMiddleware))

		r.Get("/realtime/connect", a.GetWebsocketUpgrade)
		r.Post("/realtime/token", a.PostCreateJWT)
	})
}

func (a *api) PostCreateJWT(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("content-type", "application/json")

	auth, err := a.opts.AuthFinder(r.Context())
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 401, "Not authenticated"))
		return
	}

	// We expect the user to post a list of topics that they're interested in.
	topics := []Topic{}
	if err := json.NewDecoder(r.Body).Decode(&topics); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid request: must provide a list of topics"))
		return
	}

	// Set the env ID from the authentication context.
	for n, _ := range topics {
		topics[n].EnvID = auth.WorkspaceID()
	}

	jwt, err := NewJWT(
		r.Context(),
		a.opts.JWTSecret,
		auth.AccountID(),
		auth.WorkspaceID(),
		topics,
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Error creating JWT.  Please try again"))
		return
	}

	w.WriteHeader(201)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"jwt": jwt,
	})
}

func (a *api) GetWebsocketUpgrade(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// NOTE: Here we always use the realtime auth finder, which attempts to auth
	// realtime connections via single-use JWTs, falling back to other auth methods
	// as necessary.
	auth, err := realtimeAuth(ctx, a.opts.AuthFinder)
	if err != nil {
		w.Header().Add("content-type", "application/json")
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 401, "Not authenticated"))
		return
	}

	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		w.Header().Add("content-type", "application/json")
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Error creating websocket connection"))
		return
	}

	sub, err := NewWebsocketSubscription(
		ctx,
		a.opts.Broadcaster,
		auth.AccountID(),
		auth.WorkspaceID(),
		a.opts.JWTSecret,
		ws,
		nil,
	)
	if err != nil {
		ws.Close(websocket.StatusAbnormalClosure, "error subscribing to topics")
		return
	}

	// Handle reading of additional messages such as subscription requests from the WS
	pollCtx := context.Background()
	if err := sub.Poll(pollCtx); err != nil {
		logger.StdlibLogger(ctx).Warn(
			"error reading from rt ws conn",
			"error", err,
		)
	}

	_ = ws.CloseNow()
}
