package devserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/inngest/inngest/inngest/version"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/sdk"
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
	ctx := r.Context()
	req := &sdk.RegisterRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		a.err(ctx, w, 400, fmt.Errorf("Invalid request: %w", err.Error()))
		return
	}

	var key string
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		key = bearer[7:]
	}
	if key == "" {
		// In development, we log a warning here.
		logger.From(ctx).Warn().Msg("You're missing the INNGEST_SIGNING_KEY parameter when serving your functions.  This will not work in production.")
	}

	// XXX (tonyhb): If we're authenticated, we can match the signing key against the workspace's
	// signing key and warn if the user has an invalid key.

	if err := req.Validate(ctx); err != nil {
		a.err(ctx, w, 400, fmt.Errorf("At least one function is invalid:\n%w", err.Error()))
		return
	}

	// For each function, add it to our loader.
	for _, fn := range req.Functions {
		if err := a.loader.AddFunction(ctx, &fn); err != nil {
			a.err(ctx, w, 400, err)
			return
		}
	}

	logger.From(ctx).Info().
		Int("len", len(req.Functions)).
		Str("app", req.AppName).
		Str("url", req.URL).
		Str("sdk", req.SDK).
		Str("framework", req.Framework).
		Msg("registered functions")
}

func (a devapi) err(ctx context.Context, w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	logger.From(ctx).Error().Msg(err.Error())
}

type InfoResponse struct {
	// Version lists the version of the development server
	Version       string `json:"version"`
	Authenticated bool   `json:"authed"`

	// TODO
	StartOpts any
}
