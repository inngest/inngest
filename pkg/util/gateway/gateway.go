package gateway

import (
	"bytes"
	"net/http"
)

type Request struct {
	// URL is the full endpoint that we're sending the request to.  This must
	// always be provided by our SDKs.
	URL string `json:"url,omitempty"`
	// Headers represent additional headers to send in the request.
	Headers map[string]string `json:"headers,omitempty"`
	// Body indicates the raw content of the request, as a slice of JSON bytes.
	// It's expected that this comes from our SDKs directly.
	Body string `json:"body"`
	// Method is the HTTP method to use for the request.  This is almost always
	// POST for AI requests, but can be specified too.
	Method string `json:"method,omitempty"`
}

func (r Request) MarshalJSON() ([]byte, error) {
	// Do not allow this to be marshalled.  We do not want the auth creds to
	// be logged.
	return nil, nil
}

func (r Request) HTTPRequest() (*http.Request, error) {
	method := http.MethodPost
	if r.Method != "" {
		method = r.Method
	}

	// If the body is empty, we need to set it to an empty JSON object.
	req, err := http.NewRequest(method, r.URL, bytes.NewReader([]byte(r.Body)))
	if err != nil {
		return nil, err
	}

	// Overwrite any headers if custom headers are added to opts.
	for header, val := range r.Headers {
		req.Header.Add(header, val)
	}

	return req, nil
}

type Response struct {
	// URL is the full endpoint that we're sending the request to.  This must
	// always be provided by our SDKs.
	URL string `json:"url,omitempty"`
	// Headers represent additional headers to send in the request.
	Headers map[string]string `json:"headers,omitempty"` // Body indicates the raw content of the request, as a slice of JSON bytes.
	// It's expected that this comes from our SDKs directly.
	Body string `json:"body"`
	// StatusCode is the HTTP status code of the response.
	StatusCode int `json:"status_code"`
}
