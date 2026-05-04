package apiresult

import (
	"encoding/json"
	"time"
)

// APIResult represents the final result of a sync-based API function call.
//
// For SSE/streaming responses, the JS SDK cannot re-read the body and instead
// posts a placeholder ("null"); the actual response bytes go directly to the
// original HTTP caller.
//
// APIResult has custom JSON (un)marshaling to work with Body as a string in
// JSON.
type APIResult struct {
	// StatusCode represents the status code for the API result
	StatusCode int `json:"status"`
	// Headers represents any response headers sent in the server response
	Headers map[string]string `json:"headers"`
	// Body represents the API response.  This may be nil by default.  It is only
	// captured when you manually specify that you want to track the result.
	Body []byte `json:"-"`
	// Duration represents the overall time that it took for the API to execute.
	Duration time.Duration `json:"-"`
}

// apiResultWire is the on-the-wire representation.
//
// Notably, Body is a string (on-the-wire form) rather than []byte (internal
// form).
type apiResultWire struct {
	StatusCode int               `json:"status"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
}

func (a *APIResult) UnmarshalJSON(data []byte) error {
	w := apiResultWire{}
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	a.StatusCode = w.StatusCode
	a.Headers = w.Headers
	a.Body = []byte(w.Body)
	return nil
}

func (a APIResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(apiResultWire{
		StatusCode: a.StatusCode,
		Headers:    a.Headers,
		Body:       string(a.Body),
	})
}
