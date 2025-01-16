package anthropic

import (
	"fmt"
)

// VertexAPIError provides error information returned by the Anthropic API.
type VertexAPIError struct {
	Code    int                     `json:"code"`
	Message string                  `json:"message"`
	Status  string                  `json:"status"`
	Details []VertexAPIErrorDetails `json:"details"`
}

type VertexAPIErrorDetails struct {
	Type     string `json:"@type"`
	Reason   string `json:"reason"`
	Metadata struct {
		Method  string `json:"method"`
		Service string `json:"service"`
	} `json:"metadata"`
}

type VertexAIErrorResponse struct {
	Error *VertexAPIError `json:"error,omitempty"`
}

func (e *VertexAPIError) Error() string {
	return fmt.Sprintf(
		"vertex api error code: %d, status: %s, message: %s",
		e.Code,
		e.Status,
		e.Message,
	)
}
