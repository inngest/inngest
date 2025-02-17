package inngestgo

import (
	"context"
	"fmt"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngestgo/connect"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"net/url"
)

const (
	defaultMaxWorkerConcurrency = 1_000
)

type ConnectOpts struct {
	// InstanceID represents a stable identifier to be used for identifying connected SDKs.
	// This can be a hostname or other identifier that remains stable across restarts.
	//
	// If nil, this defaults to the current machine's hostname.
	InstanceID *string

	RewriteGatewayEndpoint func(endpoint url.URL) (url.URL, error)

	// MaxConcurrency defines the maximum number of requests the worker can process at once.
	// This affects goroutines available to handle connnect workloads, as well as flow control.
	// Defaults to 1000.
	MaxConcurrency int
}

func (h *handler) Connect(ctx context.Context, opts ConnectOpts) (connect.WorkerConnection, error) {
	concurrency := opts.MaxConcurrency
	if concurrency < 1 {
		concurrency = defaultMaxWorkerConcurrency
	}

	connectPlaceholder := url.URL{
		Scheme: "ws",
		Host:   "connect",
	}

	if opts.InstanceID == nil {
		return nil, fmt.Errorf("missing required Instance ID")
	}

	fns, err := createFunctionConfigs(h.appName, h.funcs, connectPlaceholder, true)
	if err != nil {
		return nil, fmt.Errorf("error creating function configs: %w", err)
	}

	signingKey := h.GetSigningKey()
	if signingKey == "" {
		return nil, fmt.Errorf("signing key is required")
	}

	hashedKey, err := hashedSigningKey([]byte(signingKey))
	if err != nil {
		return nil, fmt.Errorf("failed to hash signing key: %w", err)
	}

	var hashedFallbackKey []byte
	{
		if fallbackKey := h.GetSigningKeyFallback(); fallbackKey != "" {
			hashedFallbackKey, err = hashedSigningKey([]byte(fallbackKey))
			if err != nil {
				return nil, fmt.Errorf("failed to hash fallback signing key: %w", err)
			}
		}
	}

	return connect.Connect(ctx, connect.Opts{
		AppName:                  h.appName,
		Env:                      h.Env,
		Functions:                fns,
		Capabilities:             capabilities,
		HashedSigningKey:         hashedKey,
		HashedSigningKeyFallback: hashedFallbackKey,
		MaxConcurrency:           concurrency,
		APIBaseUrl:               h.GetAPIBaseURL(),
		IsDev:                    h.isDev(),
		DevServerUrl:             DevServerURL(),
		InstanceID:               opts.InstanceID,
		AppVersion:               h.AppVersion,
		Platform:                 Ptr(platform()),
		SDKVersion:               SDKVersion,
		SDKLanguage:              SDKLanguage,
		RewriteGatewayEndpoint:   opts.RewriteGatewayEndpoint,
	}, h, h.Logger)
}

func (h *handler) getServableFunctionBySlug(slug string) ServableFunction {
	h.l.RLock()
	var fn ServableFunction
	for _, f := range h.funcs {
		if f.Slug(h.appName) == slug {
			fn = f
			break
		}
	}
	h.l.RUnlock()

	return fn
}

func (h *handler) InvokeFunction(ctx context.Context, slug string, stepId *string, request sdkrequest.Request) (any, []state.GeneratorOpcode, error) {
	fn := h.getServableFunctionBySlug(slug)

	if fn == nil {
		// XXX: This is a 500 within the JS SDK.  We should probably change
		// the JS SDK's status code to 410.  404 indicates that the overall
		// API for serving Inngest isn't found.
		return nil, nil, publicerr.Error{
			Message: fmt.Sprintf("function not found: %s", slug),
			Status:  410,
		}
	}

	// Invoke function, always complete regardless of
	resp, ops, err := invoke(context.Background(), fn, &request, stepId)

	return resp, ops, err
}
