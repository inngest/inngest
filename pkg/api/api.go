package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/inngest/inngestctl/pkg/event"
	"github.com/inngest/inngestctl/pkg/logger"
)

type EventHandler func(*event.Event) error

type Options struct {
	Port         string
	EventHandler EventHandler
	Logger       logger.Logger
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
	api := API{
		Port:         o.Port,
		EventHandler: o.EventHandler,
		Logger:       o.Logger,
	}

	http.HandleFunc("/", api.HealthCheck)
	http.HandleFunc("/health", api.HealthCheck)
	http.HandleFunc("/e/", api.ReceiveEvent)

	return api, nil
}

type API struct {
	Port string
	EventHandler
	Logger logger.Logger
}

func (a API) Start(ctx context.Context) error {
	a.Logger.Log(logger.Message{
		Object: "API",
		Action: "STARTED",
		Msg:    fmt.Sprintf("Server starting on port %s", a.Port),
	})
	return http.ListenAndServe(fmt.Sprintf(":%s", a.Port), nil)
}

func (API) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, HTTPResponse{
		StatusCode: http.StatusOK,
		Message:    "OK",
	})
}

func (a API) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > MaxSize {
		writeResponse(w, HTTPResponse{
			StatusCode: http.StatusRequestEntityTooLarge,
			Error:      "Payload larger than maximum allowed 256KB",
		})
		return
	}

	matches := EventPathRegex.FindStringSubmatch(r.URL.Path)
	if matches == nil || len(matches) != 2 {
		writeResponse(w, HTTPResponse{
			StatusCode: http.StatusUnauthorized,
			Error:      "API Key is required",
		})
		return
	}
	// noop on the key for now
	// key := matches[1]

	body, err := io.ReadAll(io.LimitReader(r.Body, MaxSize))
	if err != nil {
		writeResponse(w, HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Error:      "Could not read event payload",
		})
		return
	}
	defer r.Body.Close()

	events, err := parseBody(body)
	if err != nil {
		writeResponse(w, HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Error:      "Unable to process event payload",
		})
		return
	}

	for _, evt := range events {
		a.Logger.Log(logger.Message{
			Object:  "EVENT",
			Msg:     evt.Name,
			Context: evt,
		})
		if err := a.EventHandler(evt); err != nil {
			a.Logger.Log(logger.Message{
				Object: "EVENT",
				Action: "REJECTED",
				Msg:    fmt.Sprintf("Failed to process event: %s", evt.Name),
			})
		}
	}

	writeResponse(w, HTTPResponse{
		StatusCode: http.StatusOK,
		Message:    fmt.Sprintf("Received %d events", len(events)),
	})
}
