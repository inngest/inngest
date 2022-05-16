package api

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/inngest/inngestctl/pkg/event"
	"github.com/inngest/inngestctl/pkg/logger"
	"github.com/inngest/inngestctl/pkg/logger/stdoutlogger"
)

type EventHandler func(*event.Event) error

type Opts struct {
	Port         string
	EventHandler EventHandler
	PrettyOutput bool
}

const (
	// MaxSize represents the maximum size of the event payload we process,
	// currently 256KB.
	MaxSize = 256 * 1024
)

var (
	EventPathRegex = regexp.MustCompile("^/e/([a-zA-Z0-9-_]+)$")
)

func NewAPI(o Opts) error {
	l := stdoutlogger.NewLogger(logger.Options{
		Pretty: o.PrettyOutput,
	})

	api := API{
		EventHandler: o.EventHandler,
		Logger:       l,
	}

	http.HandleFunc("/", api.HealthCheck)
	http.HandleFunc("/health", api.HealthCheck)
	http.HandleFunc("/e/", api.ReceiveEvent)

	l.Log(logger.Message{
		Object: "API",
		Action: "STARTED",
		Msg:    fmt.Sprintf("Server starting on port %s", o.Port),
	})

	return http.ListenAndServe(fmt.Sprintf(":%s", o.Port), nil)
}

type API struct {
	EventHandler
	Logger logger.Logger
}

func (API) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
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
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\":\"Unable to process event payload\"}"))
		return
	}

	for _, evt := range events {
		if err := a.EventHandler(evt); err != nil {
			a.Logger.Log(logger.Message{
				Object: "EVENT",
				Action: "REJECTED",
				Msg:    fmt.Sprintf("Failed to process event: %s", evt.Name),
			})
		} else {
			a.Logger.Log(logger.Message{
				Object:  "EVENT",
				Msg:     evt.Name,
				Context: evt,
			})
		}
	}

	writeResponse(w, HTTPResponse{
		StatusCode: http.StatusOK,
		Message:    fmt.Sprintf("Received %d events", len(events)),
	})
}
