package anthropic

import (
	"net/http"
)

// ClientAdapter is an interface that defines the methods that allow use of the anthropic API with different providers.
type ClientAdapter interface {
	// Translate provider specific errors.  Responds with an error and a boolean indicating if the error has been successfully parsed.
	TranslateError(resp *http.Response, body []byte) (error, bool)
	// Prepare the request for the provider and return the full URL
	PrepareRequest(c *Client, method, urlSuffix string, body any) (string, error)
	// Set the request headers for the provider
	SetRequestHeaders(c *Client, req *http.Request) error
}
