package stephttp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/inngest/inngestgo/pkg/env"
	"github.com/inngest/inngestgo/pkg/httputil"
	"github.com/inngest/inngestgo/pkg/version"
	"github.com/oklog/ulid/v2"
)

func processRequest(p *provider, r *http.Request, w http.ResponseWriter, next http.HandlerFunc) error {
	owner := &requestOwner{
		r:        r,
		w:        newResponseWriter(w),
		next:     next,
		provider: p,
		mgr: sdkrequest.NewManager(sdkrequest.Opts{
			SigningKey: p.opts.signingKey(),
			Mode:       sdkrequest.StepModeManual,
		}),

		startTime: time.Now(),
	}

	owner.run = CheckpointRun{
		RunID: ulid.MustNew(
			uint64(owner.startTime.UnixMilli()),
			rand.Reader,
		),
	}

	return owner.handle(r.Context())
}

// requestOwner represents a manager for a single request to a sync function.
// this is short lived and only exists for one request.
type requestOwner struct {
	// # Dependency injection options

	// r represents the incoming http request for the sync function.  This may be
	// an end user's request, or it may be a re-entry from Inngest.
	r *http.Request
	// w responds to the request. This is always our wrapped responseWriter.
	w *responseWriter
	// next is the API handler to call to execute the sync function, ie. next in
	// the middleware chain.
	next http.HandlerFunc
	// provider references the parent provider that created the
	// request.
	provider *provider
	// sdkrequest is the step execution manager for the underlying function.
	mgr sdkrequest.InvocationManager

	// # Run-specific options

	// config represents function-specific config.
	config *FnOpts
	// startTime tracks the start time of the API request. We must track this
	// as early as possible.
	startTime time.Time
	// run represents the IDs for the current sync run.
	run CheckpointRun
}

func (o *requestOwner) handle(ctx context.Context) error {
	// Always add the run ID to the header.
	o.w.Header().Add("x-run-id", o.run.RunID.String())
	o.w.Header().Add("X-Inngest-SDK", version.GetVersion())

	// Always add the manager to context.
	ctx = sdkrequest.SetManager(ctx, o.mgr)
	// and add a setter which is invoked to update the function config via
	// ctx during an API call (by calling stephttp.FnConfig)
	ctx = o.withConfigSetter(ctx, o.mgr)
	// Add a getter, allowing us to fetch config to update values (eg. in UpdateOmitResponseBody)
	ctx = o.withConfigGetter(ctx)

	if o.getExistingRun(ctx) {
		// In this case, we're re-entering an existing run, which means we're now
		// running async and are responding to an Inngest's executor call.
		//
		// In this case, we always want to start returning opcodes to the HTTP request
		// directly so that the async engine can take over.
		o.mgr.SetStepMode(sdkrequest.StepModeYield)

		// Call the handler to execute the next steps.
		_ = o.call(ctx)

		if len(o.mgr.Ops()) > 0 {
			// Write the ops to the response writer.  We know that this request is from Inngest,
			// therefore its safe to write opcodes directly to the response.
			byt, err := json.Marshal(o.mgr.Ops())
			if err != nil {
				o.provider.logger.Error("error marshalling opcodes", "error", err)
				return err
			}
			o.w.WriteHeader(206)
			_, _ = o.w.Write(byt)
		}
		return nil
	}

	// Here, we're always creating a net-new run.  Firstly, we must hit the API endpoint
	// to begin the logic and check for any function config.  This will continue to execute
	// step.run calls until either an error, an async step, or the fn finishes.
	result := o.call(ctx)

	// After calling the API, check if we have nil function config;  if so, `stephttp.FnConfig`
	// wasn't called.  In this case, use defaults.
	if o.config == nil {
		o.config = &FnOpts{
			// Use the global provider default (which may be nil)
			AsyncResponse: o.provider.opts.Optional.DefaultAsyncResponse,
		}
	}

	// Note that at this point the request would typically have finished, therefore the
	// context could be cancelled.  Stop this from breaking our API calls.
	ctx = context.WithoutCancel(ctx)

	if sdkrequest.HasAsyncOps(o.mgr.Ops(), o.run.Attempt, 0) {
		// Always checkpoint first, then handle the async conversion.
		token := o.handleFirstCheckpoint(ctx)
		return o.handleAsyncConversion(ctx, token)
	}

	// Attempt to flush the response directly to the client immediately, reducing TTFB
	// Only flush if the connection hasn't been hijacked (e.g., for WebSocket upgrades)
	if !o.w.hijacked {
		o.w.Flush()
	}

	if len(o.mgr.Ops()) == 0 && !o.provider.opts.Optional.TrackAllEndpoints {
		// If there are no steps and TrackAllEndpoints is disabled, we don't actually
		// need to do anbything.
		return nil
	}

	// In this case, the run must have finished - as no async conversion happened.
	//
	// Append the run complete result to the ops, which finalizes the run in
	// a single call.
	if err := o.appendResult(ctx, result); err != nil {
		o.provider.logger.Error("error appending run complete op",
			"error", err,
			"run_id", o.run.RunID,
		)
	}

	o.handleFirstCheckpoint(ctx)

	return nil
}

// handleAsyncConversion handles the conversion of sync -> async functions, which
// essetially means checkpointing the steps in the foreground (blocking) so that
// we can handle them with the async executor.
//
// We also need to handle the API response to our user, which is either a token,
// a redirect, or a custom response.
func (o *requestOwner) handleAsyncConversion(ctx context.Context, token string) error {
	if !sdkrequest.HasAsyncOps(o.mgr.Ops(), o.run.Attempt, 0) {
		return nil
	}

	// Then handle the response to our user.
	if o.config == nil {
		o.config = &FnOpts{
			// Use the global provider default (which may be nil)
			AsyncResponse: o.provider.opts.Optional.DefaultAsyncResponse,
		}
	}

	// Assign defaults if not set in provider config;  use redirect.
	if o.config.AsyncResponse == nil {
		o.config.AsyncResponse = AsyncResponseRedirect{}
	}

	var url string

	switch v := o.config.AsyncResponse.(type) {
	case AsyncResponseToken:
		return json.NewEncoder(o.w).Encode(asyncResponseToken{
			RunID: o.run.RunID,
			Token: token,
		})
	case AsyncResponseCustom:
		v(o.w, o.r)
		return nil
	case AsyncResponseRedirect:
		if v.URL != nil {
			url = v.URL(o.run.RunID, token)
		}
	}

	if url == "" {
		url = defaultRedirectURL(o.provider.opts, o.run.RunID, token)
	}

	http.Redirect(o.w, o.r, url, http.StatusSeeOther)
	return nil
}

func (o *requestOwner) getExistingRun(ctx context.Context) bool {
	// Validate signature and extract run information
	if !validateResumeRequestSignature(ctx, o.r, o.provider.opts.signingKey(), o.provider.opts.signingKeyFallback()) {
		return false
	}

	// Extract headers after validation passes
	runID, err := ulid.Parse(o.r.Header.Get(headerRunID))
	if err != nil {
		return false
	}

	o.run.RunID = runID
	o.run.Signature = o.r.Header.Get(headerSignature)

	// XXX: Use V2 API when created.
	steps, err := o.provider.api.GetSteps(ctx, o.run.RunID)
	if err != nil {
		return false
	}

	// This is now always async.
	o.mgr.SetSteps(steps)
	o.mgr.SetStepMode(sdkrequest.StepModeYield)

	// XXX: When using the V2 API, we should update o.run with the new run context.

	return true
}

// call initializes the hijacking control flow, then executes the API-based Inngest function.
// Depending on the step mode, this may execute all steps or execute a single step then halt
// once the step finishes.
//
// It is the callers responsibility to handle the generated opcodes added to the invocation
// manager.
func (o *requestOwner) call(ctx context.Context) APIResult {
	var panicErr error

	defer func() {
		if r := recover(); r != nil {
			callCtx := o.mgr.CallContext()

			// Was this us attepmting to prevent functions from continuing, using
			// panic as a crappy control flow because go doesn't have generators?
			if _, ok := r.(sdkrequest.ControlHijack); ok {
				// Step attempt ended (completed or errored).
				//
				// NOTE: In this case, for API-based functions, we only get ControlHijack
				// panics when we need to checkpoint via a blocking call.
				//
				// For example, when you `step.sleep` or `step.waitForEvent`, the function
				// turns from a synchronous API to an asynchronous background function
				// automatically.
				o.mgr.SetStepMode(sdkrequest.StepModeYield)
				o.provider.mw.AfterExecution(ctx, callCtx, nil, nil)
				return
			}

			// TODO: How many retries does this function have?  If zero, we can ignore
			// any retries and show the error directly to the user, keeping StepModeBackground
			// checkpointing.

			panicStack := string(debug.Stack())
			panicErr = fmt.Errorf("function panicked: %v.  stack:\n%s", r, panicStack)

			o.provider.mw.AfterExecution(ctx, callCtx, nil, nil)
			o.provider.mw.OnPanic(ctx, callCtx, r, panicStack)
		}
	}()

	// Execute the handler with step tooling available (o.w is already wrapped)
	o.next(o.w, o.r.WithContext(ctx))
	duration := time.Since(o.startTime)

	result := APIResult{
		StatusCode: o.w.statusCode,
		Headers:    flattenHeaders(o.w.Header()),
		Body:       o.w.body.Bytes(),
		Duration:   duration,
	}

	if panicErr != nil {
		result.Error = panicErr.Error()
	}
	if o.mgr.Err() != nil {
		result.Error = o.mgr.Err().Error()
	}

	return result
}

// handleFirstCheckpoint creates a new run with the given request information.
//
// This automatically upserts the requried apps and functions via the same API
// request whilst creating a new run.
//
// It also checkpoints the first N steps (potentially including the entire function).
//
// This is a blocking operation;  to run this in the background use a goroutine.
//
// This returns an optional token used when redirecting to async outputs.
func (o *requestOwner) handleFirstCheckpoint(ctx context.Context) string {
	var (
		requestBody []byte
		err         error
	)

	// Only read the request body if the config specifies so.
	if o.config == nil || !o.config.OmitRequestBody {
		requestBody, err = readRequestBody(o.r)
		if err != nil {
			if errors.Is(err, http.ErrBodyReadAfterClose) {
				o.provider.logger.Warn("attempted to read request body twice")
			} else {
				o.provider.logger.Error("error reading request body creating new run", "error", err)
			}
		}

		// TODO: End to end encryption, if enabled.
	}

	// Create new API-based run in a goroutine.  This can always happen in the background whilst
	// the API is executing.
	//
	// Note that it is important that this finishes before we begin to checkpoint step data.
	scheme := httputil.GetScheme(o.r)

	// fnID is the optional function slug to use.  If this is undefined, a slug will be generated
	// using the URL and method directly in our API.
	fnID := ""
	if o.config != nil {
		fnID = o.config.ID
	}

	resp, err := o.provider.api.CheckpointNewRun(ctx, o.run.RunID, NewAPIRunData{
		Domain:      scheme + "://" + o.r.Host,
		Method:      o.r.Method,
		Path:        o.r.URL.Path,
		IP:          getClientIP(o.r),
		ContentType: o.r.Header.Get("Content-Type"),
		QueryParams: o.r.URL.RawQuery,
		Body:        requestBody,
		Fn:          fnID,
	}, o.mgr.Ops()...)
	if err != nil {
		o.provider.logger.Error("error creating new api-based inngest run", "error", err, "run_id", o.run.RunID)
		return ""
	}

	o.run = *resp
	return resp.Token
}

// validateResumeRequestSignature validates the signature for resume requests.
// The signature is computed over the X-Run-ID header value, not the request body.
// Returns true if validation passes or if in dev mode, false otherwise.
func validateResumeRequestSignature(ctx context.Context, r *http.Request, signingKey, signingKeyFallback string) bool {
	// Skip validation in dev mode
	if env.IsDev() {
		return true
	}

	// Extract required headers
	signatureHeader := r.Header.Get(headerSignature)
	runIDHeader := r.Header.Get(headerRunID)

	// Require both headers in non-dev mode
	if signatureHeader == "" || runIDHeader == "" {
		return false
	}

	// Validate signature with primary and fallback keys using the run ID as the payload
	valid, _, err := inngestgo.ValidateRequestSignature(
		ctx,
		signatureHeader,
		signingKey,
		signingKeyFallback,
		[]byte(runIDHeader),
		false, // not dev mode
	)
	if err != nil {
		return false
	}

	return valid
}

func (o *requestOwner) appendResult(ctx context.Context, res APIResult) error {
	// When appending API results, never yield (panic).
	o.mgr.SetStepMode(sdkrequest.StepModeManual)

	defer func() {
		// Always ignore any control hijacks, just in case.
		if r := recover(); r != nil {
			if _, ok := r.(sdkrequest.ControlHijack); ok {
				return
			}
			// Repanic.
			panic(r)
		}
	}()

	var (
		responseBody []byte
		err          error
	)

	if o.config == nil || !o.config.OmitResponseBody {
		// Append the fn complete opcode.
		responseBody, err = json.Marshal(map[string]any{"data": res})
		if err != nil {
			return err
		}
	}

	op := sdkrequest.GeneratorOpcode{
		ID:   o.mgr.NewOp(enums.OpcodeRunComplete, "complete").MustHash(),
		Op:   enums.OpcodeRunComplete,
		Data: responseBody,
	}

	o.mgr.AppendOp(ctx, op)
	return nil
}

// withConfigSetter allows a caller to update the request's function config from a nested
// call via ctx.
func (o *requestOwner) withConfigSetter(ctx context.Context, mgr sdkrequest.InvocationManager) context.Context {
	return context.WithValue(ctx, fnSetterCtx, func(cfg FnOpts) {
		o.setConfig(cfg, mgr)
	})
}

func (o *requestOwner) withConfigGetter(ctx context.Context) context.Context {
	return context.WithValue(ctx, fnGetterCtx, func() FnOpts {
		if o.config == nil {
			o.config = &FnOpts{}
		}
		return *o.config
	})
}

func (o *requestOwner) setConfig(cfg FnOpts, mgr sdkrequest.InvocationManager) {
	o.config = &cfg
	// Set the servable function in our manager now that it exists.
	mgr.SetFn(servableRestFn{cfg})
}
