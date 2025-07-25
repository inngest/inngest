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
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/api/tel"
	"github.com/inngest/inngest/pkg/config"
	connectv0 "github.com/inngest/inngest/pkg/connect/rest/v0"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/apiutil"
	"github.com/inngest/inngest/pkg/coreapi/generated"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/resolvers"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/history_reader"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
)

type Options struct {
	Data cqrs.Manager

	AuthMiddleware func(http.Handler) http.Handler
	Config         config.Config
	Logger         logger.Logger
	Runner         runner.Runner
	State          state.Manager
	Queue          queue.JobQueueReader
	EventHandler   api.EventHandler
	Executor       execution.Executor
	HistoryReader  history_reader.Reader

	// LocalSigningKey is the key used to sign events for self-hosted services.
	LocalSigningKey string

	// RequireKeys defines whether event and signing keys are required for the
	// server to function. If this is true and signing keys are not defined,
	// the server will still boot but core actions such as syncing, runs, and
	// ingesting events will not work.
	RequireKeys bool

	// DisableGraphQL controls whether GraphQL endpoints are enabled
	DisableGraphQL *bool

	ConnectOpts connectv0.Opts
}

func (o Options) isGraphQLEnabled() bool {
	// Default to true if not explicitly set to false
	return o.DisableGraphQL == nil || !*o.DisableGraphQL
}

func NewCoreApi(o Options) (*CoreAPI, error) {
	logger := o.Logger.With("caller", "coreapi")

	if o.HistoryReader == nil {
		return nil, fmt.Errorf("history reader is required")
	}

	a := &CoreAPI{
		data:   o.Data,
		config: o.Config,
		log:    logger,
		Router: chi.NewMux(),
		runner: o.Runner,
		state:  o.State,
	}

	cors := cors.New(cors.Options{
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	})
	a.Use(
		cors.Handler,
		headers.StaticHeadersMiddleware(o.Config.GetServerKind()),
		loader.Middleware(loader.LoaderParams{
			DB: o.Data,
		}),
	)

	if o.isGraphQLEnabled() {
		srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers.Resolver{
			Data:            o.Data,
			HistoryReader:   o.HistoryReader,
			Runner:          o.Runner,
			Queue:           o.Queue,
			EventHandler:    o.EventHandler,
			Executor:        o.Executor,
			ServerKind:      o.Config.GetServerKind(),
			LocalSigningKey: o.LocalSigningKey,
			RequireKeys:     o.RequireKeys,
		}}))

		// TODO - Add option for enabling GraphQL Playground
		a.Handle("/", playground.Handler("GraphQL playground", "/v0/gql"))
		a.Handle("/gql", srv)
	}

	// V0 APIs
	a.With(o.AuthMiddleware).Delete("/runs/{runID}", a.CancelRun)
	// NOTE: These are present in the 2.x and 3.x SDKs to enable large payload sizes.
	a.With(o.AuthMiddleware).Get("/runs/{runID}/batch", a.GetEventBatch)
	a.With(o.AuthMiddleware).Get("/runs/{runID}/actions", a.GetActions)
	a.With(o.AuthMiddleware).Post("/telemetry", a.TrackEvent)

	a.With(o.AuthMiddleware).Mount("/connect", connectv0.New(a, o.ConnectOpts))

	return a, nil
}

type CoreAPI struct {
	chi.Router
	data   cqrs.Manager
	config config.Config
	log    logger.Logger
	server *http.Server
	state  state.Manager
	runner runner.Runner
}

func (a *CoreAPI) Start(ctx context.Context) error {
	a.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.config.CoreAPI.Addr, a.config.CoreAPI.Port),
		Handler: a.Router,
	}

	a.log.Info("starting server", "addr", a.server.Addr)
	return a.server.ListenAndServe()
}

func (a CoreAPI) Stop(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

func (a CoreAPI) GetActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var runID *ulid.ULID
	if id := chi.URLParam(r, "runID"); id != "" {
		if parsed, err := ulid.Parse(id); err == nil {
			runID = &parsed
		}
	}
	if runID == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Message: apiutil.ErrRunIDInvalid.Error(),
			Status:  400,
			Err:     apiutil.ErrRunIDInvalid,
		})
		return
	}

	// Find this run
	state, err := a.state.Load(ctx, consts.DevServerAccountID, *runID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  410,
			Message: fmt.Sprintf("runtime state is no longer available for runID: %s", runID),
			Err:     err,
		})
		return
	}

	actions := state.Actions()
	_ = json.NewEncoder(w).Encode(actions)
}

func (a CoreAPI) TrackEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metadata := tel.NewMetadata(ctx)

	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err == nil {
		for k, v := range requestBody {
			metadata.Context[k] = v
		}
	}

	eventName, ok := requestBody["eventName"].(string)
	if ok {
		tel.SendEvent(ctx, eventName, metadata)
	} else {
		tel.SendMetadata(ctx, metadata)
	}

	w.WriteHeader(http.StatusOK)
}

func (a CoreAPI) GetEventBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var runID *ulid.ULID
	if id := chi.URLParam(r, "runID"); id != "" {
		if parsed, err := ulid.Parse(id); err == nil {
			runID = &parsed
		}
	}

	if runID == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Message: apiutil.ErrRunIDInvalid.Error(),
			Status:  400,
			Err:     apiutil.ErrRunIDInvalid,
		})
		return
	}

	// Find this run
	state, err := a.state.Load(ctx, consts.DevServerAccountID, *runID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  410,
			Message: fmt.Sprintf("runtime state is no longer available for runID: %s", runID),
			Err:     err,
		})
		return
	}

	events := state.Events()
	_ = json.NewEncoder(w).Encode(events)
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

	a.log.Info("cancelling function", "run_id", runID.String())

	if err := apiutil.CancelRun(ctx, a.state, consts.DevServerAccountID, *runID); err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	w.WriteHeader(204)
}
