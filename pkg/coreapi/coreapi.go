package coreapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coreapi/apiutil"
	"github.com/inngest/inngest/pkg/coreapi/generated"
	"github.com/inngest/inngest/pkg/coreapi/graph/resolvers"
	"github.com/inngest/inngest/pkg/coredata"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
)

type Options struct {
	Data cqrs.Manager

	Config        config.Config
	Logger        *zerolog.Logger
	APIReadWriter coredata.APIReadWriter
	Runner        runner.Runner
	Tracker       *runner.Tracker
	State         state.Manager
}

func NewCoreApi(o Options) (*CoreAPI, error) {
	logger := o.Logger.With().Str("caller", "coreapi").Logger()

	a := &CoreAPI{
		data:    o.Data,
		config:  o.Config,
		log:     &logger,
		Router:  chi.NewMux(),
		runner:  o.Runner,
		tracker: o.Tracker,
		state:   o.State,
	}

	cors := cors.New(cors.Options{
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	})
	a.Use(cors.Handler)

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers.Resolver{
		Data:          o.Data,
		APIReadWriter: o.APIReadWriter,
		Runner:        o.Runner,
	}}))

	// TODO - Add option for enabling GraphQL Playground
	a.Handle("/", playground.Handler("GraphQL playground", "/v0/gql"))
	a.Handle("/gql", srv)

	// V0 APIs
	a.Get("/events/{eventID}/runs", a.EventRuns)
	a.Delete("/runs/{runID}", a.CancelRun)

	return a, nil
}

type CoreAPI struct {
	chi.Router
	data    cqrs.Manager
	config  config.Config
	log     *zerolog.Logger
	server  *http.Server
	state   state.Manager
	runner  runner.Runner
	tracker *runner.Tracker
}

func (a *CoreAPI) Start(ctx context.Context) error {
	a.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.config.CoreAPI.Addr, a.config.CoreAPI.Port),
		Handler: a.Router,
	}

	a.log.Info().Str("addr", a.server.Addr).Msg("starting server")
	return a.server.ListenAndServe()
}

func (a CoreAPI) Stop(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

func (a CoreAPI) EventRuns(w http.ResponseWriter, r *http.Request) {
	if a.tracker == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  500,
			Message: "No tracker",
		})
		return
	}

	// NOTE: In development this does no authentication.  This must check API keys
	// in self-hosted and production environments.
	ctx := r.Context()
	eventID := chi.URLParam(r, "eventID")
	if eventID == "" {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  400,
			Message: "No event ID found",
		})
		return
	}

	runs, err := a.tracker.Runs(ctx, eventID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  500,
			Message: "Unable to load function runs from event ID",
			Err:     err,
		})
		return
	}

	if runs == nil {
		runs = []ulid.ULID{}
	}

	_ = json.NewEncoder(w).Encode(runs)
}

// CancelRun is used to cancel a function run via an API callo.
func (a CoreAPI) CancelRun(w http.ResponseWriter, r *http.Request) {
	// NOTE: In development this does no authentication.  This must check API keys
	// in self-hosted and production environments.
	ctx := r.Context()
	var runID *ulid.ULID
	if id := chi.URLParam(r, "runID"); id != "" {
		if parsed, err := ulid.Parse(id); err == nil {
			runID = &parsed
		}
	}

	// Only check/handle invalid IDs once across all cases - no ULID, invalid ULID, etc.
	if runID == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Message: apiutil.ErrRunIDInvalid.Error(),
			Status:  400,
			Err:     apiutil.ErrRunIDInvalid,
		})
		return
	}

	logger.From(ctx).Info().
		Str("run_id", runID.String()).
		Msg("cancelling function")

	if err := apiutil.CancelRun(ctx, a.state, *runID); err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	w.WriteHeader(204)
}
