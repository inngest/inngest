package devserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/inngest/inngest/inngest/version"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
)

type devapi struct {
	chi.Router
	// loader stores all registered functions in the dev server.
	loader *inmemory.FSLoader
}

func newDevAPI(loader *inmemory.FSLoader) chi.Router {
	// Return a chi router, which lets us attach routes to a handler.
	api := &devapi{
		Router: chi.NewMux(),
		loader: loader,
	}
	api.addRoutes()
	return api
}

func (a *devapi) addRoutes() {
	a.Get("/dev", a.Info)
	a.Post("/fn/register", a.Register)
}

// Info returns information about the dev server and its registered functions.
func (a devapi) Info(w http.ResponseWriter, r *http.Request) {
	ir := InfoResponse{
		Version: version.Print(),
	}
	byt, _ := json.MarshalIndent(ir, "", "  ")
	_, _ = w.Write(byt)
}

// Register regsters functions served via SDKs
func (a devapi) Register(w http.ResponseWriter, r *http.Request) {
}

type InfoResponse struct {
	// Version lists the version of the development server
	Version       string `json:"version"`
	Authenticated bool   `json:"authed"`

	// TODO
	StartOpts any
}
