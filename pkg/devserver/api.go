package devserver

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/tel"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/util"
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
	a.Use(headers.StaticHeadersMiddleware(headers.ServerKindDev))

	a.Get("/dev", a.Info)
	a.Post("/fn/register", a.Register)
	// This allows tests to remove apps by URL
	a.Delete("/fn/remove", a.RemoveApp)

	// Go embeds files relative to the current source, which embeds
	// all under ./static.  We remove the ./static
	// directory by using fs.Sub: https://pkg.go.dev/io/fs#Sub.
	staticFS, _ := fs.Sub(static, "static")
	a.Get("/images/*", http.FileServer(http.FS(staticFS)).ServeHTTP)
	a.Get("/assets/*", http.FileServer(http.FS(staticFS)).ServeHTTP)
	a.Get("/_next/*", http.FileServer(http.FS(staticFS)).ServeHTTP)
	a.Get("/{file}.txt", http.FileServer(http.FS(staticFS)).ServeHTTP)
	a.Get("/{file}.svg", http.FileServer(http.FS(staticFS)).ServeHTTP)
	a.Get("/{file}.jpg", http.FileServer(http.FS(staticFS)).ServeHTTP)
	a.Get("/{file}.png", http.FileServer(http.FS(staticFS)).ServeHTTP)
	// Everything else loads the UI.
	a.NotFound(a.UI)
}

func (a devapi) UI(w http.ResponseWriter, r *http.Request) {
	// If there's a file that exists within `static` for this particular route,
	// return it as a static asset.
	path := r.URL.Path
	if f, err := static.Open("static" + path); err == nil {
		if stat, err := f.Stat(); err == nil && !stat.IsDir() {
			_, _ = io.Copy(w, f)
			return
		}
	}

	// If there's a trailing slash, redirect to non-trailing slashes.
	if strings.HasSuffix(r.URL.Path, "/") && r.URL.Path != "/" {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		http.Redirect(w, r, r.URL.String(), 303)
		return
	}

	m := tel.NewMetadata(r.Context())
	tel.SendEvent(r.Context(), "cli/dev_ui.loaded", m)
	tel.SendMetadata(r.Context(), m)

	byt := parsedRoutes.serve(r.Context(), r.URL.Path)
	_, _ = w.Write(byt)
}

// Info returns information about the dev server and its registered functions.
func (a devapi) Info(w http.ResponseWriter, r *http.Request) {
	a.devserver.handlerLock.Lock()
	defer a.devserver.handlerLock.Unlock()

	all, _ := a.devserver.data.GetFunctions(r.Context())
	funcs := make([]inngest.Function, len(all))
	for n, i := range all {
		f := inngest.Function{}
		_ = json.Unmarshal([]byte(i.Config), &f)
		funcs[n] = f
	}

	ir := InfoResponse{
		Version:   version.Print(),
		StartOpts: a.devserver.opts,
		Functions: funcs,
		Handlers:  a.devserver.handlers,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	byt, _ := json.MarshalIndent(ir, "", "  ")
	_, _ = w.Write(byt)
}

// Register regsters functions served via SDKs
func (a devapi) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	a.devserver.handlerLock.Lock()
	defer a.devserver.handlerLock.Unlock()

	ctx := r.Context()
	req := &sdk.RegisterRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		logger.From(ctx).Warn().Msgf("Invalid request:\n%s", err)
		a.err(ctx, w, 400, fmt.Errorf("Invalid request: %w", err))
		return
	}

	if err := a.register(ctx, *req); err != nil {
		logger.From(ctx).Warn().Msgf("Error registering functions:\n%s", err)
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	// Re-initialize our cron manager.
	if err := a.devserver.runner.InitializeCrons(ctx); err != nil {
		logger.From(ctx).Warn().Msgf("Error initializing crons:\n%s", err)
		a.err(ctx, w, 400, err)
		return
	}

	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (a devapi) register(ctx context.Context, r sdk.RegisterRequest) (err error) {
	r.URL = util.NormalizeAppURL(r.URL)

	sum, err := r.Checksum()
	if err != nil {
		return publicerr.Wrap(err, 400, "Invalid request")
	}

	if _, err := a.devserver.data.GetAppByChecksum(ctx, sum); err == nil {
		// Already registered.
		return nil
	}

	// Attempt to get the existing app by URL, and delete it if possible.
	// We're going to recreate it below.
	//
	// We need to do this as we always create an app when entering the URL
	// via the UI.  This is a dev-server specific quirk.
	app, err := a.devserver.data.GetAppByURL(ctx, r.URL)
	if err == nil && app != nil {
		_ = a.devserver.data.DeleteApp(ctx, app.ID)
	}

	// We need a UUID to register functions with.
	appParams := cqrs.InsertAppParams{
		// Use a deterministic ID for the app in dev.
		ID:          uuid.NewSHA1(uuid.NameSpaceOID, []byte(r.URL)),
		Name:        r.AppName,
		SdkLanguage: r.SDKLanguage(),
		SdkVersion:  r.SDKVersion(),
		Framework: sql.NullString{
			String: r.Framework,
			Valid:  r.Framework != "",
		},
		Url:      r.URL,
		Checksum: sum,
	}

	tx, err := a.devserver.data.WithTx(ctx)
	if err != nil {
		return publicerr.Wrap(err, 500, "Error starting registration tx")
	}

	defer func() {
		// We want to save an app at the end, after handling each error.
		if err != nil {
			appParams.Error = sql.NullString{
				String: err.Error(),
				Valid:  true,
			}
		}
		_, _ = tx.InsertApp(ctx, appParams)
		err = tx.Commit(ctx)
		if err != nil {
			logger.From(ctx).Error().Err(err).Msg("error registering functions")
		}
	}()

	// Get a list of all functions
	existing, _ := tx.GetAppFunctions(ctx, appParams.ID)
	// And get a list of functions that we've upserted.  We'll delete all existing functions not in
	// this set.
	seen := map[uuid.UUID]struct{}{}

	// XXX (tonyhb): If we're authenticated, we can match the signing key against the workspace's
	// signing key and warn if the user has an invalid key.
	funcs, err := r.Parse(ctx)
	if err != nil && err != sdk.ErrNoFunctions {
		return publicerr.Wrap(err, 400, "At least one function is invalid")
	}

	// For each function,
	for _, fn := range funcs {
		// Create a new UUID for the function.
		fn.ID = inngest.DeterministicUUID(*fn)

		// Mark as seen.
		seen[fn.ID] = struct{}{}

		config, err := json.Marshal(fn)
		if err != nil {
			return publicerr.Wrap(err, 500, "Error marshalling function")
		}

		if _, err := tx.GetFunctionByID(ctx, fn.ID); err == nil {
			// Update the function config.
			_, err = tx.UpdateFunctionConfig(ctx, cqrs.UpdateFunctionConfigParams{
				ID:     fn.ID,
				Config: string(config),
			})
			if err != nil {
				return publicerr.Wrap(err, 500, "Error updating function config")
			}
			continue
		}

		_, err = tx.InsertFunction(ctx, cqrs.InsertFunctionParams{
			ID:        fn.ID,
			Name:      fn.Name,
			Slug:      fn.Slug,
			AppID:     appParams.ID,
			Config:    string(config),
			CreatedAt: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("Function %s is invalid: %w", fn.Slug, err)
			return publicerr.Wrap(err, 500, "Error saving function")
		}
	}

	// Remove all unseen functions.
	deletes := []uuid.UUID{}
	for _, fn := range existing {
		if _, ok := seen[fn.ID]; !ok {
			deletes = append(deletes, fn.ID)
		}
	}
	if len(deletes) == 0 {
		return nil
	}

	if err = tx.DeleteFunctionsByIDs(ctx, deletes); err != nil {
		return publicerr.Wrap(err, 500, "Error deleting removed function")
	}
	return nil
}

// RemoveApp allows users to de-register an app by its URL
func (a devapi) RemoveApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	url := r.FormValue("url")

	app, err := a.devserver.data.GetAppByURL(ctx, url)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 404, "App not found: %s", url))
		return
	}

	if err := a.devserver.data.DeleteFunctionsByAppID(ctx, app.ID); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Error deleting functions"))
		return
	}

	if err := a.devserver.data.DeleteApp(ctx, app.ID); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Error deleting app"))
		return
	}
}

func (a devapi) err(ctx context.Context, w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	logger.From(ctx).Error().Msg(err.Error())
}

type InfoResponse struct {
	// Version lists the version of the development server
	Version       string             `json:"version"`
	Authenticated bool               `json:"authed"`
	StartOpts     StartOpts          `json:"startOpts"`
	Functions     []inngest.Function `json:"functions"`
	Handlers      []SDKHandler       `json:"handlers"`
}
