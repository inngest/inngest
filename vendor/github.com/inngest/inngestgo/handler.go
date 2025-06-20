package inngestgo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/syscode"
	sdkerrors "github.com/inngest/inngestgo/errors"
	"github.com/inngest/inngestgo/internal"
	"github.com/inngest/inngestgo/internal/event"
	"github.com/inngest/inngestgo/internal/fn"
	"github.com/inngest/inngestgo/internal/middleware"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/inngest/inngestgo/internal/types"
	"github.com/inngest/inngestgo/step"
)

var (
	ErrTypeMismatch = fmt.Errorf("cannot invoke function with mismatched types")

	errBadRequest      = fmt.Errorf("bad request")
	errFunctionMissing = fmt.Errorf("function not found")
	errUnauthorized    = fmt.Errorf("unauthorized")

	// DefaultMaxBodySize is the default maximum size read within a single incoming
	// invoke request (100MB).
	DefaultMaxBodySize = 1024 * 1024 * 100

	capabilities = types.Capabilities{
		InBandSync: types.InBandSyncV1,
		TrustProbe: types.TrustProbeV1,
		Connect:    types.ConnectV1,
	}
)

type handlerOpts struct {
	// Logger is the structured logger to use from Go's builtin structured
	// logging package.
	Logger *slog.Logger

	// SigningKey is the signing key for your app.  If nil, this defaults
	// to os.Getenv("INNGEST_SIGNING_KEY").
	SigningKey *string

	// SigningKeyFallback is the fallback signing key for your app. If nil, this
	// defaults to os.Getenv("INNGEST_SIGNING_KEY_FALLBACK").
	SigningKeyFallback *string

	// APIOrigin is the specified host to be used to make API calls
	APIBaseURL *string

	// EventAPIOrigin is the specified host to be used to send events to
	EventAPIBaseURL *string

	// ServeOrigin is the host to used for HTTP base function invoking.
	// It's used to specify the host were the functions are hosted on sync.
	// e.g. https://example.com
	ServeOrigin *string

	// ServePath is the path to use for HTTP base function invoking
	// It's used to specify the path were the functions are hosted on sync.
	// e.g. /api/inngest
	ServePath *string

	// Env is the branch environment to deploy to.  If nil, this uses
	// os.Getenv("INNGEST_ENV").  This only deploys to branches if the
	// signing key is a branch signing key.
	Env *string

	// RegisterURL is the URL to use when registering functions.  If nil
	// this defaults to Inngest's API.
	//
	// This only needs to be set when self hosting.
	RegisterURL *string

	// AppVersion supplies an application version identifier. This should change
	// whenever code within one of your Inngest function or any dependency thereof changes.
	AppVersion *string

	// MaxBodySize is the max body size to read for incoming invoke requests
	MaxBodySize int

	// URL that the function is served at.  If not supplied this is taken from
	// the incoming request's data.
	URL *url.URL

	// UseStreaming enables streaming - continued writes to the HTTP writer.  This
	// differs from true streaming in that we don't support server-sent events.
	UseStreaming bool

	// AllowInBandSync allows in-band syncs to occur. If nil, in-band syncs are
	// disallowed.
	AllowInBandSync *bool

	Dev *bool
}

// GetSigningKey returns the signing key defined within handlerOpts, or the default
// defined within INNGEST_SIGNING_KEY.
//
// This is the private key used to register functions and communicate with the private
// API.
func (h handlerOpts) GetSigningKey() string {
	if h.SigningKey == nil {
		return os.Getenv("INNGEST_SIGNING_KEY")
	}
	return *h.SigningKey
}

// GetSigningKeyFallback returns the signing key fallback defined within
// handlerOpts, or the default defined within INNGEST_SIGNING_KEY_FALLBACK.
//
// This is the fallback private key used to register functions and communicate
// with the private API. If a request fails auth with the signing key then we'll
// try again with the fallback
func (h handlerOpts) GetSigningKeyFallback() string {
	if h.SigningKeyFallback == nil {
		return os.Getenv("INNGEST_SIGNING_KEY_FALLBACK")
	}
	return *h.SigningKeyFallback
}

// GetAPIOrigin returns the host to use for sending API requests
func (h handlerOpts) GetAPIBaseURL() string {
	if h.APIBaseURL != nil {
		return *h.APIBaseURL
	}

	base := os.Getenv("INNGEST_API_BASE_URL")
	if base != "" {
		return base
	}

	base = os.Getenv("INNGEST_BASE_URL")
	if base != "" {
		return base
	}

	if h.isDev() {
		return DevServerURL()
	}

	return defaultAPIOrigin
}

// GetEventAPIOrigin returns the host to use for sending events
func (h handlerOpts) GetEventAPIBaseURL() string {
	if h.EventAPIBaseURL != nil {
		return *h.EventAPIBaseURL
	}

	origin := os.Getenv("INNGEST_EVENT_API_BASE_URL")
	if origin != "" {
		return origin
	}

	origin = os.Getenv("INNGEST_BASE_URL")
	if origin != "" {
		return origin
	}

	if h.isDev() {
		return DevServerURL()
	}

	return defaultEventAPIOrigin
}

// GetServeOrigin returns the host used for HTTP based executions
func (h handlerOpts) GetServeOrigin() string {
	if h.ServeOrigin != nil {
		return *h.ServeOrigin
	}
	return ""
}

// GetServePath returns the path used for HTTP based executions
func (h handlerOpts) GetServePath() string {
	if h.ServePath != nil {
		return *h.ServePath
	}
	return ""
}

// GetEnv returns the env defined within handlerOpts, or the default
// defined within INNGEST_ENV.
//
// This is the environment name used for preview/branch environments within Inngest.
func (h handlerOpts) GetEnv() string {
	if h.Env == nil {
		return os.Getenv("INNGEST_ENV")
	}
	return *h.Env
}

// GetRegisterURL returns the registration URL defined wtihin handlerOpts,
// defaulting to the production Inngest URL if nil.
func (h handlerOpts) GetRegisterURL() string {
	if h.RegisterURL == nil {
		return "https://www.inngest.com/fn/register"
	}
	return *h.RegisterURL
}

func (h handlerOpts) IsInBandSyncAllowed() bool {
	if h.AllowInBandSync != nil {
		return *h.AllowInBandSync
	}

	// TODO: Default to true once in-band syncing is stable
	if isTrue(os.Getenv(envKeyAllowInBandSync)) {
		return true
	}

	return false
}

func (h handlerOpts) isDev() bool {
	if h.Dev != nil {
		return *h.Dev
	}

	return IsDev()
}

// newHandler returns a new Handler for serving Inngest functions.
func newHandler(c Client, opts handlerOpts) *handler {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	if opts.MaxBodySize == 0 {
		opts.MaxBodySize = DefaultMaxBodySize
	}

	return &handler{
		handlerOpts: opts,
		appName:     c.AppID(),
		client:      c,
		funcs:       []ServableFunction{},
	}
}

type handler struct {
	handlerOpts

	appName string
	client  Client
	funcs   []ServableFunction
	// lock prevents reading the function maps while serving
	l sync.RWMutex
}

func (h *handler) GetAppName() string {
	return h.appName
}

func (h *handler) GetAppVersion() *string {
	return h.AppVersion
}

func (h *handler) GetFunctions() []ServableFunction {
	return h.funcs
}

func (h *handler) SetOptions(opts handlerOpts) *handler {
	h.handlerOpts = opts

	if opts.MaxBodySize == 0 {
		opts.MaxBodySize = DefaultMaxBodySize
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	return h
}

func (h *handler) SetAppName(name string) *handler {
	h.appName = name
	return h
}

func (h *handler) Register(funcs ...ServableFunction) {
	h.l.Lock()
	defer h.l.Unlock()

	// Create a map of functions by slug.  If we're registering a function
	// that already exists, clear it.
	slugs := map[string]ServableFunction{}
	for _, f := range h.funcs {
		slugs[f.FullyQualifiedID()] = f
	}

	for _, f := range funcs {
		slugs[f.FullyQualifiedID()] = f
	}

	newFuncs := make([]ServableFunction, len(slugs))
	i := 0
	for _, f := range slugs {
		newFuncs[i] = f
		i++
	}

	h.funcs = newFuncs
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Logger.Debug("received http request", "method", r.Method)
	SetBasicResponseHeaders(w)

	switch r.Method {
	case http.MethodGet:
		if err := h.inspect(w, r); err != nil {
			_ = publicerr.WriteHTTP(w, err)
		}
		return
	case http.MethodPost:
		probe := r.URL.Query().Get("probe")
		if probe == "trust" {
			err := h.trust(r.Context(), w, r)
			if err != nil {
				var perr publicerr.Error
				if !errors.As(err, &perr) {
					perr = publicerr.Error{
						Err:     err,
						Message: err.Error(),
						Status:  500,
					}
				}

				if perr.Status == 0 {
					perr.Status = http.StatusInternalServerError
				}

				_ = publicerr.WriteHTTP(w, perr)
			}
			return
		}

		if err := h.invoke(w, r); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, errFunctionMissing) {
				// XXX: This is a 500 within the JS SDK. We should probably
				// change the JS SDK's status code to 410. 404 indicates that
				// the overall API for serving Inngest isn't found.
				status = http.StatusGone
			} else if errors.Is(err, errBadRequest) {
				status = http.StatusBadRequest
			} else if errors.Is(err, errUnauthorized) {
				status = http.StatusUnauthorized
			}
			w.WriteHeader(status)
			w.Header().Set("content-type", "application/json")
			_ = json.NewEncoder(w).Encode(sdkrequest.ErrorResponse{
				Message: err.Error(),
			})
		}
		return
	case http.MethodPut:
		if err := h.register(w, r); err != nil {
			h.Logger.Error("error registering functions", "error", err.Error())

			code := syscode.CodeUnknown
			status := http.StatusInternalServerError
			if err, ok := err.(publicerr.Error); ok {
				status = err.Status

				if err, ok := err.Err.(syscode.Error); ok {
					code = err.Code
				}
			}
			w.WriteHeader(status)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code":    code,
				"message": err.Error(),
			})
		}
		return
	}
}

// register self-registers the handler's functions with Inngest.  This upserts
// all functions and automatically allows all functions to immediately be triggered
// by incoming events or schedules.
func (h *handler) register(w http.ResponseWriter, r *http.Request) error {
	var syncKind string
	var err error
	if r.Header.Get(HeaderKeySyncKind) == SyncKindInBand && h.IsInBandSyncAllowed() {
		syncKind = SyncKindInBand
		err = h.inBandSync(w, r)
	} else {
		syncKind = SyncKindOutOfBand
		err = h.outOfBandSync(w, r)
	}

	if err != nil {
		h.Logger.Error(
			"sync error",
			"error", err,
			"syncKind", syncKind,
		)
	}
	return err
}

type inBandSynchronizeRequest struct {
	URL string `json:"url"`
}

func (i inBandSynchronizeRequest) Validate() error {
	if i.URL == "" {
		return fmt.Errorf("missing URL")
	}
	return nil
}

type inBandSynchronizeResponse struct {
	AppID       string          `json:"app_id"`
	Env         *string         `json:"env"`
	Framework   *string         `json:"framework"`
	Functions   []fn.SyncConfig `json:"functions"`
	Inspection  map[string]any  `json:"inspection"`
	Platform    *string         `json:"platform"`
	SDKAuthor   string          `json:"sdk_author"`
	SDKLanguage string          `json:"sdk_language"`
	SDKVersion  string          `json:"sdk_version"`
	URL         string          `json:"url"`
}

func (h *handler) inBandSync(
	w http.ResponseWriter,
	r *http.Request,
) error {
	ctx := r.Context()
	defer func() {
		_ = r.Body.Close()
	}()

	var sig string
	if !h.isDev() {
		if sig = r.Header.Get(HeaderKeySignature); sig == "" {
			return publicerr.Error{
				Err: syscode.Error{
					Code:    syscode.CodeHTTPMissingHeader,
					Message: fmt.Sprintf("missing %s header", HeaderKeySignature),
				},
				Status: 401,
			}
		}
	}

	max := h.MaxBodySize
	if max == 0 {
		max = DefaultMaxBodySize
	}
	reqByt, err := io.ReadAll(http.MaxBytesReader(w, r.Body, int64(max)))
	if err != nil {
		return publicerr.Error{
			Err:    fmt.Errorf("error reading request body"),
			Status: 500,
		}
	}

	valid, skey, err := ValidateRequestSignature(
		ctx,
		sig,
		h.GetSigningKey(),
		h.GetSigningKeyFallback(),
		reqByt,
		h.isDev(),
	)
	if err != nil {
		return publicerr.Error{
			Err: syscode.Error{
				Code:    syscode.CodeSigVerificationFailed,
				Message: "error validating signature",
			},
			Status: 401,
		}
	}
	if !valid {
		return publicerr.Error{
			Err: syscode.Error{
				Code:    syscode.CodeSigVerificationFailed,
				Message: "invalid signature",
			},
			Status: 401,
		}
	}

	var reqBody inBandSynchronizeRequest
	err = json.Unmarshal(reqByt, &reqBody)
	if err != nil {
		return publicerr.Error{
			Err:    fmt.Errorf("malformed input: %w", err),
			Status: 400,
		}
	}
	err = reqBody.Validate()
	if err != nil {
		return publicerr.Error{
			Err:    fmt.Errorf("malformed input: %w", err),
			Status: 400,
		}
	}

	appURL, err := url.Parse(reqBody.URL)
	if err != nil {
		return publicerr.Error{
			Err:    fmt.Errorf("malformed input: %w", err),
			Status: 400,
		}
	}
	if h.URL != nil {
		appURL = h.URL
	}

	fns, err := createFunctionConfigs(h.appName, h.funcs, *appURL, false)
	if err != nil {
		return fmt.Errorf("error creating function configs: %w", err)
	}

	var env *string
	if h.GetEnv() != "" {
		val := h.GetEnv()
		env = &val
	}

	inspection, err := h.createSecureInspection()
	if err != nil {
		return fmt.Errorf("error creating inspection: %w", err)
	}
	inspectionMap, err := types.StructToMap(inspection)
	if err != nil {
		return fmt.Errorf("error converting inspection to map: %w", err)
	}

	respBody := inBandSynchronizeResponse{
		AppID:       h.appName,
		Env:         env,
		Functions:   fns,
		Inspection:  inspectionMap,
		SDKAuthor:   SDKAuthor,
		SDKLanguage: SDKLanguage,
		SDKVersion:  SDKVersion,
		URL:         appURL.String(),
	}

	respByt, err := json.Marshal(respBody)
	if err != nil {
		return fmt.Errorf("error marshalling response: %w", err)
	}

	resSig, err := signWithoutJCS(time.Now(), []byte(skey), respByt)
	if err != nil {
		return fmt.Errorf("error signing response: %w", err)
	}
	w.Header().Add(HeaderKeySignature, resSig)
	w.Header().Add(HeaderKeyContentType, "application/json")
	w.Header().Add(HeaderKeySyncKind, SyncKindInBand)

	err = json.NewEncoder(w).Encode(respBody)
	if err != nil {
		return fmt.Errorf("error writing response: %w", err)
	}

	return nil
}

func (h *handler) outOfBandSync(w http.ResponseWriter, r *http.Request) error {
	h.l.Lock()
	defer h.l.Unlock()

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host

	// Get the sync ID from the URL and then remove it, since we don't want the
	// sync ID to show in the function URLs (that would affect the checksum and
	// is ugly in the UI)
	qp := r.URL.Query()
	syncID := qp.Get("deployId")
	qp.Del("deployId")
	r.URL.RawQuery = qp.Encode()

	pathAndParams := r.URL.String()

	appVersion := ""
	if h.AppVersion != nil {
		appVersion = *h.AppVersion
	}

	config := types.RegisterRequest{
		URL:        fmt.Sprintf("%s://%s%s", scheme, host, pathAndParams),
		V:          "1",
		DeployType: types.DeployTypePing,
		SDK:        HeaderValueSDK,
		AppName:    h.appName,
		Headers: types.Headers{
			Env:      h.GetEnv(),
			Platform: platform(),
		},
		Capabilities: capabilities,
		AppVersion:   appVersion,
	}

	fns, err := createFunctionConfigs(h.appName, h.funcs, *h.url(r), false)
	if err != nil {
		return fmt.Errorf("error creating function configs: %w", err)
	}
	config.Functions = fns

	registerURL := fmt.Sprintf("%s/fn/register", h.GetAPIBaseURL())
	if h.RegisterURL != nil {
		registerURL = *h.RegisterURL
	}

	createRequest := func() (*http.Request, error) {
		byt, err := json.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("error marshalling function config: %w", err)
		}

		req, err := http.NewRequest(http.MethodPost, registerURL, bytes.NewReader(byt))
		if err != nil {
			return nil, fmt.Errorf("error creating new request: %w", err)
		}
		if syncID != "" {
			qp := req.URL.Query()
			qp.Set("deployId", syncID)
			req.URL.RawQuery = qp.Encode()
		}

		// If the request specifies a server kind then include it as an expectation
		// in the outgoing request
		if r.Header.Get(HeaderKeyServerKind) != "" {
			req.Header.Set(
				HeaderKeyExpectedServerKind,
				r.Header.Get(HeaderKeyServerKind),
			)
		}

		if h.GetEnv() != "" {
			req.Header.Add(HeaderKeyEnv, h.GetEnv())
		}

		SetBasicRequestHeaders(req)

		return req, nil
	}

	resp, err := fetchWithAuthFallback(
		createRequest,
		h.GetSigningKey(),
		h.GetSigningKeyFallback(),
	)
	if err != nil {
		return fmt.Errorf("error performing registration request: %w", err)
	}
	if resp.StatusCode > 299 {
		body := map[string]any{}
		byt, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(byt, &body); err != nil {
			return fmt.Errorf("error reading register response: %w\n\n%s", err, byt)
		}
		return fmt.Errorf("error registering functions: %s", body["error"])
	}

	w.Header().Add(HeaderKeySyncKind, SyncKindOutOfBand)

	return nil
}

func (h *handler) url(r *http.Request) *url.URL {
	if h.URL != nil {
		return h.URL
	}

	// Get the current URL.
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	u, _ := url.Parse(fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI))
	return u
}

func createFunctionConfigs(
	appName string,
	fns []ServableFunction,
	appURL url.URL,
	isConnect bool,
) ([]fn.SyncConfig, error) {
	if appName == "" {
		return nil, fmt.Errorf("missing app name")
	}
	if !isConnect && appURL == (url.URL{}) {
		return nil, fmt.Errorf("missing URL")
	}

	fnConfigs := make([]fn.SyncConfig, len(fns))
	for i, sf := range fns {
		f := fn.GetFnSyncConfig(sf)
		f.UpdateSteps(appURL)

		fnConfigs[i] = *f
	}

	return fnConfigs, nil
}

// invoke handles incoming POST calls to invoke a function, delegating to invoke() after validating
// the request.
func (h *handler) invoke(w http.ResponseWriter, r *http.Request) error {
	cImpl, ok := h.client.(*apiClient)
	if !ok {
		return errors.New("invalid client type")
	}
	mw := middleware.NewMiddlewareManager().Add(cImpl.Middleware...)

	var sig string
	defer func() {
		_ = r.Body.Close()
	}()

	if !h.isDev() {
		if sig = r.Header.Get(HeaderKeySignature); sig == "" {
			return errUnauthorized
		}
	}

	max := h.MaxBodySize
	if max == 0 {
		max = DefaultMaxBodySize
	}
	byt, err := io.ReadAll(http.MaxBytesReader(w, r.Body, int64(max)))
	if err != nil {
		h.Logger.Error("error decoding function request", "error", err)
		return fmt.Errorf("%w: %s", errBadRequest, err)
	}

	if valid, _, err := ValidateRequestSignature(
		r.Context(),
		sig,
		h.GetSigningKey(),
		h.GetSigningKeyFallback(),
		byt,
		h.isDev(),
	); !valid {
		h.Logger.Error("unauthorized inngest invoke request", "error", err)
		return errUnauthorized
	}

	fnID := r.URL.Query().Get("fnId")

	request := &sdkrequest.Request{}
	if err := json.Unmarshal(byt, request); err != nil {
		h.Logger.Error("error decoding function request", "error", err)
		return fmt.Errorf("%w: %s", errBadRequest, err)
	}

	if request.UseAPI {
		// TODO: implement this
		// retrieve data from API
		// request.Steps =
		// request.Events =
		_ = 0 // no-op to avoid linter error
	}

	h.l.RLock()
	var fn ServableFunction
	for _, f := range h.funcs {
		isOldFormat := f.ID() == fnID // Only include function slug
		if f.FullyQualifiedID() == fnID || isOldFormat {
			fn = f
			break
		}
	}
	h.l.RUnlock()

	if fn == nil {
		return fmt.Errorf("%w: %s", errFunctionMissing, fnID)
	}

	l := h.Logger.With("fn", fnID, "call_ctx", request.CallCtx)
	l.Debug("calling function")

	stream, streamCancel := context.WithCancel(context.Background())
	if h.UseStreaming {
		w.WriteHeader(201)
		go func() {
			for {
				if stream.Err() != nil {
					return
				}
				_, _ = w.Write([]byte(" "))
				<-time.After(5 * time.Second)
			}
		}()
	}

	var stepID *string
	if rawStepID := r.URL.Query().Get("stepId"); rawStepID != "" && rawStepID != "step" {
		stepID = &rawStepID
	}

	// Invoke the function, then immediately stop the streaming buffer.
	resp, ops, err := invoke(
		r.Context(),
		h.client,
		mw,
		fn,
		h.GetSigningKey(),
		request,
		stepID,
	)
	streamCancel()

	// NOTE: When triggering step errors, we should have an OpcodeStepError
	// within ops alongside an error.  We can safely ignore that error, as it's
	// only used for checking whether the step used a NoRetryError or RetryAtError
	//
	// For that reason, we check those values first.
	noRetry := sdkerrors.IsNoRetryError(err)
	retryAt := sdkerrors.GetRetryAtTime(err)
	if len(ops) == 1 && ops[0].Op == enums.OpcodeStepError {
		// Now we've handled error types we can ignore step
		// errors safely.
		err = nil
	}

	// Now that we've handled the OpcodeStepError, if we *still* ahve
	// a StepError kind returned from a function we must have an unhandled
	// step error.  This is a NonRetryableError, as the most likely code is:
	//
	// 	_, err := step.Run(ctx, func() (any, error) { return fmt.Errorf("") })
	// 	if err != nil {
	// 	     return err
	// 	}
	if sdkerrors.IsStepError(err) {
		err = fmt.Errorf("unhandled step error: %s", err)
		noRetry = true
	}

	if h.UseStreaming {
		if err != nil {
			// TODO: Add retry-at.
			return json.NewEncoder(w).Encode(StreamResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("error calling function: %s", err.Error()),
				NoRetry:    noRetry,
				RetryAt:    retryAt,
			})
		}
		if len(ops) > 0 {
			return json.NewEncoder(w).Encode(StreamResponse{
				StatusCode: 206,
				Body:       ops,
			})
		}
		return json.NewEncoder(w).Encode(StreamResponse{
			StatusCode: 200,
			Body:       resp,
		})
	}

	// These may be added even for 2xx codes with step errors.
	if noRetry {
		w.Header().Add(HeaderKeyNoRetry, "true")
	}
	if retryAt != nil {
		w.Header().Add(HeaderKeyRetryAfter, retryAt.Format(time.RFC3339))
	}

	if err != nil {
		l.Error("error calling function", "error", err)
		return err
	}

	if len(ops) > 0 {
		// Return the function opcode returned here so that we can re-invoke this
		// function and manage state appropriately.  Any opcode here takes precedence
		// over function return values as the function has not yet finished.
		w.WriteHeader(206)
		return json.NewEncoder(w).Encode(ops)
	}

	// Return the function response.
	return json.NewEncoder(w).Encode(resp)
}

type insecureInspection struct {
	SchemaVersion string `json:"schema_version"`

	AuthenticationSucceeded *bool  `json:"authentication_succeeded"`
	FunctionCount           int    `json:"function_count"`
	HasEventKey             bool   `json:"has_event_key"`
	HasSigningKey           bool   `json:"has_signing_key"`
	HasSigningKeyFallback   bool   `json:"has_signing_key_fallback"`
	Mode                    string `json:"mode"`
}

type secureInspection struct {
	insecureInspection

	APIOrigin              string             `json:"api_origin"`
	AppID                  string             `json:"app_id"`
	Capabilities           types.Capabilities `json:"capabilities"`
	Env                    *string            `json:"env"`
	EventAPIOrigin         string             `json:"event_api_origin"`
	EventKeyHash           *string            `json:"event_key_hash"`
	Framework              string             `json:"framework"`
	SDKLanguage            string             `json:"sdk_language"`
	SDKVersion             string             `json:"sdk_version"`
	ServeOrigin            *string            `json:"serve_origin"`
	ServePath              *string            `json:"serve_path"`
	SigningKeyFallbackHash *string            `json:"signing_key_fallback_hash"`
	SigningKeyHash         *string            `json:"signing_key_hash"`
}

func (h *handler) createInsecureInspection(
	authenticationSucceeded *bool,
) (*insecureInspection, error) {
	mode := "cloud"
	if h.isDev() {
		mode = "dev"
	}

	return &insecureInspection{
		AuthenticationSucceeded: authenticationSucceeded,
		FunctionCount:           len(h.funcs),
		HasEventKey:             os.Getenv("INNGEST_EVENT_KEY") != "",
		HasSigningKey:           h.GetSigningKey() != "",
		HasSigningKeyFallback:   h.GetSigningKeyFallback() != "",
		Mode:                    mode,
		SchemaVersion:           "2024-05-24",
	}, nil
}

func (h *handler) createSecureInspection() (*secureInspection, error) {
	apiOrigin := defaultAPIOrigin
	eventAPIOrigin := defaultEventAPIOrigin
	if h.isDev() {
		apiOrigin = DevServerURL()
		eventAPIOrigin = DevServerURL()
	}

	var eventKeyHash *string
	if os.Getenv("INNGEST_EVENT_KEY") != "" {
		hash := hashEventKey(os.Getenv("INNGEST_EVENT_KEY"))
		eventKeyHash = &hash
	}

	var signingKeyHash *string
	if h.GetSigningKey() != "" {
		key, err := hashedSigningKey([]byte(h.GetSigningKey()))
		if err != nil {
			return nil, fmt.Errorf("error hashing signing key: %w", err)
		}
		hash := string(key)
		signingKeyHash = &hash
	}

	var signingKeyFallbackHash *string
	if h.GetSigningKeyFallback() != "" {
		key, err := hashedSigningKey([]byte(h.GetSigningKeyFallback()))
		if err != nil {
			return nil, fmt.Errorf("error hashing signing key fallback: %w", err)
		}
		hash := string(key)
		signingKeyFallbackHash = &hash
	}

	authenticationSucceeded := true

	var env *string
	if h.GetEnv() != "" {
		val := h.GetEnv()
		env = &val
	}

	var serveOrigin, servePath *string
	if h.URL != nil {
		serveOriginStr := h.URL.Scheme + "://" + h.URL.Host
		serveOrigin = &serveOriginStr

		servePath = &h.URL.Path
	}

	authenticationSucceeded = true
	insecureInspection, err := h.createInsecureInspection(&authenticationSucceeded)
	if err != nil {
		return nil, fmt.Errorf("error creating inspection: %w", err)
	}

	return &secureInspection{
		insecureInspection:     *insecureInspection,
		APIOrigin:              apiOrigin,
		AppID:                  h.appName,
		Capabilities:           capabilities,
		Env:                    env,
		EventAPIOrigin:         eventAPIOrigin,
		EventKeyHash:           eventKeyHash,
		SDKLanguage:            SDKLanguage,
		SDKVersion:             SDKVersion,
		SigningKeyFallbackHash: signingKeyFallbackHash,
		SigningKeyHash:         signingKeyHash,
		ServeOrigin:            serveOrigin,
		ServePath:              servePath,
	}, nil
}

func (h *handler) inspect(w http.ResponseWriter, r *http.Request) error {
	defer func() {
		_ = r.Body.Close()
	}()

	sig := r.Header.Get(HeaderKeySignature)
	if sig != "" {
		valid, _, _ := ValidateRequestSignature(
			r.Context(),
			sig,
			h.GetSigningKey(),
			h.GetSigningKeyFallback(),
			[]byte{},
			h.isDev(),
		)
		if valid {
			inspection, err := h.createSecureInspection()
			if err != nil {
				return err
			}

			w.Header().Set(HeaderKeyContentType, "application/json")
			return json.NewEncoder(w).Encode(inspection)
		}
	}

	var authenticationSucceeded *bool
	if sig != "" {
		val := false
		authenticationSucceeded = &val
	}

	inspection, err := h.createInsecureInspection(authenticationSucceeded)
	if err != nil {
		return fmt.Errorf("error creating inspection: %w", err)
	}

	w.Header().Set(HeaderKeyContentType, "application/json")
	return json.NewEncoder(w).Encode(inspection)
}

type trustProbeResponse struct {
	Error *string `json:"error,omitempty"`
}

func (h *handler) trust(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	if h.isDev() {
		w.WriteHeader(200)
		return nil
	}

	w.Header().Add("Content-Type", "application/json")
	sig := r.Header.Get(HeaderKeySignature)
	if sig == "" {
		return publicerr.Error{
			Message: fmt.Sprintf("missing %s header", HeaderKeySignature),
			Status:  401,
		}
	}

	max := h.MaxBodySize
	if max == 0 {
		max = DefaultMaxBodySize
	}
	byt, err := io.ReadAll(http.MaxBytesReader(w, r.Body, int64(max)))
	if err != nil {
		h.Logger.Error("error decoding function request", "error", err)
		return publicerr.Error{
			Message: fmt.Sprintf("error decoding function request: %s", err),
			Status:  400,
		}
	}

	valid, key, err := ValidateRequestSignature(
		ctx,
		r.Header.Get("X-Inngest-Signature"),
		h.GetSigningKey(),
		h.GetSigningKeyFallback(),
		byt,
		h.isDev(),
	)
	if err != nil {
		return publicerr.Error{
			Message: fmt.Sprintf("error validating signature: %s", err),
			Status:  401,
		}
	}
	if !valid {
		return publicerr.Error{
			Message: "invalid signature",
			Status:  401,
		}
	}

	byt, err = json.Marshal(trustProbeResponse{})
	if err != nil {
		return err
	}

	resSig, err := signWithoutJCS(time.Now(), []byte(key), byt)
	if err != nil {
		return publicerr.Error{
			Message: fmt.Sprintf("error signing response: %s", err),
			Status:  500,
		}
	}

	w.Header().Add("X-Inngest-Signature", resSig)
	w.WriteHeader(200)
	_, err = w.Write(byt)
	if err != nil {
		h.Logger.Error("error writing trust probe response", "error", err)
	}

	return nil
}

type StreamResponse struct {
	StatusCode int               `json:"status"`
	Body       any               `json:"body"`
	RetryAt    *time.Time        `json:"retryAt"`
	NoRetry    bool              `json:"noRetry"`
	Headers    map[string]string `json:"headers"`
}

// invoke calls a given servable function with the specified input event.  The input event must
// be fully typed.
func invoke(
	ctx context.Context,
	client Client,
	mw *middleware.MiddlewareManager,
	sf ServableFunction,
	signingKey string,
	input *sdkrequest.Request,
	stepID *string,
) (any, []sdkrequest.GeneratorOpcode, error) {
	if sf.Func() == nil {
		// This should never happen, but as sf.Func returns a nillable type we
		// must check that the function exists.
		return nil, nil, fmt.Errorf("no function defined")
	}

	// Create a new context.  This context is cancellable and stores the opcode that ran
	// within a step.  This allows us to prevent any execution of future tools after a
	// tool has run.
	fCtx, cancel := context.WithCancel(
		internal.ContextWithMiddlewareManager(
			internal.ContextWithEventSender(ctx, client),
			mw,
		),
	)
	if stepID != nil {
		fCtx = step.SetTargetStepID(fCtx, *stepID)
	}

	// This must be a pointer so that it can be mutated from within function tools.
	mgr := sdkrequest.NewManager(sf, mw, cancel, input, signingKey)
	fCtx = sdkrequest.SetManager(fCtx, mgr)

	// Create a new Input type.  We don't know ahead of time the type signature as
	// this is generic;  we instead grab the generic event element and instantiate
	// it using the data within request.
	fVal := reflect.ValueOf(sf.Func())
	inputVal := reflect.New(fVal.Type().In(1)).Elem()

	updateInput(
		mgr,
		sf,
		inputVal,
		input.Event,
		types.ToAnySlice(input.Events),
	)

	// Set InputCtx
	callCtx := InputCtx{
		Env:        input.CallCtx.Env,
		FunctionID: input.CallCtx.FunctionID,
		RunID:      input.CallCtx.RunID,
		StepID:     input.CallCtx.StepID,
		Attempt:    input.CallCtx.Attempt,
	}
	inputVal.FieldByName("InputCtx").Set(reflect.ValueOf(callCtx))

	var (
		res      []reflect.Value
		panicErr error

		// fnResponse is the actual response from the fn
		fnResponse any
		// fnError is the actual error from the fn.
		fnError error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				callCtx := mgr.MiddlewareCallCtx()

				// Was this us attepmting to prevent functions from continuing, using
				// panic as a crappy control flow because go doesn't have generators?
				//
				// XXX: I'm not very happy with using this;  it is dirty
				if _, ok := r.(step.ControlHijack); ok {
					// Step attempt ended (completed or errored).
					//
					// Note that if this is a step.Run, middleware has already been invoked
					// via step.Run and this is skipped due to idempotency in the middleware manager.
					// Because this isn't a step.Run, it's safe to call this with nil data and error.
					mw.AfterExecution(ctx, callCtx, nil, nil)
					return
				}

				panicStack := string(debug.Stack())
				panicErr = fmt.Errorf("function panicked: %v.  stack:\n%s", r, panicStack)

				mw.AfterExecution(ctx, callCtx, nil, nil)
				mw.OnPanic(ctx, callCtx, r, panicStack)
			}
		}()

		// Run the TransformInput middleware hook.
		{
			// Build TransformableInput.
			var evt Event
			_ = json.Unmarshal(input.Event, &evt)
			evts := make([]*event.Event, len(input.Events))
			for i, rawjson := range input.Events {
				var evt event.Event
				if err := json.Unmarshal(rawjson, &evt); err != nil {
					mgr.SetErr(fmt.Errorf("error unmarshalling event for function: %w", err))
					panic(step.ControlHijack{})
				}
				evts[i] = &evt
			}
			mwInput := &middleware.TransformableInput{
				Event:  &evt,
				Events: evts,
			}
			mwInput.WithContext(fCtx)

			// Run hook.
			mw.TransformInput(ctx, mgr.MiddlewareCallCtx(), mwInput)

			// Update the context in case the hook changed it.
			fCtx = mwInput.Context()

			// Update the input we're passing to the Inngest function.
			updateInput(
				mgr,
				sf,
				inputVal,
				mwInput.Event,
				types.ToAnySlice(mwInput.Events),
			)
		}

		if len(input.Steps) == 0 {
			// There are no memoized steps, so the start of the function is "new
			// code".
			mw.BeforeExecution(fCtx, mgr.MiddlewareCallCtx())
		}

		// Call the defined function with the input data.
		res = fVal.Call([]reflect.Value{
			reflect.ValueOf(fCtx),
			inputVal,
		})

		// Set the function response.
		if len(res) >= 1 {
			fnResponse = res[0].Interface()
		}

		// Function ended.  Get the types for the middleare call.
		if len(res) >= 2 && !res[1].IsNil() {
			fnError = res[1].Interface().(error)
		}

		mw.AfterExecution(ctx, mgr.MiddlewareCallCtx(), fnResponse, fnError)

		{
			// Transform output via MW
			out := &middleware.TransformableOutput{
				Result: fnResponse,
				Error:  fnError,
			}
			mw.TransformOutput(ctx, mgr.MiddlewareCallCtx(), out)
			// And update the vars
			fnResponse = out.Result
			fnError = out.Error
		}
	}()

	// Override errors here.
	if panicErr != nil {
		fnError = panicErr
	} else if mgr.Err() != nil {
		// This is higher precedence than a return error.
		fnError = mgr.Err()
	}

	return fnResponse, mgr.Ops(), fnError
}

// updateInput applies the middleware input to the function input.
func updateInput(
	mgr sdkrequest.InvocationManager,
	fn ServableFunction,
	fnInput reflect.Value,
	// mwInput *middleware.TransformableInput,
	event any,
	events []any,
) {
	// If we have an actual value to add to the event, vs `Input[any]`, set it.
	if fn.ZeroEvent() != nil {
		eventType := reflect.TypeOf(fn.ZeroEvent())

		// Apply event.
		{
			// byt, err := json.Marshal(mwInput.Event)
			byt, err := json.Marshal(event)
			if err != nil {
				mgr.SetErr(fmt.Errorf("error marshalling event for function: %w", err))
				panic(step.ControlHijack{})
			}

			// The same type as the event.
			newEvent := reflect.New(eventType).Interface()

			if err := json.Unmarshal(byt, newEvent); err != nil {
				mgr.SetErr(fmt.Errorf("error unmarshalling event for function: %w", err))
				panic(step.ControlHijack{})
			}
			fnInput.FieldByName("Event").Set(reflect.ValueOf(newEvent).Elem())
		}

		// Apply events.
		{
			eventsType := reflect.SliceOf(eventType)
			newEvents := reflect.MakeSlice(eventsType, 0, len(events))
			for _, evt := range events {
				// events := reflect.MakeSlice(eventsType, 0, len(mwInput.Events))
				// for _, evt := range mwInput.Events {
				byt, err := json.Marshal(evt)
				if err != nil {
					mgr.SetErr(fmt.Errorf("error marshalling event for function: %w", err))
					panic(step.ControlHijack{})
				}

				// The same type as the event.
				newEvent := reflect.New(eventType).Interface()
				if err := json.Unmarshal(byt, newEvent); err != nil {
					mgr.SetErr(fmt.Errorf("error unmarshalling event for function: %w", err))
					panic(step.ControlHijack{})
				}

				newEvents = reflect.Append(newEvents, reflect.ValueOf(newEvent).Elem())
			}
			fnInput.FieldByName("Events").Set(newEvents)
		}
	} else {
		// Apply event.
		{
			byt, err := json.Marshal(event)
			if err != nil {
				mgr.SetErr(fmt.Errorf("error marshalling event for function: %w", err))
				panic(step.ControlHijack{})
			}

			newEvent := map[string]any{}
			if err := json.Unmarshal(byt, &newEvent); err != nil {
				mgr.SetErr(fmt.Errorf("error unmarshalling event for function: %w", err))
				panic(step.ControlHijack{})
			}
			fnInput.FieldByName("Event").Set(reflect.ValueOf(newEvent))
		}

		// Apply events.
		{
			newEvents := make([]any, len(events))
			for i, evt := range events {
				byt, err := json.Marshal(evt)
				if err != nil {
					mgr.SetErr(fmt.Errorf("error marshalling event for function: %w", err))
					panic(step.ControlHijack{})
				}

				var newEvent map[string]any
				if err := json.Unmarshal(byt, &newEvent); err != nil {
					mgr.SetErr(fmt.Errorf("error unmarshalling event for function: %w", err))
					panic(step.ControlHijack{})
				}

				newEvents[i] = newEvent
			}
			fnInput.FieldByName("Events").Set(reflect.ValueOf(newEvents))
		}
	}
}
