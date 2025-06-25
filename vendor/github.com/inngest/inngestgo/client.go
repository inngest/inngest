package inngestgo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/inngest/inngestgo/internal/middleware"
)

const (
	defaultEndpoint = "https://inn.gs"
	retryAttempts   = 5
	retryBaseDelay  = 100 * time.Millisecond
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
	//
	// Deprecated: Use EventAPIBaseURL instead.
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

	if opts.EventURL != nil {
		opts.EventAPIBaseURL = opts.EventURL
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

	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
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

	if a.IsDev() {
		return "NO_EVENT_KEY_SET"
	}

	return ""
}

func (a apiClient) IsDev() bool {
	if a.Dev != nil {
		return *a.Dev
	}
	return IsDev()
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
	a.h.ServeOrigin = opts.Origin
	a.h.ServePath = opts.Path
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
	a.URL = u
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

func (a apiClient) SendMany(ctx context.Context, e []any) (ids []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic sending events: %v", r)
		}
	}()

	for _, e := range e {
		if v, ok := e.(validatable); ok {
			if err := v.Validate(); err != nil {
				return nil, fmt.Errorf("error validating event: %w", err)
			}
		}
	}

	seed, err := seed()
	if err != nil {
		return nil, err
	}

	byt, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("error marshalling event to json: %w", err)
	}

	var (
		resp    *http.Response
		respErr error
	)
	for attempt := 0; attempt < retryAttempts; attempt++ {
		req, err := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("%s/e/%s", a.eventAPIBaseURL(), a.GetEventKey()),
			bytes.NewBuffer(byt),
		)
		if err != nil {
			return nil, fmt.Errorf("error creating event request: %w", err)
		}
		SetBasicRequestHeaders(req)
		req.Header.Set(HeaderKeyEventIDSeed, seed)

		if a.GetEnv() != "" {
			req.Header.Add(HeaderKeyEnv, a.GetEnv())
		}
		resp, respErr = a.HTTPClient.Do(req)

		// Don't retry if the request was successful or if there was a 4xx
		// status code. We don't want to retry on 4xx because the request is
		// malformed and retrying will just fail again.
		if respErr == nil && resp.StatusCode < 500 {
			break
		}

		if respErr != nil && resp != nil && resp.Body != nil {
			// Close since we're gonna retry and we don't want to leak resources.
			_ = resp.Body.Close()
		}

		// Jitter between 0 and the base delay.
		jitter := time.Duration(mathrand.Float64() * float64(retryBaseDelay))

		// Exponential backoff with jitter.
		delay := retryBaseDelay*time.Duration(math.Pow(2, float64(attempt))) + jitter

		time.Sleep(delay)
	}

	if respErr != nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return nil, respErr
	}

	if resp == nil {
		// NOTE: We'd expect respErr to be non-nil and caught above in every case.  It's typically
		// impossible that the response is nil AND respErr is nil.  For safety, though, we must check
		// both cases.
		return nil, fmt.Errorf("unable to send events:  no http response")
	}

	// There is no body to read;  the ingest API responds with status codes representing
	// each error.  We don't necessarily care about the error behind this close.
	defer func() {
		_ = resp.Body.Close()
	}()

	var respBody eventAPIResponse
	_ = json.NewDecoder(resp.Body).Decode(&respBody)

	return handleEventResponse(respBody, resp.StatusCode)
}

func (a apiClient) eventAPIBaseURL() string {
	if a.EventAPIBaseURL != nil {
		return *a.EventAPIBaseURL
	}

	origin := os.Getenv("INNGEST_EVENT_API_BASE_URL")
	if origin != "" {
		return origin
	}

	origin = os.Getenv("INNGEST_BASE_URL")
	if origin != "" {
		return origin
	}

	if a.IsDev() {
		return DevServerURL()
	}

	return defaultEventAPIOrigin
}

func handleEventResponse(r eventAPIResponse, status int) ([]string, error) {
	msg := "unknown error"
	if r.Error != "" {
		msg = r.Error
	}

	switch status {
	case 200, 201:
		return r.IDs, nil
	case 400:
		// E.g. the event is invalid.
		return nil, fmt.Errorf("bad request: %s", msg)
	case 401:
		// E.g. the event key is invalid.
		return nil, fmt.Errorf("unauthorized: %s", msg)
	case 403:
		// E.g. the ingest key has an IP or event type allow/denylist.
		return nil, fmt.Errorf("forbidden: %s", msg)
	}

	return nil, fmt.Errorf("unknown status code sending event: %d", status)
}

func seed() (string, error) {
	// Create the event ID seed header value. This is used to seed a
	// deterministic event ID in the Inngest Server.
	millis := time.Now().UnixMilli()
	entropy := make([]byte, 10)
	_, err := rand.Read(entropy)
	if err != nil {
		return "", fmt.Errorf("error creating event ID seed: %w", err)
	}
	entropyBase64 := base64.StdEncoding.EncodeToString(entropy)
	return fmt.Sprintf("%d,%s", millis, entropyBase64), nil
}

// eventAPIResponse is the API response sent when responding to incoming events.
type eventAPIResponse struct {
	IDs    []string `json:"ids"`
	Status int      `json:"status"`
	Error  string   `json:"error,omitempty"`
}
