package inngestgo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/inngest/inngestgo/internal/middleware"
)

const (
	defaultEndpoint = "https://inn.gs"
)

// Client represents a client used to send events to Inngest.
type Client interface {
	AppID() string

	// Send sends the specific event to the ingest API.
	Send(ctx context.Context, evt any) (string, error)
	// Send sends a batch of events to the ingest API.
	SendMany(ctx context.Context, evt []any) ([]string, error)

	Serve() http.Handler
	ServeWithOpts(opts ServeOpts) http.Handler
	SetOptions(opts ClientOpts) error
	SetURL(u *url.URL)
}

type ClientOpts struct {
	AppID string

	// HTTPClient is the HTTP client used to send events.
	HTTPClient *http.Client
	// EventKey is your Inngest event key for sending events.  This defaults to the
	// `INNGEST_EVENT_KEY` environment variable if nil.
	EventKey *string
	// EventURL is the URL of the event API to send events to.  This defaults to
	// https://inn.gs if nil.
	EventURL *string
	// Env is the branch environment to deploy to.  If nil, this uses
	// os.Getenv("INNGEST_ENV").  This only deploys to branches if the
	// signing key is a branch signing key.
	Env *string

	// Logger is the structured logger to use from Go's builtin structured
	// logging package.
	Logger *slog.Logger

	// SigningKey is the signing key for your app.  If nil, this defaults
	// to os.Getenv("INNGEST_SIGNING_KEY").
	SigningKey *string

	// SigningKeyFallback is the fallback signing key for your app. If nil, this
	// defaults to os.Getenv("INNGEST_SIGNING_KEY_FALLBACK").
	SigningKeyFallback *string

	// APIOrigin is the specified host to be used to make API calls
	APIBaseURL *string

	// EventAPIOrigin is the specified host to be used to send events to
	EventAPIBaseURL *string

	// RegisterURL is the URL to use when registering functions.  If nil
	// this defaults to Inngest's API.
	//
	// This only needs to be set when self hosting.
	RegisterURL *string

	// AppVersion supplies an application version identifier. This should change
	// whenever code within one of your Inngest function or any dependency thereof changes.
	AppVersion *string

	// MaxBodySize is the max body size to read for incoming invoke requests
	MaxBodySize int

	// URL that the function is served at.  If not supplied this is taken from
	// the incoming request's data.
	URL *url.URL

	// UseStreaming enables streaming - continued writes to the HTTP writer.  This
	// differs from true streaming in that we don't support server-sent events.
	UseStreaming bool

	// AllowInBandSync allows in-band syncs to occur. If nil, in-band syncs are
	// disallowed.
	AllowInBandSync *bool

	// Dev is whether to use the Dev Server.
	Dev *bool

	// Middleware is a list of middleware to apply to the client.
	Middleware []func() middleware.Middleware
}

func (c ClientOpts) validate() error {
	if c.AppID == "" {
		return errors.New("app id is required")
	}
	return nil
}

// NewClient returns a concrete client initialized with the given ingest key,
// which can immediately send events to the ingest API.
func NewClient(opts ClientOpts) (Client, error) {
	err := opts.validate()
	if err != nil {
		return nil, err
	}

	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	// Add the default log middleware as the first middleware.
	mw := []func() middleware.Middleware{middleware.LogMiddleware(opts.Logger)}
	opts.Middleware = append(mw, opts.Middleware...)

	c := &apiClient{
		ClientOpts: opts,
	}
	c.h = newHandler(c, clientOptsToHandlerOpts(opts))

	if c.ClientOpts.HTTPClient == nil {
		c.ClientOpts.HTTPClient = http.DefaultClient
	}

	return c, nil
}

func clientOptsToHandlerOpts(opts ClientOpts) handlerOpts {
	return handlerOpts{
		Logger:             opts.Logger,
		SigningKey:         opts.SigningKey,
		SigningKeyFallback: opts.SigningKeyFallback,
		APIBaseURL:         opts.APIBaseURL,
		EventAPIBaseURL:    opts.EventAPIBaseURL,
		Env:                opts.Env,
		RegisterURL:        opts.RegisterURL,
		AppVersion:         opts.AppVersion,
		MaxBodySize:        opts.MaxBodySize,
		URL:                opts.URL,
		UseStreaming:       opts.UseStreaming,
		AllowInBandSync:    opts.AllowInBandSync,
		Dev:                opts.Dev,
	}
}

// apiClient is a concrete implementation of Client that uses the given HTTP client
// to send events to the ingest API
type apiClient struct {
	ClientOpts
	h *handler
}

func (a apiClient) AppID() string {
	return a.ClientOpts.AppID
}

func (a apiClient) GetEnv() string {
	if a.Env == nil {
		return os.Getenv("INNGEST_ENV")
	}
	return *a.Env
}

func (a apiClient) GetEventKey() string {
	if a.EventKey != nil {
		return *a.EventKey
	}

	envVar := os.Getenv("INNGEST_EVENT_KEY")
	if envVar != "" {
		return envVar
	}

	if IsDev() {
		return "NO_EVENT_KEY_SET"
	}

	return ""
}

type ServeOpts struct {
	// Origin is the host to used for HTTP base function invoking.
	// It's used to specify the host were the functions are hosted on sync.
	// e.g. https://example.com
	Origin *string

	// Path is the path to use for HTTP base function invoking
	// It's used to specify the path were the functions are hosted on sync.
	// e.g. /api/inngest
	Path *string
}

func (a apiClient) Serve() http.Handler {
	return a.ServeWithOpts(ServeOpts{})
}

func (a apiClient) ServeWithOpts(opts ServeOpts) http.Handler {
	a.h.handlerOpts.ServeOrigin = opts.Origin
	a.h.handlerOpts.ServePath = opts.Path
	return a.h
}

func (a *apiClient) SetOptions(opts ClientOpts) error {
	err := opts.validate()
	if err != nil {
		return err
	}

	a.ClientOpts = opts
	a.h.SetOptions(clientOptsToHandlerOpts(opts))
	return nil
}

func (a *apiClient) SetURL(u *url.URL) {
	a.ClientOpts.URL = u
	a.h.SetOptions(clientOptsToHandlerOpts(a.ClientOpts))
}

type validatable interface {
	Validate() error
}

func (a apiClient) Send(ctx context.Context, e any) (string, error) {
	res, err := a.SendMany(ctx, []any{e})
	if err != nil || len(res) == 0 {
		return "", err
	}
	return res[0], nil
}

func (a apiClient) SendMany(ctx context.Context, e []any) ([]string, error) {
	for _, e := range e {
		if v, ok := e.(validatable); ok {
			if err := v.Validate(); err != nil {
				return nil, fmt.Errorf("error validating event: %w", err)
			}
		}
	}

	byt, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("error marshalling event to json: %w", err)
	}

	ep := defaultEndpoint
	if IsDev() {
		ep = DevServerURL()
	}
	if a.EventURL != nil {
		ep = *a.EventURL
	}

	url := fmt.Sprintf("%s/e/%s", ep, a.GetEventKey())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(byt))
	if err != nil {
		return nil, fmt.Errorf("error creating event request: %w", err)
	}
	SetBasicRequestHeaders(req)

	if a.GetEnv() != "" {
		req.Header.Add(HeaderKeyEnv, a.GetEnv())
	}

	resp, err := a.HTTPClient.Post(url, "application/json", bytes.NewBuffer(byt))
	if err != nil {
		return nil, fmt.Errorf("error sending event request: %w", err)
	}

	// There is no body to read;  the ingest API responds with status codes representing
	// each error.  We don't necessarily care about the error behind this close.
	defer resp.Body.Close()

	var respBody eventAPIResponse
	_ = json.NewDecoder(resp.Body).Decode(&respBody)

	switch resp.StatusCode {
	case 200, 201:
		return respBody.IDs, nil
	case 400:
		var msg string
		if respBody.Error != "" {
			msg = respBody.Error
		} else {
			msg = "unknown error"
		}

		// E.g. the event is invalid.
		return nil, fmt.Errorf("bad request: %s", msg)
	case 401:
		var msg string
		if respBody.Error != "" {
			msg = respBody.Error
		} else {
			msg = "unknown error"
		}

		// E.g. the event key is invalid.
		return nil, fmt.Errorf("unauthorized: %s", msg)
	case 403:
		var msg string
		if respBody.Error != "" {
			msg = respBody.Error
		} else {
			msg = "unknown error"
		}

		// E.g. the ingest key has an IP or event type allow/denylist.
		return nil, fmt.Errorf("forbidden: %s", msg)
	}

	return nil, fmt.Errorf("unknown status code sending event: %d", resp.StatusCode)
}

// eventAPIResponse is the API response sent when responding to incoming events.
type eventAPIResponse struct {
	IDs    []string `json:"ids"`
	Status int      `json:"status"`
	Error  string   `json:"error,omitempty"`
}
