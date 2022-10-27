package devserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/inngest/version"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/sdk"
)

var (
	// signingKeyErrorLoggedCount ensures that we log the signing key message once
	// every N times, instead of spamming the console every time we poll for functions.
	signingKeyErrorCount = 0
)

type devapi struct {
	chi.Router

	// loader stores all registered functions in the dev server.
	//
	// TODO: Refactor this so that it's a part of the devserver, instead
	// of holding a reference which is a weird pattern (tonyhb)
	devserver *devserver
}

func newDevAPI(d *devserver) chi.Router {
	// Return a chi router, which lets us attach routes to a handler.
	api := &devapi{
		Router:    chi.NewMux(),
		devserver: d,
	}
	api.addRoutes()
	return api
}

func (a *devapi) addRoutes() {
	a.Use(func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			l := logger.From(r.Context()).With().Str("caller", a.devserver.Name()).Logger()
			r = r.WithContext(logger.With(r.Context(), l))
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	})

	a.Get("/", a.UI)
	a.Get("/dev", a.Info)
	a.Post("/fn/register", a.Register)
}

func (a devapi) UI(w http.ResponseWriter, r *http.Request) {
	// TODO: Respond with a UI here.
}

// Info returns information about the dev server and its registered functions.
func (a devapi) Info(w http.ResponseWriter, r *http.Request) {
	a.devserver.handlerLock.Lock()
	a.devserver.workspaceLock.RLock()

	defer a.devserver.handlerLock.Unlock()
	defer a.devserver.workspaceLock.RUnlock()

	workspaces := DevWorkspaces{}
	for _, w := range a.devserver.workspaces {
		if w.Name != "default" {
			continue
		}
		if w.Test {
			workspaces.Test = w
		} else {
			workspaces.Prod = w
		}
	}

	funcs, _ := a.devserver.loader.Functions(r.Context())
	ir := InfoResponse{
		Version:       version.Print(),
		StartOpts:     a.devserver.opts,
		Authenticated: len(a.devserver.workspaces) > 0,
		Functions:     funcs,
		Handlers:      a.devserver.handlers,
		Workspaces:    workspaces,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	byt, _ := json.MarshalIndent(ir, "", "  ")
	_, _ = w.Write(byt)
}

// Register regsters functions served via SDKs
func (a devapi) Register(w http.ResponseWriter, r *http.Request) {
	a.devserver.handlerLock.Lock()
	defer a.devserver.handlerLock.Unlock()

	ctx := r.Context()
	req := &sdk.RegisterRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		a.err(ctx, w, 400, fmt.Errorf("Invalid request: %w", err))
		return
	}

	var key string
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		key = bearer[7:]
	}
	if key == "" {
		// In development, we log a warning here.
		if signingKeyErrorCount%20 == 0 {
			logger.From(ctx).Warn().Msg("You're missing the INNGEST_SIGNING_KEY parameter when serving your functions.  This will not work in production.")
		}
		signingKeyErrorCount++
	}

	// XXX (tonyhb): If we're authenticated, we can match the signing key against the workspace's
	// signing key and warn if the user has an invalid key.
	if err := req.Validate(ctx); err != nil {
		a.err(ctx, w, 400, fmt.Errorf("At least one function is invalid:\n%w", err))
		return
	}

	// Find and update this SDK handler, if it exists.
	var h *SDKHandler
	for n, item := range a.devserver.handlers {
		if item.SDK.URL != req.URL {
			continue
		}

		// Check if the checksum exists and is the same.  If so, we can ignore
		// this request.
		if item.SDK.Hash != nil && req.Hash != nil && *item.SDK.Hash == *req.Hash {
			_, _ = w.Write([]byte(`{"ok":true, "skipped": true}`))
			return
		}

		// Remove this item from the handlers list.
		h = &item
		a.devserver.handlers = append(a.devserver.handlers[:n], a.devserver.handlers[n+1:]...)
		break
	}

	if h == nil {
		h = &SDKHandler{
			SDK:       *req,
			CreatedAt: time.Now(),
		}
	}
	// Reset function IDs;  we'll add these as we iterate through the requests.
	h.FunctionIDs = []string{}
	h.UpdatedAt = time.Now()

	// For each function, add it to our loader.
	for _, fn := range req.Functions {
		h.FunctionIDs = append(h.FunctionIDs, fn.ID)
		if err := a.devserver.loader.AddFunction(ctx, &fn); err != nil {
			a.err(ctx, w, 400, err)
			return
		}
	}

	// Re-initialize our cron manager.
	if err := a.devserver.runner.InitializeCrons(ctx); err != nil {
		a.err(ctx, w, 400, err)
		return
	}

	a.devserver.handlers = append(a.devserver.handlers, *h)
	_, _ = w.Write([]byte(`{"ok":true}`))

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
	Version       string              `json:"version"`
	Authenticated bool                `json:"authed"`
	StartOpts     StartOpts           `json:"startOpts"`
	Functions     []function.Function `json:"functions"`
	Handlers      []SDKHandler        `json:"handlers"`
	Workspaces    DevWorkspaces       `json:"workspaces"`
}

type DevWorkspaces struct {
	Prod client.Workspace `json:"prod"`
	Test client.Workspace `json:"test"`
}
