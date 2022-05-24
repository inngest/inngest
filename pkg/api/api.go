package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/inngest/inngest-cli/pkg/event"
	"github.com/rs/zerolog"
)

type EventHandler func(context.Context, *event.Event) error

type Options struct {
	Port         string
	EventHandler EventHandler
	Logger       *zerolog.Logger
}

const (
	// MaxSize represents the maximum size of the event payload we process,
	// currently 256KB.
	MaxSize = 256 * 1024
)

var (
	EventPathRegex = regexp.MustCompile("^/e/([a-zA-Z0-9-_]+)$")
)

func NewAPI(o Options) (API, error) {
	logger := o.Logger.With().Str("caller", "api").Logger()

	api := API{
		port:    o.Port,
		handler: o.EventHandler,
		log:     &logger,
	}

	http.HandleFunc("/", api.HealthCheck)
	http.HandleFunc("/health", api.HealthCheck)
	http.HandleFunc("/e/", api.ReceiveEvent)

	return api, nil
}

type API struct {
	handler EventHandler
	port    string
	log     *zerolog.Logger
}

func (a API) Start(ctx context.Context) error {
	a.log.Info().Msgf("Starting server on port %s", a.port)
	return http.ListenAndServe(fmt.Sprintf(":%s", a.port), http.DefaultServeMux)
}

func (a API) HealthCheck(w http.ResponseWriter, r *http.Request) {
	a.log.Debug().Msg("healthcheck")
	a.writeResponse(w, apiResponse{
		StatusCode: http.StatusOK,
		Message:    "OK",
	})
}

func (a API) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.ContentLength > MaxSize {
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusRequestEntityTooLarge,
			Error:      "Payload larger than maximum allowed 256KB",
		})
		return
	}

	matches := EventPathRegex.FindStringSubmatch(r.URL.Path)
	if matches == nil || len(matches) != 2 {
		a.writeResponse(w, apiResponse{
			StatusCode: http.StatusUnauthorized,
			Error:      "API Key is required",
		})
		return
	}

	// noop on the key for now
	// key := matches[1]

	body, err := io.ReadAll(io.LimitReader(r.Body, MaxSize))
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

	for _, evt := range events {
		copied := evt
		go func(e event.Event) {
			a.log.Info().Str("event", e.Name).Msg("received event")
			if err := a.handler(r.Context(), &e); err != nil {
				a.log.Error().Msg(err.Error())
			}
		}(*copied)
	}

	a.writeResponse(w, apiResponse{
		StatusCode: http.StatusOK,
		Message:    fmt.Sprintf("Received %d events", len(events)),
	})
}
