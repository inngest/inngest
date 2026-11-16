package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Mount struct {
	At      string
	Router  chi.Router
	Handler http.Handler
}

type APIServiceOptions struct {
	Config config.Config
	Mounts []Mount

	// LocalEventKeys are the keys used to send events to the local event API
	// from an app. If this is set, only keys that match one of these values
	// will be accepted.
	LocalEventKeys []string

	// requireKeys defines whether event and signing keys are required for the
	// server to function. If this is true and signing keys are not defined,
	// the server will still boot but core actions such as syncing, runs, and
	// ingesting events will not work.
	RequireKeys bool

	Logger logger.Logger
}

// NewServer creates the HTTP server of the event API, with every mount in
// opts on the same router.  events that the server receives are published to
// the event stream of opts.Config.
func NewServer(ctx context.Context, opts APIServiceOptions) (*Server, error) {
	s := &Server{
		config: opts.Config,
		log:    opts.Logger,
	}

	api, err := NewAPI(Options{
		Config:         opts.Config,
		Logger:         opts.Logger,
		EventHandler:   s.handleEvent,
		LocalEventKeys: opts.LocalEventKeys,
		RequireKeys:    opts.RequireKeys,
	})
	if err != nil {
		return nil, err
	}
	s.api = api.(*API)

	for _, m := range opts.Mounts {
		if m.Handler != nil {
			api.Mount(m.At, m.Handler)
			continue
		}
		api.Mount(m.At, m.Router)
	}

	s.publisher, err = pubsub.NewPublisher(ctx, opts.Config.EventStream.Service)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Server is the HTTP server of the event API.
type Server struct {
	config    config.Config
	api       *API
	publisher pubsub.Publisher
	log       logger.Logger
}

// Run serves HTTP until ctx is done.
func (a *Server) Run(ctx context.Context) error {
	err := a.api.Start(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *Server) handleEvent(
	ctx context.Context,
	e *event.Event,
	seed *event.SeededID,
) (string, error) {
	// ctx is the request context, so we need to re-add
	// the caller here.
	l := a.log.With("caller", "api")
	ctx = logger.WithStdlib(ctx, l)
	span := trace.SpanFromContext(ctx)

	l.Debug("handling event", "event", e.Name)

	trackedEvent := event.NewBaseTrackedEvent(
		*e,
		seed,
	)

	byt, err := json.Marshal(trackedEvent)
	if err != nil {
		l.Error("error unmarshalling event as JSON", "error", err)
		span.SetStatus(codes.Error, "error parsing event as JSON")
		return "", err
	}

	l.Info("publishing event",
		"event_name", trackedEvent.GetEvent().Name,
		"internal_id", trackedEvent.GetInternalID().String(),
		"external_id", trackedEvent.GetEvent().ID,
		"event", trackedEvent.GetEvent(),
	)

	carrier := itrace.NewTraceCarrier()
	itrace.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

	err = a.publisher.Publish(
		ctx,
		a.config.EventStream.Service.TopicName(),
		pubsub.Message{
			Name:      event.EventReceivedName,
			Data:      string(byt),
			Timestamp: time.Now(),
			Metadata: map[string]any{
				consts.OtelPropagationKey: carrier,
			},
		},
	)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return trackedEvent.GetInternalID().String(), err
}

// Stop waits for the requests that are in progress.  call it after Run
// returns.
func (a *Server) Stop(ctx context.Context) error {
	return a.api.Stop(ctx)
}
