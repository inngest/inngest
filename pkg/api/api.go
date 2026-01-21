package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/apiutil"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/eventstream"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

type EventHandler func(context.Context, *event.Event, *event.SeededID) (string, error)

type Options struct {
	Config config.Config

	EventHandler EventHandler
	Logger       logger.Logger

	// LocalEventKeys are the keys used to send events to the local event API
	// from an app. If this is set, only keys that match one of these values
	// will be accepted.
	LocalEventKeys []string

	// RequireKeys defines whether event and signing keys are required for the
	// server to function. If this is true and signing keys are not defined,
	// the server will still boot but core actions such as syncing, runs, and
	// ingesting events will not work.
	RequireKeys bool
}

func NewAPI(o Options) (chi.Router, error) {
	logger := o.Logger.With("caller", "api")

	api := &API{
		Router:         chi.NewMux(),
		config:         o.Config,
		handler:        o.EventHandler,
		log:            logger,
		localEventKeys: o.LocalEventKeys,
		requireKeys:    o.RequireKeys,
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
	api.Use(headers.StaticHeadersMiddleware(o.Config.GetServerKind()))

	api.Get("/health", api.HealthCheck)
	api.Post("/e/{key}", api.ReceiveEvent)
	api.Post("/invoke/{slug}", api.Invoke)

	return api, nil
}

type API struct {
	chi.Router

	config config.Config

	handler EventHandler
	log     logger.Logger

	server *http.Server

	// localEventKeys are the keys used to send events to the local event API
	// from an app. If this is set, only keys that match one of these values
	// will be accepted.
	localEventKeys []string

	// requireKeys defines whether event and signing keys are required for the
	// server to function. If this is true and signing keys are not defined,
	// the server will still boot but core actions such as syncing, runs, and
	// ingesting events will not work.
	requireKeys bool
}

func (a *API) AddRoutes() {
}

func (a *API) Start(ctx context.Context) error {
	a.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.config.EventAPI.Addr, a.config.EventAPI.Port),
		Handler: a.Router,
	}
	a.log.Info("starting server", "addr", a.server.Addr)

	lerrChan := make(chan error)
	go func() {
		lerrChan <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutting down server")
		return a.server.Shutdown(ctx)
	case err := <-lerrChan:
		return err
	}
}

func (a API) Stop(ctx context.Context) error {
	if a.server == nil {
		return nil
	}

	return a.server.Shutdown(ctx)
}

func (a API) HealthCheck(w http.ResponseWriter, r *http.Request) {
	a.log.Trace("healthcheck")
	w.Header().Add("Content-Type", "application/json")
	a.writeResponse(w, apiResponse{
		StatusCode: http.StatusOK,
		Message:    "OK",
	})
}

func (a API) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	defer r.Body.Close()

	// If self hosting and keys are not defined, error.
	if a.requireKeys && len(a.localEventKeys) == 0 {
		a.log.Error("rejecting event; event keys are required to process events securely")
		w.Header().Add("Content-Type", "application/json")
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusServiceUnavailable,
			Error:      "Event keys are required to process events securely",
		})
		return
	}

	key := chi.URLParam(r, "key")
	if key == "" {
		a.log.Error("rejecting event; event key is required")
		w.Header().Add("Content-Type", "application/json")
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusUnauthorized,
			Error:      "Event key is required",
		})

		return
	}

	if len(a.localEventKeys) > 0 {
		var found bool
		for _, k := range a.localEventKeys {
			if k == key {
				found = true
				break
			}
		}

		if !found {
			a.log.Error("rejecting event; event key not recognized")
			w.Header().Add("Content-Type", "application/json")
			a.writeResponse(w, apiResponse{
				StatusCode: http.StatusUnauthorized,
				Error:      "Event key not found",
			})
			return
		}
	}

	ctx, cancel := context.WithCancel(ctx)

	// Create a new trace that may have a link to a previous one
	ctx = itrace.UserTracer().Propagator().Extract(ctx, propagation.HeaderCarrier(r.Header))

	// Create a new channel which receives a stream of events from the incoming HTTP request
	stream := make(chan eventstream.StreamItem)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return eventstream.ParseStream(
			ctx,
			r.Body,
			stream,
			consts.AbsoluteMaxEventSize,
			r.Header.Get("Content-Type"),
		)
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

		index := 0
		for s := range stream {
			index++
			evt := event.Event{}
			if err := json.Unmarshal(s.Item, &evt); err != nil {
				return err
			}

			if evt.IsInternal() {
				err := fmt.Errorf("event name is reserved for internal use: %s", evt.Name)
				return err
			}

			// External event (i.e. doesn't have the "inngest/" prefix) data
			// must not have internal metadata since it can cause issues. For
			// example, if an invoked function's event data is forwarded into a
			// new event then it may accidentally fulfill the invocation
			delete(evt.Data, "_inngest")

			ts := time.Now()
			if evt.Timestamp == 0 {
				evt.Timestamp = ts.UnixMilli()
			}
			if evt.User == nil {
				evt.User = map[string]any{}
			}

			if err := evt.Validate(ctx); err != nil {
				return err
			}

			ctx, span := itrace.UserTracer().Provider().
				Tracer(consts.OtelScopeEvent).
				Start(ctx, consts.OtelSpanEvent,
					trace.WithTimestamp(ts),
					trace.WithNewRoot(),
					trace.WithLinks(trace.LinkFromContext(ctx)),
				)
			defer span.End()

			seed := event.SeededIDFromString(
				r.Header.Get(headers.HeaderEventIDSeed),
				index,
			)
			id, err := a.handler(ctx, &evt, seed)
			if err != nil {
				a.log.Error("error handling event", "error", err, "event", evt.Name)
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
			Error:  err.Error(),
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
	slug, err := url.QueryUnescape(slug)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Unable to decode function slug"))
		return
	}

	rawEvt := event.Event{}
	if err := json.NewDecoder(r.Body).Decode(&rawEvt); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Unable to read post data request"))
		return
	}

	newInvOpts := event.NewInvocationEventOpts{
		Event: rawEvt,
		FnID:  slug,
	}
	evt := event.NewInvocationEvent(newInvOpts)

	seed := event.SeededIDFromString(
		r.Header.Get(headers.HeaderEventIDSeed),
		0,
	)
	evtID, err := a.handler(r.Context(), &evt.Event, seed)
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
