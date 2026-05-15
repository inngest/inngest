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
	"github.com/inngest/inngest/pkg/service"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// NewService returns a new API service for ingesting events.  Any additional
// APIs can be mounted to this service to provide additional functionality.
//
// XXX (tonyhb): refactor this to remove extra mounts.

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

func NewService(opts APIServiceOptions) service.Service {
	return &apiServer{
		config:         opts.Config,
		mounts:         opts.Mounts,
		localEventKeys: opts.LocalEventKeys,
		requireKeys:    opts.RequireKeys,
		log:            opts.Logger,
	}
}

type apiServer struct {
	config    config.Config
	api       *API
	publisher pubsub.Publisher

	mounts []Mount

	// localEventKeys are the keys used to send events to the local event API
	// from an app. If this is set, only keys that match one of these values
	// will be accepted.
	localEventKeys []string

	// requireKeys defines whether event and signing keys are required for the
	// server to function. If this is true and signing keys are not defined,
	// the server will still boot but core actions such as syncing, runs, and
	// ingesting events will not work.
	requireKeys bool
	log         logger.Logger
}

func (a *apiServer) Name() string {
	return "api"
}

func (a *apiServer) Pre(ctx context.Context) error {
	var err error

	api, err := NewAPI(Options{
		Config:         a.config,
		Logger:         a.log,
		EventHandler:   a.handleEvent,
		LocalEventKeys: a.localEventKeys,
		RequireKeys:    a.requireKeys,
	})
	if err != nil {
		return err
	}
	a.api = api.(*API)

	for _, m := range a.mounts {
		if m.Handler != nil {
			api.Mount(m.At, m.Handler)
			continue
		}
		api.Mount(m.At, m.Router)
	}

	a.publisher, err = pubsub.NewPublisher(ctx, a.config.EventStream.Service)
	if err != nil {
		return err
	}

	return nil
}

func (a *apiServer) Run(ctx context.Context) error {
	err := a.api.Start(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *apiServer) handleEvent(
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

func (a *apiServer) Stop(ctx context.Context) error {
	return a.api.Stop(ctx)
}
