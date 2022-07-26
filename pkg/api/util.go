package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/inngest/inngest/pkg/event"
)

type apiResponse struct {
	StatusCode int    `json:"status"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
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
		return nil, err
	}

	return []*event.Event{evt}, nil
}

func (a API) writeResponse(w http.ResponseWriter, h apiResponse) {
	w.WriteHeader(h.StatusCode)

	byt, err := json.Marshal(h)
	if err != nil {
		fmt.Println("Error marshalling response:", err)
	}

	_, err = w.Write(byt)
	if err != nil {
		fmt.Println("Error writing response:", err)
	}
	_, _ = w.Write([]byte("\n"))
}
