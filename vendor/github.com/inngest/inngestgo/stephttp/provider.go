package stephttp

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/inngest/inngestgo/internal/middleware"
	"github.com/inngest/inngestgo/pkg/env"
)

const (
	headerRunID     = "x-run-id"
	headerSignature = "x-inngest-signature"
)

type Provider interface {
	// ServeHTTP is the middleware that allows the Inngest handler to work.
	ServeHTTP(next http.HandlerFunc) http.HandlerFunc

	// Middleware returns stdlib-style middleware to wrap other HTTP handlers
	Middleware(next http.Handler) http.Handler

	// Wait provides a mechanism to wait for all cehckpoints to finish before shutting down.
	// Cancel the incoming context to quit polling for checkpoint progres.
	Wait(ctx context.Context) chan bool
}

// SetupOpts contains configuration for the API middleware.  Optional
// configuration is supplied via SetupOpt adapters.
type SetupOpts struct {
	// Domain is the domain for this API (e.g., "api.mycompany.com")
	Domain string
	// Optional represents optional setup options that you can confgure.
	Optional OptionalSetupOpts
}

type OptionalSetupOpts struct {
	// TrackAllEndpoints, if set to true, will track all requests to all API endpoints,
	// even if they don't use steps.
	//
	// By default, only API endpoints that use steps will be tracked.
	TrackAllEndpoints bool

	// DefaultAsyncResponse defines the default async response type.  Each function
	// can override the async repsonse type using function configuration.
	DefaultAsyncResponse AsyncResponse

	// SigningKey is the Inngest signing key for authentication.  If empty, this defaults
	// to os.Getenv("INNGEST_SIGNING_KEY").
	SigningKey string
	// SigningKeyFallback is the optional signing key fallback. If empty, this defaults
	// to os.Getenv("INNGEST_SIGNING_KEY_FALLBACK").
	SigningKeyFallback string
	// BaseURL is the URL of the Inngest API.  If empty, this:
	//
	//   1. Checks to see if INNGEST_DEV is set, indicating dev mode.  If set, we
	//      attempt to use the INNGEST_DEV env var as the base URL if set to a URL,
	//      or default to "http://127.0.0.1:8288" for dev mode.
	//   2. If INNGEST_DEV is not set, we default to the production URL:
	//      "https://api.inngest.com".
	BaseURL string
	// Middleware represents optional middleware to run before and after processing.
	Middleware []func() middleware.Middleware
}

func (o SetupOpts) signingKey() string {
	if o.Optional.SigningKey == "" {
		return os.Getenv("INNGEST_SIGNING_KEY")
	}
	return o.Optional.SigningKey
}

func (o SetupOpts) signingKeyFallback() string {
	if o.Optional.SigningKeyFallback == "" {
		return os.Getenv("INNGEST_SIGNING_KEY_FALLBACK")
	}
	return o.Optional.SigningKeyFallback
}

func (o SetupOpts) baseURL() string {
	if o.Optional.BaseURL != "" {
		return o.Optional.BaseURL
	}
	return env.APIServerURL()
}

// provider wraps HTTP handlers to provide Inngest step tooling for API functions.
// This creates a new manager which handles the associated step and request lifecycles.
type provider struct {
	opts   SetupOpts
	api    checkpointAPI
	mw     *middleware.MiddlewareManager
	logger *slog.Logger

	// inflight records the total number of in flight requests.
	inflight *atomic.Int32
}

// Setup creates a new API provider instance
func Setup(opts SetupOpts) *provider {
	// Create a middleware manager for step execution hooks
	mw := middleware.New()
	for _, m := range opts.Optional.Middleware {
		mw.Add(m)
	}

	p := &provider{
		opts:     opts,
		mw:       mw,
		inflight: &atomic.Int32{},
		logger:   slog.Default(),
	}

	p.api = NewAPIClient(p.opts.baseURL(), p.opts.signingKey(), p.opts.signingKeyFallback())

	return p
}

// Middleware returns an HTTP middleware handler that accepts an http.Handler and
// returns an http.Handler.
func (p *provider) Middleware(next http.Handler) http.Handler {
	return p.ServeHTTP(next.ServeHTTP)
}

// Handler wraps an HTTP HandlerFunc to provide Inngest step tooling directly inside of
// your APIs.
func (p *provider) ServeHTTP(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p.inflight.Add(1)
		defer func() { p.inflight.Add(-1) }()

		if err := processRequest(p, r, w, next); err != nil {
			p.logger.Error("error handling api request", "error", err)
		}
	}
}

// Wait returns a channel that is sent when all in progress checkpoints finish.
func (p *provider) Wait(ctx context.Context) chan bool {
	c := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				// Continue on.
			}

			if p.inflight.Load() == 0 {
				c <- true
				return
			}
		}
	}()
	return c
}
