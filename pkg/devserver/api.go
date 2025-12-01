package devserver

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/tel"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/sync"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/cron"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/oklog/ulid/v2"
	ptrace "go.opentelemetry.io/collector/pdata/ptrace"
)

type devapi struct {
	chi.Router

	// loader stores all registered functions in the dev server.
	//
	// TODO: Refactor this so that it's a part of the devserver, instead
	// of holding a reference which is a weird pattern (tonyhb)
	devserver *devserver
	disableUI bool
}

type DevAPIOptions struct {
	disableUI      bool
	AuthMiddleware func(http.Handler) http.Handler
}

func NewDevAPI(d *devserver, o DevAPIOptions) chi.Router {
	// Return a chi router, which lets us attach routes to a handler.
	api := &devapi{
		Router:    chi.NewMux(),
		devserver: d,
		disableUI: o.disableUI,
	}
	api.addRoutes(o.AuthMiddleware)
	return api
}

func (a *devapi) addRoutes(AuthMiddleware func(http.Handler) http.Handler) {
	a.Use(func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			l := a.devserver.log.With("caller", a.devserver.Name())
			r = r.WithContext(logger.WithStdlib(r.Context(), l))
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	})
	a.Use(headers.StaticHeadersMiddleware(a.devserver.Opts.Config.GetServerKind()))

	a.Post("/dev/traces", a.OTLPTrace) // Intentionally outside the AuthMiddleware

	a.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)

		r.Get("/dev", a.Info)

		r.Post("/fn/register", a.Register)
		// This allows tests to remove apps by URL
		r.Delete("/fn/remove", a.RemoveApp)

		// This allows tests to update step limits per function
		r.Post("/fn/step-limit", a.SetStepLimit)
		r.Delete("/fn/step-limit", a.RemoveStepLimit)
		r.Post("/fn/state-size-limit", a.SetStateSizeLimit)
		r.Delete("/fn/state-size-limit", a.RemoveStateSizeLimit)
	})

	// Only register static file serving if UI is enabled
	if !a.disableUI {
		//
		// Create filesystem rooted at static/client for Tanstack assets
		staticFS, _ := fs.Sub(static, "static/client")
		a.Get("/images/*", http.FileServer(http.FS(staticFS)).ServeHTTP)
		a.Get("/assets/*", http.FileServer(http.FS(staticFS)).ServeHTTP)
		a.Get("/{file}.txt", http.FileServer(http.FS(staticFS)).ServeHTTP)
		a.Get("/{file}.svg", http.FileServer(http.FS(staticFS)).ServeHTTP)
		a.Get("/{file}.jpg", http.FileServer(http.FS(staticFS)).ServeHTTP)
		a.Get("/{file}.png", http.FileServer(http.FS(staticFS)).ServeHTTP)
		//
		// Everything else loads the UI (SPA fallback)
		a.NotFound(a.UI)
	}

}

func (a devapi) UI(w http.ResponseWriter, r *http.Request) {
	//
	// If there's a file that exists within static/client for this route, serve it as a static asset
	path := r.URL.Path
	if f, err := static.Open("static/client" + path); err == nil {
		if stat, err := f.Stat(); err == nil && !stat.IsDir() {
			_, _ = io.Copy(w, f)
			return
		}
	}

	//
	// If there's a trailing slash, redirect to non-trailing slashes
	if strings.HasSuffix(r.URL.Path, "/") && r.URL.Path != "/" {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
		return
	}

	m := tel.NewMetadata(r.Context())
	tel.SendEvent(r.Context(), "cli/dev_ui.loaded", m)
	tel.SendMetadata(r.Context(), m)

	byt := serve(r.Context(), r.URL.Path)
	_, _ = w.Write(byt)
}

// Info returns information about the dev server and its registered functions.
func (a devapi) Info(w http.ResponseWriter, r *http.Request) {
	// Return 404 in self-hosted mode
	if a.devserver.Opts.Config.ServerKind == "cloud" {
		http.NotFound(w, r)
		return
	}

	a.devserver.handlerLock.Lock()
	defer a.devserver.handlerLock.Unlock()

	all, _ := a.devserver.Data.GetFunctions(r.Context())
	funcs := make([]inngest.Function, len(all))
	for n, i := range all {
		f := inngest.Function{}
		_ = json.Unmarshal([]byte(i.Config), &f)
		funcs[n] = f
	}

	features := map[string]bool{}
	for _, flag := range featureFlags {
		enabled, _ := strconv.ParseBool(os.Getenv(flag))
		features[flag] = enabled
	}

	//
	// inngest feature flags are a map of arbitrary key value
	// pairs such that we can use the same feature flag names as cloud
	if ffEnv := os.Getenv("INNGEST_FEATURE_FLAGS"); ffEnv != "" {
		pairs := strings.Split(ffEnv, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				valStr := strings.TrimSpace(kv[1])
				if value, err := strconv.ParseBool(valStr); err == nil {
					features[key] = value
				}
			}
		}
	}

	ir := InfoResponse{
		Version:             version.Print(),
		StartOpts:           a.devserver.Opts,
		Functions:           funcs,
		Handlers:            a.devserver.handlers,
		IsSingleNodeService: a.devserver.IsSingleNodeService(),
		IsMissingSigningKey: a.devserver.Opts.RequireKeys && !a.devserver.HasSigningKey(),
		IsMissingEventKeys:  a.devserver.Opts.RequireKeys && !a.devserver.HasEventKeys(),
		Features:            features,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	byt, _ := json.MarshalIndent(ir, "", "  ")
	_, _ = w.Write(byt)
}

// Register regsters functions served via SDKs
func (a devapi) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()

	l := a.devserver.log
	l.Debug("received register request")

	expectedServerKind := r.Header.Get(headers.HeaderKeyExpectedServerKind)
	if expectedServerKind != "" && expectedServerKind != a.devserver.Opts.Config.GetServerKind() {
		a.err(ctx, w, 400, fmt.Errorf("Expected server kind %s, got %s", a.devserver.Opts.Config.GetServerKind(), expectedServerKind))
		return
	}

	a.devserver.handlerLock.Lock()
	defer a.devserver.handlerLock.Unlock()

	req, err := sdk.FromReadCloser(r.Body, sdk.FromReadCloserOpts{})
	if err != nil {
		l.Warn("Invalid request", "error", err)
		a.err(ctx, w, 400, fmt.Errorf("Invalid request: %w", err))
		return
	}

	reply, err := a.register(ctx, req)
	if err != nil {
		l.Warn("error registering functions", "error", err)
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	resp, err := json.Marshal(reply)
	if err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	_, _ = w.Write(resp)
}

func (a devapi) register(ctx context.Context, r sdk.RegisterRequest) (*sync.Reply, error) {
	sum, err := r.Checksum()
	if err != nil {
		return nil, publicerr.Wrap(err, 400, "Invalid request")
	}

	l := a.devserver.log

	// TODO Retrieve same syncID for connect, if r.IdempotencyKey is the same
	syncID := uuid.New()

	if app, err := a.devserver.Data.GetAppByChecksum(ctx, consts.DevServerEnvID, sum); err == nil {
		if !app.Error.Valid {
			// Skip registration since the app was already successfully
			// registered.
			return &sync.Reply{
				OK:     true,
				AppID:  &app.ID,
				SyncID: &syncID,
			}, nil
		}

		// Clear app error.
		_, err = a.devserver.Data.UpdateAppError(
			ctx,
			cqrs.UpdateAppErrorParams{
				ID:    app.ID,
				Error: sql.NullString{},
			},
		)
		if err != nil {
			return nil, publicerr.Wrap(err, 500, "Error updating app error")
		}
	}

	// setup a list of crons to be upserted into the queue for scheduling
	var crons []cron.CronItem

	// Attempt to get the existing app by URL, and delete it if possible.
	// We're going to recreate it below.
	//
	// We need to do this as we always create an app when entering the URL
	// via the UI.  This is a dev-server specific quirk.
	appID := inngest.DeterministicAppUUID(r.URL)

	tx, err := a.devserver.Data.WithTx(ctx)
	if err != nil {
		return nil, publicerr.Wrap(err, 500, "Error starting registration tx")
	}

	defer func() {
		method := enums.AppMethodServe
		if r.IsConnect() {
			method = enums.AppMethodConnect
		}

		appParams := cqrs.UpsertAppParams{
			// Use a deterministic ID for the app in dev.
			ID:          appID,
			Name:        r.AppName,
			SdkLanguage: r.SDKLanguage(),
			SdkVersion:  r.SDKVersion(),
			Framework: sql.NullString{
				String: r.Framework,
				Valid:  r.Framework != "",
			},
			Url:        r.URL,
			Checksum:   sum,
			Method:     method.String(),
			AppVersion: r.AppVersion,
		}

		// We want to save an app at the end, after handling each error.
		if err != nil {
			appParams.Error = sql.NullString{
				String: err.Error(),
				Valid:  true,
			}
		}
		_, _ = tx.UpsertApp(ctx, appParams)
		err = tx.Commit(ctx)
		if err != nil {
			l.Error("error registering functions", "error", err)
		}

		// handle cron sync to system queue
		for _, ci := range crons {
			if err := a.devserver.CronSyncer.Sync(ctx, ci); err != nil {
				l.Error("error on triggering cron-sync", "functionID", ci.FunctionID, "cronExpr", ci.Expression, "functionVersion", ci.FunctionVersion, "error", err)
			}
		}
	}()

	// Get a list of all functions
	existing, _ := tx.GetFunctionsByAppInternalID(ctx, appID)
	// And get a list of functions that we've upserted.  We'll delete all existing functions not in
	// this set.
	seen := map[uuid.UUID]struct{}{}

	// XXX (tonyhb): If we're authenticated, we can match the signing key against the workspace's
	// signing key and warn if the user has an invalid key.
	funcs, err := r.Parse(ctx)
	if err != nil && err != sdk.ErrNoFunctions {
		return nil, publicerr.Wrap(err, 400, "At least one function is invalid")
	}

	// For each function,
	for _, fn := range funcs {
		// Create a new UUID for the function.
		fn.ID = fn.DeterministicUUID()

		// Mark as seen.
		seen[fn.ID] = struct{}{}

		fnExists := false
		var currentFn *inngest.Function
		if cqrsFn, err := tx.GetFunctionByInternalUUID(ctx, fn.ID); err == nil {
			currentFn, err = cqrsFn.InngestFunction()
			if err != nil {
				return nil, publicerr.Wrap(err, 500, "Error unmarshalling function config")
			}
			if currentFn == nil {
				return nil, publicerr.Wrap(fmt.Errorf("function config empty"), 500, "Error unmarshalling function config")
			}
			fnExists = true
		}

		if fnExists {
			fn.FunctionVersion = currentFn.FunctionVersion + 1
		}
		config, err := json.Marshal(fn)
		if err != nil {
			return nil, publicerr.Wrap(err, 500, "Error marshalling function")
		}

		if fnExists {
			// Update the function config.
			_, err = tx.UpdateFunctionConfig(ctx, cqrs.UpdateFunctionConfigParams{
				ID:     fn.ID,
				Config: string(config),
			})
			if err != nil {
				return nil, publicerr.Wrap(err, 500, "Error updating function config")
			}
			cronExprs := fn.ScheduleExpressions()
			for _, cronExpr := range cronExprs {
				crons = append(crons, cron.CronItem{
					ID:              ulid.MustNew(ulid.Now(), rand.Reader),
					AccountID:       consts.DevServerAccountID,
					WorkspaceID:     consts.DevServerEnvID,
					AppID:           appID,
					FunctionID:      fn.ID,
					FunctionVersion: fn.FunctionVersion,
					Expression:      cronExpr,
					Op:              enums.CronOpUpdate,
				})
			}

			continue
		}

		_, err = tx.InsertFunction(ctx, cqrs.InsertFunctionParams{
			ID:        fn.ID,
			Name:      fn.Name,
			Slug:      fn.Slug,
			AppID:     appID,
			Config:    string(config),
			CreatedAt: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("function %s is invalid: %w", fn.Slug, err)
			return nil, publicerr.Wrap(err, 500, "Error saving function")
		}

		cronExprs := fn.ScheduleExpressions()
		for _, cronExpr := range cronExprs {
			crons = append(crons, cron.CronItem{
				ID:              ulid.MustNew(ulid.Now(), rand.Reader),
				AccountID:       consts.DevServerAccountID,
				WorkspaceID:     consts.DevServerEnvID,
				AppID:           appID,
				FunctionID:      fn.ID,
				FunctionVersion: fn.FunctionVersion,
				Expression:      cronExpr,
				Op:              enums.CronOpNew,
			})

		}
	}

	reply := &sync.Reply{
		OK:       true,
		Modified: true,
		AppID:    &appID,
		SyncID:   &syncID,
	}

	// Remove all unseen functions.
	deletes := []uuid.UUID{}
	for _, fn := range existing {
		if _, ok := seen[fn.ID]; !ok {
			deletes = append(deletes, fn.ID)
		}
	}
	if len(deletes) == 0 {
		return reply, nil
	}
	if err = tx.DeleteFunctionsByIDs(ctx, deletes); err != nil {
		return nil, publicerr.Wrap(err, 500, "Error deleting removed function")
	}
	return reply, nil
}

func (a devapi) OTLPTrace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := a.devserver.log

	body, err := io.ReadAll(r.Body)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  400,
			Err:     err,
			Message: err.Error(),
		})
	}
	defer r.Body.Close()

	var encoder ptrace.Unmarshaler
	cnt := r.Header.Get("Content-Type")
	switch cnt {
	case "application/x-protobuf":
		encoder = &ptrace.ProtoUnmarshaler{}
	case "application/json":
		encoder = &ptrace.JSONUnmarshaler{}
	default:
		l.Error("unknown content type for traces", "content-type", cnt)
		err = fmt.Errorf("unable to handle unknown content type for traces: %s", cnt)
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  400,
			Err:     err,
			Message: err.Error(),
		})
		return
	}

	hasAI := false

	traces, err := encoder.UnmarshalTraces(body)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Status:  400,
			Err:     err,
			Message: err.Error(),
		})
		return
	}
	l.Trace("recording otel trace spans", "len", traces.SpanCount())

	handler := newSpanIngestionHandler(a.devserver.Data)

	for i := 0; i < traces.ResourceSpans().Len(); i++ {
		rs := traces.ResourceSpans().At(i)

		rattr, err := convertMap(rs.Resource().Attributes().AsRaw())
		if err != nil {
			l.Warn("error parsing resource attributes", "error", err, "resource", rs.Resource().Attributes().AsRaw())
		}

		var serviceName string
		if v, ok := rattr["service.name"]; ok {
			serviceName = v
		}

		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)

			scopeName := ss.Scope().Name()
			scopeVersion := ss.Scope().Version()

			for k := 0; k < ss.Spans().Len(); k++ {
				span := ss.Spans().At(k)

				dur := span.EndTimestamp().AsTime().Sub(span.StartTimestamp().AsTime())
				sattr, err := convertMap(span.Attributes().AsRaw())
				if err != nil {
					l.Warn("error parsing span attributes", "error", err, "span_attr", span.Attributes().AsRaw())
				}

				if val, ok := sattr[consts.OtelSysFunctionHasAI]; ok {
					if boolVal, err := strconv.ParseBool(val); err == nil && boolVal {
						hasAI = true
					}
				}

				evts := []cqrs.SpanEvent{}
				for ei := 0; ei < span.Events().Len(); ei++ {
					evt := span.Events().At(ei)
					attr, err := convertMap(evt.Attributes().AsRaw())
					if err != nil {
						l.Error("error parsing span event", "error", err, "span_event", evt.Attributes().AsRaw())
						continue
					}

					evts = append(evts, cqrs.SpanEvent{
						Timestamp:  evt.Timestamp().AsTime(),
						Name:       evt.Name(),
						Attributes: attr,
					})
				}

				links := []cqrs.SpanLink{}
				for li := 0; li < span.Links().Len(); li++ {
					link := span.Links().At(li)
					attr, err := convertMap(link.Attributes().AsRaw())
					if err != nil {
						l.Error("error parsing span link", "error", err, "span_link", link.Attributes().AsRaw())
					}

					links = append(links, cqrs.SpanLink{
						TraceID:    link.TraceID().String(),
						SpanID:     link.SpanID().String(),
						TraceState: link.TraceState().AsRaw(),
						Attributes: attr,
					})
				}

				cqrsspan := &cqrs.Span{
					Timestamp:          span.StartTimestamp().AsTime(),
					TraceID:            span.TraceID().String(),
					SpanID:             span.SpanID().String(),
					SpanName:           span.Name(),
					SpanKind:           span.Kind().String(),
					ServiceName:        serviceName,
					ResourceAttributes: rattr,
					ScopeName:          scopeName,
					ScopeVersion:       scopeVersion,
					SpanAttributes:     sattr,
					Duration:           dur,
					StatusCode:         span.Status().Code().String(),
					Events:             evts,
					Links:              links,
				}

				if !span.ParentSpanID().IsEmpty() {
					id := span.ParentSpanID().String()
					cqrsspan.ParentSpanID = &id
				}
				if span.TraceState().AsRaw() != "" {
					state := span.TraceState().AsRaw()
					cqrsspan.TraceState = &state
				}
				if span.Status().Message() != "" {
					msg := span.Status().Message()
					cqrsspan.StatusMessage = &msg
				}

				if v, ok := sattr[consts.OtelAttrSDKRunID]; ok {
					if rid, err := ulid.Parse(v); err == nil {
						cqrsspan.RunID = &rid
					}
				}

				handler.Add(ctx, cqrsspan)
			}
		}
	}

	for _, s := range handler.Spans() {
		if err := a.devserver.Data.InsertSpan(ctx, s); err != nil {
			l.Error("error inserting span", "error", err, "span", *s)
		}
	}

	for _, r := range handler.TraceRuns() {
		// l.Debug("trace run", "run", r)
		r.HasAI = hasAI
		if err := a.devserver.Data.InsertTraceRun(ctx, r); err != nil {
			l.Error("error inserting trace run", "error", err, "trace_run", r)
		}
	}
}

// RemoveApp allows users to de-register an app by its URL
func (a devapi) RemoveApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	url := r.FormValue("url")

	app, err := a.devserver.Data.GetAppByURL(ctx, consts.DevServerEnvID, url)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 404, "App not found: %s", url))
		return
	}

	if err := a.devserver.Data.DeleteFunctionsByAppID(ctx, app.ID); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Error deleting functions"))
		return
	}

	if err := a.devserver.Data.DeleteApp(ctx, app.ID); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Error deleting app"))
		return
	}
}

func (a devapi) SetStepLimit(w http.ResponseWriter, r *http.Request) {
	functionId := r.FormValue("functionId")
	limitStr := r.FormValue("limit")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid limit: %s", limitStr))
		return
	}

	a.devserver.stepLimitOverrides[functionId] = limit
}

func (a devapi) RemoveStepLimit(w http.ResponseWriter, r *http.Request) {
	functionId := r.FormValue("functionId")

	if _, ok := a.devserver.stepLimitOverrides[functionId]; !ok {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(nil, 404, "No step limit set for function: %s", functionId))
		return
	}

	delete(a.devserver.stepLimitOverrides, functionId)
}

func (a devapi) SetStateSizeLimit(w http.ResponseWriter, r *http.Request) {
	functionId := r.FormValue("functionId")
	limitStr := r.FormValue("limit")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid limit: %s", limitStr))
		return
	}

	a.devserver.stateSizeLimitOverrides[functionId] = limit
}

func (a devapi) RemoveStateSizeLimit(w http.ResponseWriter, r *http.Request) {
	functionId := r.FormValue("functionId")

	if _, ok := a.devserver.stateSizeLimitOverrides[functionId]; !ok {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(nil, 404, "No state size limit set for function: %s", functionId))
		return
	}

	delete(a.devserver.stateSizeLimitOverrides, functionId)
}

func (a devapi) err(ctx context.Context, w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

type InfoResponse struct {
	// Version lists the version of the development server
	Version             string             `json:"version"`
	Authenticated       bool               `json:"authed"`
	StartOpts           StartOpts          `json:"startOpts"`
	Functions           []inngest.Function `json:"functions"`
	Handlers            []SDKHandler       `json:"handlers"`
	IsSingleNodeService bool               `json:"isSingleNodeService"`

	// If true, the server is running in a mode where it requires a signing key
	// to function and it is not set.
	IsMissingSigningKey bool `json:"isMissingSigningKey"`

	// If true, the server is running in a mode where it requires one or more
	// event keys to function and they are not set.
	IsMissingEventKeys bool `json:"isMissingEventKeys"`

	// Features acts as an in-memory feature flag for the UI
	Features map[string]bool `json:"features"`
}
