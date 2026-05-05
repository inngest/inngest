package apiresult

import "time"

// APIResult represents the final result of a sync-based API function call.
//
// For SSE/streaming responses, the JS SDK cannot re-read the body and instead
// posts a placeholder ("null"); the actual response bytes go directly to the
// original HTTP caller.
type APIResult struct {
	// StatusCode represents the status code for the API result
	StatusCode int `json:"status"`
	// Headers represents any response headers sent in the server response
	Headers map[string]string `json:"headers,omitempty"`
	// Body represents the API response.  This may be empty by default.  It is
	// only captured when you manually specify that you want to track the
	// result.
	Body string `json:"body,omitempty"`
	// Duration represents the overall time that it took for the API to execute.
	Duration time.Duration `json:"duration"`
}
