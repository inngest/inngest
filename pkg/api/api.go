package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/apiutil"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/eventstream"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type EventHandler func(context.Context, *event.Event) (string, error)

type Options struct {
	Config config.Config

	EventHandler EventHandler
	Logger       *zerolog.Logger
}

func NewAPI(o Options) (chi.Router, error) {
	logger := o.Logger.With().Str("caller", "api").Logger()

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
	api.Use(headers.StaticHeadersMiddleware(headers.ServerKindDev))

	api.Get("/health", api.HealthCheck)
	api.Post("/e/{key}", api.ReceiveEvent)
	api.Post("/invoke/{slug}", api.Invoke)

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
	ctx, cancel := context.WithCancel(ctx)
	stream := make(chan eventstream.StreamItem)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return eventstream.ParseStream(ctx, r.Body, stream, consts.AbsoluteMaxEventSize)
	})

	// Create a new channel which holds all event IDs as a slice.
	var (
		max    int
		ids    = make([]string, consts.MaxEvents)
		idChan = make(chan struct {
			int
			string
		})
	)
	eg.Go(func() error {
		for item := range idChan {
			if max < item.int {
				max = item.int
			}
			ids[item.int] = item.string
		}
		return nil
	})

	// Process those incoming events
	eg.Go(func() error {
		// Close the idChan so that we stop appending to the ID slice.
		defer close(idChan)

		for s := range stream {
			evt := event.Event{}
			if err := json.Unmarshal(s.Item, &evt); err != nil {
				return err
			}

			if strings.HasPrefix(strings.ToLower(evt.Name), "inngest/") {
				err := fmt.Errorf("event name is reserved for internal use: %s", evt.Name)
				return err
			}

			if evt.Timestamp == 0 {
				evt.Timestamp = time.Now().UnixMilli()
			}

			id, err := a.handler(r.Context(), &evt)
			if err != nil {
				a.log.Error().Str("event", evt.Name).Err(err).Msg("error handling event")
				return err
			}
			idChan <- struct {
				int
				string
			}{s.N, id}
		}

		return nil
	})

	err := eg.Wait()
	cancel()

	if max+1 > len(ids) {
		max = len(ids) - 1
	}

	if err != nil {
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(apiutil.EventAPIResponse{
			IDs:    ids[0 : max+1],
			Status: 400,
			Error:  err,
		})
		return
	}

	w.WriteHeader(200)
	_ = json.NewEncoder(w).Encode(apiutil.EventAPIResponse{
		IDs:    ids[0 : max+1],
		Status: 200,
	})
}

// Invoke creates an event to invoke a specific function.
func (a API) Invoke(w http.ResponseWriter, r *http.Request) {
	// XXX: In OSS self hosting, check signing keys here.

	// Get the function slug from the route parameter.   This is the function
	// we'll invoke.  Any request is passed as the event data to the function.
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(400, "Function slug is required"))
		return
	}

	evt := event.Event{}
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Unable to read post data request"))
		return
	}

	if evt.Timestamp == 0 {
		evt.Timestamp = time.Now().UnixMilli()
	}
	if evt.Data == nil {
		evt.Data = map[string]interface{}{}
	}
	// Override the name and add the invoke key.
	evt.Name = consts.InvokeEventName
	evt.Data[consts.InvokeSlugKey] = slug

	evtID, err := a.handler(r.Context(), &evt)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to create invocation event: %s", err))
		return
	}

	// TODO: If await is true as a query parameter, await the data from the function.
	_ = json.NewEncoder(w).Encode(apiutil.InvokeAPIResponse{
		ID:     evtID,
		Status: 200,
	})
}
