package devserver

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/inngest/clistate"
	"github.com/inngest/inngest/inngest/version"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
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

	return nil
}

func (d *devserver) Run(ctx context.Context) error {
	// Create a new API endpoint which hosts SDK-related functionality for
	// registering functions.
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
		d.ulock.Lock()
		d.urls = Autodiscover(ctx)
		d.ulock.Unlock()
		<-time.After(5 * time.Second)
	}
}

func (d *devserver) pollSDKs() error {
	d.ulock.Lock()
	defer d.ulock.Unlock()

	for _, u := range URLs {
		// Make a new PUT request to the URL, indicating that the
		// SDK should push functions to the dev server.
		// TODO
	}

	return nil
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
