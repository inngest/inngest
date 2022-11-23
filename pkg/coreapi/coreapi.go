package coreapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coreapi/generated"
	"github.com/inngest/inngest/pkg/coreapi/graph/resolvers"
	"github.com/inngest/inngest/pkg/coredata"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/rs/zerolog"
)

type Options struct {
	Config        config.Config
	Logger        *zerolog.Logger
	APIReadWriter coredata.APIReadWriter
	Runner        runner.Runner
}

func NewCoreApi(o Options) (*CoreAPI, error) {
	logger := o.Logger.With().Str("caller", "coreapi").Logger()

	a := &CoreAPI{
		config: o.Config,
		log:    &logger,
		Router: chi.NewMux(),
	}

	cors := cors.New(cors.Options{
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	})
	a.Use(cors.Handler)

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers.Resolver{
		APIReadWriter: o.APIReadWriter,
		Runner:        o.Runner,
	}}))

	// TODO - Add option for enabling GraphQL Playground
	a.Handle("/", playground.Handler("GraphQL playground", "/gql"))
	a.Handle("/gql", srv)

	return a, nil
}

type CoreAPI struct {
	chi.Router
	config config.Config
	log    *zerolog.Logger
	server *http.Server
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
