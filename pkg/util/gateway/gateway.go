package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/inngest/inngest/pkg/execution/exechttp"
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
	// PublishOpts configures optional publishing to realtime.
	Publish PublishOpts `json:"publish,omitzero"`
}

// PublishOpts specifies the optional channel and topic if the response is to
// be published in realtime, using Inngest's realtime capabilities.
type PublishOpts struct {
	Channel string `json:"channel"`
	Topic   string `json:"topic"`
}

func (r Request) MarshalJSON() ([]byte, error) {
	// Do not allow this to be marshalled.  We do not want the auth creds to
	// be logged.
	return nil, nil
}

// SerializableRequest returns an exechttp.SerializableRequest type from the request, without publish opts
// filled.
func (r Request) SerializableRequest() (exechttp.SerializableRequest, error) {
	method := http.MethodPost
	if r.Method != "" {
		method = r.Method
	}

	// If the body is empty, we need to set it to an empty JSON object.
	req, err := exechttp.NewRequest(method, r.URL, json.RawMessage(r.Body))
	if err != nil {
		return exechttp.SerializableRequest{}, err
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
