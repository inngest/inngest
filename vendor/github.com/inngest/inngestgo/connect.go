package inngestgo

import (
	"context"
	"fmt"
	"net/url"

	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngestgo/connect"
	"github.com/inngest/inngestgo/internal/middleware"
	"github.com/inngest/inngestgo/internal/sdkrequest"
)

const (
	defaultMaxWorkerConcurrency = 1_000
)

type ConnectOpts struct {
	Apps []Client

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

func Connect(ctx context.Context, opts ConnectOpts) (connect.WorkerConnection, error) {
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

	apps := make([]connect.ConnectApp, len(opts.Apps))
	invokers := make(map[string]connect.FunctionInvoker, len(opts.Apps))
	for i, a := range opts.Apps {
		app, ok := a.(*apiClient)
		if !ok {
			return nil, fmt.Errorf("invalid handler passed")
		}

		appName := app.AppID()
		fns, err := createFunctionConfigs(appName, app.h.GetFunctions(), connectPlaceholder, true)
		if err != nil {
			return nil, fmt.Errorf("error creating function configs: %w", err)
		}

		apps[i] = connect.ConnectApp{
			AppName:    appName,
			Functions:  fns,
			AppVersion: app.h.GetAppVersion(),
		}

		invokers[appName] = app.h
	}

	if len(opts.Apps) < 1 {
		return nil, fmt.Errorf("must specify at least one app")
	}

	defaultClient, ok := opts.Apps[0].(*apiClient)
	if !ok {
		return nil, fmt.Errorf("invalid handler passed")
	}

	var hashedKey []byte
	var hashedFallbackKey []byte
	signingKey := defaultClient.h.GetSigningKey()
	if signingKey == "" {
		if !defaultClient.h.isDev() {
			// Signing key is only required in cloud mode.
			return nil, fmt.Errorf("signing key is required")
		}
	} else {
		var err error
		hashedKey, err = hashedSigningKey([]byte(signingKey))
		if err != nil {
			return nil, fmt.Errorf("failed to hash signing key: %w", err)
		}

		if fallbackKey := defaultClient.h.GetSigningKeyFallback(); fallbackKey != "" {
			hashedFallbackKey, err = hashedSigningKey([]byte(fallbackKey))
			if err != nil {
				return nil, fmt.Errorf("failed to hash fallback signing key: %w", err)
			}
		}
	}

	return connect.Connect(ctx, connect.Opts{
		Apps:                     apps,
		Env:                      defaultClient.Env,
		Capabilities:             capabilities,
		HashedSigningKey:         hashedKey,
		HashedSigningKeyFallback: hashedFallbackKey,
		MaxConcurrency:           concurrency,
		APIBaseUrl:               defaultClient.h.GetAPIBaseURL(),
		IsDev:                    defaultClient.h.isDev(),
		DevServerUrl:             DevServerURL(),
		InstanceID:               opts.InstanceID,
		Platform:                 Ptr(platform()),
		SDKVersion:               SDKVersion,
		SDKLanguage:              SDKLanguage,
		RewriteGatewayEndpoint:   opts.RewriteGatewayEndpoint,
	}, invokers, defaultClient.Logger)
}

func (h *handler) getServableFunctionBySlug(slug string) ServableFunction {
	h.l.RLock()
	var fn ServableFunction
	for _, f := range h.funcs {
		if f.FullyQualifiedID() == slug {
			fn = f
			break
		}
	}
	h.l.RUnlock()

	return fn
}

func (h *handler) InvokeFunction(ctx context.Context, slug string, stepId *string, request sdkrequest.Request) (any, []sdkrequest.GeneratorOpcode, error) {
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

	cImpl, ok := h.client.(*apiClient)
	if !ok {
		return nil, nil, fmt.Errorf("invalid client")
	}
	mw := middleware.NewMiddlewareManager().Add(cImpl.Middleware...)

	// Invoke function, always complete regardless of
	resp, ops, err := invoke(
		context.Background(),
		h.client,
		mw,
		fn,
		h.GetSigningKey(),
		&request,
		stepId,
	)
	return resp, ops, err
}
