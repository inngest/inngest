package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coreapi/apiutil"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/eventstream"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type EventHandler func(context.Context, *event.Event) (string, error)

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

func (a API) HealthCheck(w http.ResponseWriter, r *http.Request) {
	a.log.Trace().Msg("healthcheck")
	a.writeResponse(w, apiResponse{
		StatusCode: http.StatusOK,
		Message:    "OK",
	})
}

func (a API) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	defer r.Body.Close()

	key := chi.URLParam(r, "key")
	if key == "" {
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusUnauthorized,
			Error:      "Event key is required",
		})
		return
	}

	// Create a new channel which receives a stream of events from the incoming HTTP request
	byteStream := make(chan json.RawMessage)
	eg := errgroup.Group{}
	eg.Go(func() error {
		return eventstream.ParseStream(ctx, r.Body, byteStream, a.config.EventAPI.MaxSize)
	})

	// Create a new channel which holds all event IDs as a slice.
	var (
		ids    = []string{}
		idChan = make(chan string)
	)
	eg.Go(func() error {
		for item := range idChan {
			ids = append(ids, item)
		}
		return nil
	})

	// Process those incoming events
	eg.Go(func() error {
		// TODO: Iterate through event stream and process event.
		for byt := range byteStream {
			evt := event.Event{}
			if err := json.Unmarshal(byt, &evt); err != nil {
				return err
			}
			id, err := a.handler(r.Context(), &evt)
			if err != nil {
				a.log.Error().Str("event", evt.Name).Err(err).Msg("error handling event")
				return err
			}
			idChan <- id
		}

		// Close the idChan so that we stop appending to the ID slice.
		close(idChan)
		return nil
	})

	if err := eg.Wait(); err != nil {
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(apiutil.EventAPIResponse{
			IDs:    ids,
			Status: 400,
			Error:  err,
		})
		return
	}

	w.WriteHeader(200)
	_ = json.NewEncoder(w).Encode(apiutil.EventAPIResponse{
		IDs:    ids,
		Status: 200,
	})
}
