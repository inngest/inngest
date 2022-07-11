package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/event"
	"github.com/inngest/inngest-cli/pkg/logger"
	"github.com/inngest/inngest-cli/pkg/pubsub"
	"github.com/inngest/inngest-cli/pkg/service"
	"github.com/oklog/ulid/v2"
)

func NewService(c config.Config) service.Service {
	return &apiServer{
		config: c,
	}
}

type apiServer struct {
	config    config.Config
	api       *API
	publisher pubsub.Publisher
}

func (a *apiServer) Name() string {
	return "api"
}

func (a *apiServer) Pre(ctx context.Context) error {
	var err error

	a.api, err = NewAPI(Options{
		Config:       a.config,
		Logger:       logger.From(ctx),
		EventHandler: a.handleEvent,
	})

	if err != nil {
		return err
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

	logger.From(ctx).Debug().Str("event", e.Name).Str("id", e.ID).Msg("publishing event")

	return a.publisher.Publish(
		ctx,
		a.config.EventStream.Service.TopicName(),
		pubsub.Message{
			// TODO: Move this into a const.
			Name:      "event/event.received",
			Version:   "2022-07-01.01",
			Data:      byt,
			Timestamp: time.Now(),
		},
	)
}

func (a *apiServer) Stop(ctx context.Context) error {
	return a.api.Stop(ctx)
}
