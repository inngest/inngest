package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/inngest/inngestctl/pkg/event"
)

type HTTPResponse struct {
	StatusCode int
	Message    string
	Error      string
}

func parseBody(body []byte) ([]*event.Event, error) {
	body = bytes.TrimSpace(body)

	if len(body) > 0 && body[0] == '[' {
		evts := []*event.Event{}
		if err := json.Unmarshal(body, &evts); err != nil {
			// XXX: respond with error JSON.  If maxlen return a specific error.
			return nil, err
		}
		return evts, nil
	}

	evt := &event.Event{}
	if err := json.Unmarshal(body, evt); err != nil {
		// XXX: respond with error JSON.  If maxlen return a specific error.
		return nil, err
	}
	return []*event.Event{evt}, nil
}

func writeResponse(w http.ResponseWriter, h HTTPResponse) {
	w.WriteHeader(h.StatusCode)
	body := map[string]string{}
	if h.Message != "" {
		body["message"] = h.Message
	}
	if h.Error != "" {
		body["error"] = h.Error
	}
	byt, err := json.Marshal(body)
	if err != nil {
		fmt.Println("Error marshalling response:", err)
	}
	_, err = w.Write(byt)
	if err != nil {
		fmt.Println("Error writing response:", err)
	}
}

func BasicEventHandler(*event.Event) error {
	// TODO - Send to executor
	return nil
}
