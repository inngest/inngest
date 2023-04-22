package inngestgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client represents a client used to send events to Inngest.
type Client interface {
	// Send sends the specific event to the ingest API.
	Send(context.Context, Event) error
}

// NewClient returns a concrete client initialized with the given ingest key,
// which can immediately send events to the ingest API.
func NewClient(ingestKey string, opts ...Modifier) Client {
	c := &apiClient{
		ingestKey: ingestKey,
	}
	for _, o := range opts {
		o(c)
	}

	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	return c
}

// Modifier represents an API client
type Modifier func(c Client)

// WithHTTPClient is a Modifier that allows you to set the HTTP client used
// to send events.
//
// Typically you may want to use a new HTTP client to change the default timeouts
// used during the HTTP call.
func WithHTTPClient(hc *http.Client) Modifier {
	return func(c Client) {
		if c, ok := c.(*apiClient); ok {
			c.Client = hc
		}
	}
}

// WithIngestKey is a modifier that allows you to specify the ingest key used
// to send events.
func WithIngestKey(key string) Modifier {
	return func(c Client) {
		if c, ok := c.(*apiClient); ok {
			c.ingestKey = key
		}
	}
}

// WithEndpoint is a modifier that allows you to specify the API which we send
// events to.
func WithEndpoint(url string) Modifier {
	return func(c Client) {
		if c, ok := c.(*apiClient); ok {
			c.endpoint = &url
		}
	}
}

// apiClient is a concrete implementation of Client that uses the given HTTP client
// to send events to the ingest API
type apiClient struct {
	*http.Client

	ingestKey string
	endpoint  *string
}

func (a apiClient) Send(ctx context.Context, e Event) error {
	if err := e.Validate(); err != nil {
		return fmt.Errorf("error validating event: %w", err)
	}

	byt, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error marshalling event to json: %w", err)
	}

	ep := defaultEndpoint
	if a.endpoint != nil {
		ep = *a.endpoint
	}

	url := fmt.Sprintf("%s/e/%s", ep, a.ingestKey)
	resp, err := a.Post(url, "application/json", bytes.NewBuffer(byt))
	if err != nil {
		return fmt.Errorf("error sending event request: %w", err)
	}

	// There is no body to read;  the ingest API responds with status codes representing
	// each error.  We don't necessarily care about the error behind this close.
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200,201:
		return nil
	case 400:
		return fmt.Errorf("invalid event data")
	case 401:
		return fmt.Errorf("unknown ingest key")
	case 403:
		// The ingest key has an IP or event type allow/denylist.
		return fmt.Errorf("this ingest key is not authorized to send this event")
	}

	return fmt.Errorf("unknown status code sending event: %d", resp.StatusCode)
}
