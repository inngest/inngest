package apiutil

import (
	"fmt"
)

var (
	ErrRunIDInvalid = fmt.Errorf("The run ID specified is invalid")
)

// EventAPIResponse is the API response sent when responding to incoming events.
type EventAPIResponse struct {
	IDs    []string `json:"ids"`
	Status int      `json:"status"`
	Error  string   `json:"error,omitempty"`
}

// InvokeAPIResponse is the API response sent when responding to an invoke
// request.
type InvokeAPIResponse struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Error  error  `json:"error,omitempty"`
}
