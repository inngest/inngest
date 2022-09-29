package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/service"
	"github.com/oklog/ulid/v2"
)

// NewService returns a new API service for ingesting events.  Any additional
// APIs can be mounted to this service to provide additional functionality.
//
// XXX (tonyhb): refactor this to remove extra mounts.
func NewService(c config.Config, mounts ...chi.Router) service.Service {
	return &apiServer{
		config: c,
		mounts: mounts,
	}
}

type apiServer struct {
	config    config.Config
	api       *API
	publisher pubsub.Publisher

	mounts []chi.Router
}

func (a *apiServer) Name() string {
	return "api"
}

func (a *apiServer) Pre(ctx context.Context) error {
	var err error

	api, err := NewAPI(Options{
		Config:       a.config,
		Logger:       logger.From(ctx),
		EventHandler: a.handleEvent,
	})
	if err != nil {
		return err
	}
	a.api = api.(*API)

	for _, m := range a.mounts {
		api.Mount("/", m)
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

func (a *apiServer) handleEvent(ctx context.Context, e *event.Event) error {
	// ctx is the request context, so we need to re-add
	// the caller here.
	l := logger.From(ctx).With().Str("caller", "api").Logger()
	ctx = logger.With(ctx, l)

	l.Debug().Str("event", e.Name).Msg("handling event")

	if e.ID == "" {
		// Always ensure that the event has an ID, for idempotency.
		e.ID = ulid.MustNew(ulid.Now(), rand.Reader).String()
	}

	// TODO: Move this into the API itself, once we've yanked out the dev
	// server's logic and replaced with multiple services.
	byt, err := json.Marshal(e)
	if err != nil {
		return err
	}

	l.Debug().Str("event", e.Name).Str("id", e.ID).Msg("publishing event")

	return a.publisher.Publish(
		ctx,
		a.config.EventStream.Service.TopicName(),
		pubsub.Message{
			// TODO: Move this into a const.
			Name:      "event/event.received",
			Data:      string(byt),
			Timestamp: time.Now(),
		},
	)
}

func (a *apiServer) Stop(ctx context.Context) error {
	return a.api.Stop(ctx)
}
