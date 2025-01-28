package anthropic

import (
	"errors"
	"fmt"
)

type ErrType string

const (
	// ErrTypeInvalidRequest There was an issue with the format or content of your request.
	ErrTypeInvalidRequest ErrType = "invalid_request_error"
	// ErrTypeAuthentication There's an issue with your API key.
	ErrTypeAuthentication ErrType = "authentication_error"
	// ErrTypePermission Your API key does not have permission to use the specified resource.
	ErrTypePermission ErrType = "permission_error"
	// ErrTypeNotFound The requested resource was not found.
	ErrTypeNotFound ErrType = "not_found_error"
	// ErrTypeTooLarge Request exceeds the maximum allowed number of bytes.
	ErrTypeTooLarge ErrType = "request_too_large"
	// ErrTypeRateLimit Your account has hit a rate limit.
	ErrTypeRateLimit ErrType = "rate_limit_error"
	// ErrTypeApi An unexpected error has occurred internal to Anthropic's systems.
	ErrTypeApi ErrType = "api_error"
	// ErrTypeOverloaded Anthropic's API is temporarily overloaded.
	ErrTypeOverloaded ErrType = "overloaded_error"
)

var (
	ErrSteamingNotSupportTools = errors.New("streaming is not yet supported tools")
)

// APIError provides error information returned by the Anthropic API.
type APIError struct {
	Type    ErrType `json:"type"`
	Message string  `json:"message"`
}

func (e *APIError) IsInvalidRequestErr() bool {
	return e.Type == ErrTypeInvalidRequest
}

func (e *APIError) IsAuthenticationErr() bool {
	return e.Type == ErrTypeAuthentication
}

func (e *APIError) IsPermissionErr() bool {
	return e.Type == ErrTypePermission
}

func (e *APIError) IsNotFoundErr() bool {
	return e.Type == ErrTypeNotFound
}

func (e *APIError) IsTooLargeErr() bool {
	return e.Type == ErrTypeTooLarge
}

func (e *APIError) IsRateLimitErr() bool {
	return e.Type == ErrTypeRateLimit
}

func (e *APIError) IsApiErr() bool {
	return e.Type == ErrTypeApi
}

func (e *APIError) IsOverloadedErr() bool {
	return e.Type == ErrTypeOverloaded
}

// RequestError provides information about generic request errors.
type RequestError struct {
	StatusCode int
	Err        error
	Body       []byte
}

type ErrorResponse struct {
	Type  string    `json:"type"`
	Error *APIError `json:"error,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("anthropic api error type: %s, message: %s", e.Type, e.Message)
}

func (e *RequestError) Error() string {
	return fmt.Sprintf(
		"anthropic request error status code: %d, err: %s, body: %s",
		e.StatusCode,
		e.Err,
		e.Body,
	)
}
