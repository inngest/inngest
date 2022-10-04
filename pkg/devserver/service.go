package devserver

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/inngest/clistate"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/service"
)

const (
	SDKPollInterval = 5 * time.Second
)

func newService(opts StartOpts, loader *inmemory.FSLoader) *devserver {
	return &devserver{
		loader:        loader,
		opts:          opts,
		urls:          opts.URLs,
		urlLock:       &sync.Mutex{},
		handlerLock:   &sync.Mutex{},
		workspaceLock: &sync.RWMutex{},
	}
}

// devserver is an individual service which operates development-specific APIs.
//
// Usually, you would have the event API hosted separately to any other APIs.
// In the dev server, we only want one port open:  all APIs are hosted together
// in a single router on a single port.  This simplifies the CLI args (--port) and
// SDKs, as they can test and use a single URL.
type devserver struct {
	opts StartOpts

	apiservice service.Service

	// urls are the URLs that host SDKs
	urls    []string
	urlLock *sync.Mutex

	// loader stores all registered functions in the dev server.
	loader *inmemory.FSLoader

	// workspaces stores the Inngest workspaces, if the CLI is authenticated.
	workspaces    []client.Workspace
	workspaceLock *sync.RWMutex

	// handlers are updated by the API (d.apiservice) when registering functions.
	handlers    []SDKHandler
	handlerLock *sync.Mutex
}

func (devserver) Name() string {
	return "devserver"
}

func (d *devserver) Pre(ctx context.Context) error {
	// Create a new API endpoint which hosts SDK-related functionality for
	// registering functions.
	devAPI := newDevAPI(d)
	d.apiservice = api.NewService(d.opts.Config, devAPI)

	// Fetch workspace information in the background, retrying if this
	// errors out.  This is optimistic, and it doesn't matter if it fails.
	go d.fetchWorkspaces(ctx)

	if d.opts.Autodiscover {
		// Autodiscover the URLs that are hosting Inngest SDKs on the local machine.
		go d.autodiscover(ctx)
	}

	return d.apiservice.Pre(ctx)
}

func (d *devserver) Run(ctx context.Context) error {
	// Start polling the SDKs as the APIs are going live.
	go d.pollSDKs(ctx)

	return d.apiservice.Run(ctx)
}

func (d *devserver) Stop(ctx context.Context) error {
	return d.apiservice.Stop(ctx)
}

func (d *devserver) fetchWorkspaces(ctx context.Context) {
	var err error
	// If we're not authenticated, ensure that we poll for auth in the background.
	// This lets us fetch account-related information to share with SDKs.
	for {
		if ctx.Err() != nil {
			return
		}

		d.workspaceLock.Lock()
		d.workspaces, err = clistate.Client(ctx).Workspaces(ctx)
		d.workspaceLock.Unlock()
		if err == nil {
			return
		}
		// Poll seconds, as we may log in and share state from another CLI
		// invocation.  This prevents you from having to reboot the dev server
		// after logging in to fetch account information.
		<-time.After(30 * time.Second)
	}
}

// Autodiscover attempts to run autodiscovery while the dev server is running.
//
// This lets the dev server start and wait for the SDK server to come up at
// any point.
func (d *devserver) autodiscover(ctx context.Context) {
	logger.From(ctx).Info().Msg("autodiscovering locally hosted SDKs")
	for {
		if ctx.Err() != nil {
			return
		}
		d.urlLock.Lock()
		d.urls = Autodiscover(ctx)
		d.urlLock.Unlock()
		<-time.After(5 * time.Second)
	}
}

// pollSDKs hits each SDK's register endpoint, asking them to communicate with
// the dev server to re-register their functions.
func (d *devserver) pollSDKs(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		d.urlLock.Lock()
		for _, u := range d.urls {
			// Make a new PUT request to the URL, indicating that the
			// SDK should push functions to the dev server.
			req, _ := http.NewRequest(http.MethodPut, u, nil)
			resp, err := hc.Do(req)
			if err != nil {
				logger.From(ctx).Error().Err(err).Str("url", u).Msg("unable to connect to the SDK")
				continue
			}
			if resp.StatusCode == 200 {
				continue
			}
			// Log an error that we were unable to connect to the SDK.
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			logger.From(ctx).Error().
				Int("status", resp.StatusCode).
				Str("url", u).
				Str("response", string(body)).
				Msg("unable to connect to the SDK")
		}
		d.urlLock.Unlock()

		<-time.After(SDKPollInterval)
	}
}

// SDKHandler represents a handler that has registered with the dev server.
type SDKHandler struct {
	FunctionIDs []string            `json:"functionIDs"`
	SDK         sdk.RegisterRequest `json:"sdk"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}
