package devserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/inngest/clistate"
	"github.com/inngest/inngest/inngest/version"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/logger"
)

// devserver is an individual service which operates development-specific APIs.
type devserver struct {
	// urls are the URLs that host SDKs
	urls []string

	// rootDir stores the directory that the dev server is operating within.
	rootDir string
	// loader stores all registered functions in the dev server.
	loader *inmemory.FSLoader
	// workspaces stores the Inngest workspaces, if the CLI is authenticated.
	workspaces []client.Workspace

	ulock sync.Mutex
}

func (devserver) Name() string {
	return "devserver"
}

func (d *devserver) Pre(ctx context.Context) error {
	// Fetch workspace information in the background, retrying if this
	// errors out.  This is optimistic, and it doesn't matter if it fails.
	go d.fetchWorkspaces(ctx)

	// Autodiscover the URLs that are hosting Inngest SDKs on the local machine.
	go d.autodiscover(ctx)
	go d.pollSDKs(ctx)

	return nil
}

func (d *devserver) Run(ctx context.Context) error {
	// TODO: Create a new API endpoint which hosts SDK-related functionality for
	// registering functions.
	<-time.After(time.Minute)
	return nil
}

func (d devserver) Stop(ctx context.Context) error {
	return nil
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

func (d *devserver) Router() chi.Router {
	// Return a chi router, which lets us attach routes to a handler.
	return nil
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
				logger.From(ctx).Error().Err(err).Str("url", u).Msg("unable to register SDK functions")
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

		<-time.After(time.Second)
	}
}

// Info returns information about the dev server and its registered functions.
func (d *devserver) Info(w http.ResponseWriter, r *http.Request) {
	ir := InfoResponse{
		Version:       version.Print(),
		Authenticated: len(d.workspaces) > 0,
	}
	_ = json.NewEncoder(w).Encode(ir)
}

type InfoResponse struct {
	// Version lists the version of the development server
	Version       string `json:"version"`
	Authenticated bool   `json:"authed"`

	// TODO
	StartOpts any
}
