package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/event"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type EventHandler func(context.Context, *event.Event) error

type Options struct {
	Config config.Config

	EventHandler EventHandler
	Logger       *zerolog.Logger
}

const (
	// DefaultMaxSize represents the maximum size of the event payload we process,
	// currently 256KB.
	DefaultMaxSize = 256 * 1024
)

func NewAPI(o Options) (chi.Router, error) {
	logger := o.Logger.With().Str("caller", "api").Logger()

	if o.Config.EventAPI.MaxSize == 0 {
		o.Config.EventAPI.MaxSize = DefaultMaxSize
	}

	api := &API{
		Router:  chi.NewMux(),
		config:  o.Config,
		handler: o.EventHandler,
		log:     &logger,
	}

	cors := cors.New(cors.Options{
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           60 * 60, // 1 hour
	})
	api.Use(cors.Handler)

	api.Get("/health", api.HealthCheck)
	api.Post("/e/{key}", api.ReceiveEvent)
	api.Get("/", api.UI)

	return api, nil
}

type API struct {
	chi.Router

	config config.Config

	handler EventHandler
	log     *zerolog.Logger

	server *http.Server
}

func (a *API) AddRoutes() {
}

func (a *API) Start(ctx context.Context) error {
	a.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.config.EventAPI.Addr, a.config.EventAPI.Port),
		Handler: a.Router,
	}
	a.log.Info().Str("addr", a.server.Addr).Msg("starting server")
	return a.server.ListenAndServe()
}

func (a API) Stop(ctx context.Context) error {
	if a.server == nil {
		return nil
	}

	return a.server.Shutdown(ctx)
}

func (a API) UI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "ui/index.html")
}

func (a API) HealthCheck(w http.ResponseWriter, r *http.Request) {
	a.log.Trace().Msg("healthcheck")
	a.writeResponse(w, apiResponse{
		StatusCode: http.StatusOK,
		Message:    "OK",
	})
}

func (a API) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.ContentLength > int64(a.config.EventAPI.MaxSize) {
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusRequestEntityTooLarge,
			Error:      "Payload larger than maximum allowed",
		})
		return
	}

	key := chi.URLParam(r, "key")
	if key == "" {
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusUnauthorized,
			Error:      "Event key is required",
		})
		return
	}

	// TODO: Implement key matching from core data loader.

	body, err := io.ReadAll(io.LimitReader(r.Body, int64(a.config.EventAPI.MaxSize)))
	if err != nil {
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusBadRequest,
			Error:      "Could not read event payload",
		})
		return
	}

	events, err := parseBody(body)
	if err != nil {
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusBadRequest,
			Error:      "Unable to process event payload",
		})
		return
	}

	eg := &errgroup.Group{}
	for _, evt := range events {
		copied := evt
		eg.Go(func() error {
			if err := a.handler(r.Context(), copied); err != nil {
				a.log.Error().Str("event", copied.Name).Err(err).Msg("error handling event")
				return err
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusBadRequest,
			Error:      err.Error(),
		})
	}

	a.writeResponse(w, apiResponse{
		StatusCode: http.StatusOK,
		Message:    fmt.Sprintf("Received %d events", len(events)),
	})
}
