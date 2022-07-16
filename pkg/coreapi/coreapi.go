package coreapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/coreapi/generated"
	"github.com/inngest/inngest-cli/pkg/coreapi/graph/resolvers"
	"github.com/inngest/inngest-cli/pkg/coredata"
	"github.com/rs/zerolog"
)

type Options struct {
	Config        config.Config
	Logger        *zerolog.Logger
	APIReadWriter coredata.APIReadWriter
}

func NewCoreApi(o Options) (*CoreAPI, error) {
	logger := o.Logger.With().Str("caller", "coreapi").Logger()

	a := &CoreAPI{
		config: o.Config,
		log:    &logger,
	}

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers.Resolver{
		APIReadWriter: o.APIReadWriter,
	}}))

	// TODO - Add option for enabling GraphQL Playground
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	return a, nil
}

type CoreAPI struct {
	config config.Config
	log    *zerolog.Logger
	server *http.Server
}

func (a *CoreAPI) Start(ctx context.Context) error {
	a.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.config.CoreAPI.Addr, a.config.CoreAPI.Port),
		Handler: http.DefaultServeMux,
	}

	// Todo only show this if playground is enabled in config
	a.log.Info().Msgf("connect to http://%s/ for GraphQL playground", a.server.Addr)

	a.log.Info().Str("addr", a.server.Addr).Msg("starting server")
	return a.server.ListenAndServe()
}

func (a CoreAPI) Stop(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
