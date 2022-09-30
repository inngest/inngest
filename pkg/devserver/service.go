package devserver

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/inngest/clistate"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
)

const (
	SDKPollInterval = time.Second
)

func newService(config config.Config, rootDir string, loader *inmemory.FSLoader) *devserver {
	return &devserver{
		config:  config,
		rootDir: rootDir,
		loader:  loader,
		ulock:   &sync.Mutex{},
	}
}

// devserver is an individual service which operates development-specific APIs.
//
// Usually, you would have the event API hosted separately to any other APIs.
// In the dev server, we only want one port open:  all APIs are hosted together
// in a single router on a single port.  This simplifies the CLI args (--port) and
// SDKs, as they can test and use a single URL.
type devserver struct {
	config config.Config

	apiservice service.Service

	// urls are the URLs that host SDKs
	urls []string
	// rootDir stores the directory that the dev server is operating within.
	rootDir string
	// loader stores all registered functions in the dev server.
	loader *inmemory.FSLoader
	// workspaces stores the Inngest workspaces, if the CLI is authenticated.
	workspaces []client.Workspace

	ulock *sync.Mutex
}

func (devserver) Name() string {
	return "devserver"
}

func (d *devserver) Pre(ctx context.Context) error {
	// Create a new API endpoint which hosts SDK-related functionality for
	// registering functions.
	devAPI := newDevAPI(d.loader)
	d.apiservice = api.NewService(d.config, devAPI)

	// Fetch workspace information in the background, retrying if this
	// errors out.  This is optimistic, and it doesn't matter if it fails.
	go d.fetchWorkspaces(ctx)
	// Autodiscover the URLs that are hosting Inngest SDKs on the local machine.
	go d.autodiscover(ctx)

	return d.apiservice.Pre(ctx)
}

func (d *devserver) Run(ctx context.Context) error {
	// Start polling the SDKs as the APIs are going live.
	go d.pollSDKs(ctx)
	return d.apiservice.Run(ctx)
}

func (d devserver) Stop(ctx context.Context) error {
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

		d.workspaces, err = clistate.Client(ctx).Workspaces(ctx)
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
	for {
		if ctx.Err() != nil {
			return
		}
		d.ulock.Lock()
		d.urls = Autodiscover(ctx)
		d.ulock.Unlock()
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

		d.ulock.Lock()
		for _, u := range d.urls {
			// Make a new PUT request to the URL, indicating that the
			// SDK should push functions to the dev server.
			req, _ := http.NewRequest(http.MethodPut, u, strings.NewReader(`{"devserver":true}`))
			resp, err := hc.Do(req)
			if err != nil {
				// TODO: If the error is of type connection refused, remove the URL from the list.
				// This will be re-added when autodiscover scan re-runs.
				logger.From(ctx).Error().Err(err).Str("url", u).Msg("unable to register SDK functions")
				continue
			}
			if resp.StatusCode == 200 {
				continue
			}
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			logger.From(ctx).Error().
				Int("status", resp.StatusCode).
				Str("url", u).
				Str("response", string(body)).
				Msg("erorr registering SDK functions")
		}
		d.ulock.Unlock()

		<-time.After(SDKPollInterval)
	}
}
