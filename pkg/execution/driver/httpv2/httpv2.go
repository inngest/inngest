package httpv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"

	apiv1 "github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	state "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util/errs"
	inngestgo "github.com/inngest/inngestgo"
)

func NewDriver(client exechttp.RequestExecutor) driver.DriverV2 {
	return &httpv2{
		Client: client,
	}
}

// The HTTPV2 driver is the new driver for HTTP-based step invocation.
//
// This properly handles both sync and async functions, including re-entry
// into API-based sync functions using any HTTP method.
type httpv2 struct {
	// Client represents an http client used to create outgoing requests.
	Client exechttp.RequestExecutor
}

type HTTPV2Config struct {
	// Type represents whether this function is registered as a "sync" or "async"
	// function.
	//
	// This is important:  sync functions never have state sent to them, whereas
	// async functions can have a specific amount of state sent to them to
	// initialize the run re-entry.
	Type string `json:"type"`

	// Method ios the optional HTTP method to use when hitting the URL.  This is
	// only used in sync functions so that we can resume any specific sync API.
	//
	// For async functions, this is always a POST request.
	Method string `json:"method"`
}

func (d httpv2) Name() string {
	return "httpv2"
}

// Do executes the function via an HTTP request.
func (d httpv2) Do(ctx context.Context, sl sv2.StateLoader, opts driver.V2RequestOpts) (*state.DriverResponse, errs.UserError, errs.InternalError) {
	typ, _ := opts.Fn.Driver.Metadata["type"].(string)
	if typ == "sync" {
		return d.sync(ctx, sl, opts)
	}
	return d.async(ctx, opts)
}

// sync re-enters synchronous functions, allowing regular API endpoints to be resumed as if they're
// generic async functions.
//
// sync entry is relatively simple: we re-execute a specific API request, and we add Inngest-specific
// headers to the request.  The SDK will then fetch the requisite function state such that it can resume
// where it left off.
// loadHTTPRequestEvent attempts to load the triggering event from state and parse
// it as an HTTP request event. Returns nil if loading fails, the event name doesn't
// match, or the event can't be parsed.
func loadHTTPRequestEvent(ctx context.Context, sl sv2.StateLoader, id sv2.ID) *inngestgo.GenericEvent[apiv1.NewAPIRunData] {
	if sl == nil {
		return nil
	}
	rawEvts, err := sl.LoadEvents(ctx, id)
	if err != nil || len(rawEvts) == 0 {
		return nil
	}

	evt := &inngestgo.GenericEvent[apiv1.NewAPIRunData]{}
	if err := json.Unmarshal(rawEvts[0], evt); err != nil {
		return nil
	}
	if evt.Name != consts.HttpRequestName {
		return nil
	}
	return evt
}

func (d httpv2) sync(ctx context.Context, sl sv2.StateLoader, opts driver.V2RequestOpts) (*state.DriverResponse, errs.UserError, errs.InternalError) {
	method := http.MethodPost
	if m, _ := opts.Fn.Driver.Metadata["method"].(string); m != "" {
		method = m
	}

	// Attempt to load the original HTTP request data from the triggering event.
	var (
		body        json.RawMessage
		contentType = "application/json"
	)
	url := opts.URL

	if evt := loadHTTPRequestEvent(ctx, sl, opts.Metadata.ID); evt != nil {
		if len(evt.Data.Body) > 0 {
			body = json.RawMessage(evt.Data.Body)
		}
		if evt.Data.ContentType != "" {
			contentType = evt.Data.ContentType
		}
		if evt.Data.QueryParams != "" {
			parsed, err := neturl.Parse(url)
			if err == nil {
				existing := parsed.RawQuery
				if existing != "" {
					parsed.RawQuery = existing + "&" + evt.Data.QueryParams
				} else {
					parsed.RawQuery = evt.Data.QueryParams
				}
				url = parsed.String()
			}
		}
	}

	sig := Sign(ctx, opts.SigningKey, body)

	req, err := exechttp.NewRequest(
		method,
		url,
		body,
	)
	if err != nil {
		return nil, nil, errs.Wrap(0, true, "error creating request: %w", err)
	}
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("X-Inngest-Signature", sig)
	req.Header.Add("X-Run-ID", opts.Metadata.ID.RunID.String())
	req.Header.Add(headers.HeaderKeyRequestVersion, fmt.Sprintf("%d", opts.Metadata.Config.RequestVersion))

	if opts.Metadata.Config.ForceStepPlan {
		req.Header.Add(headers.HeaderKeyForceStepPlan, "true")
	}

	if opts.StepID != nil && *opts.StepID != "" && *opts.StepID != "step" {
		req.Header.Add(headers.HeaderInngestStepID, *opts.StepID)
	}

	resp, err := d.Client.DoRequest(ctx, req)

	if errors.Is(err, exechttp.ErrBodyTooLarge) {
		// This is a user error.
		return nil, errs.WrapUser(0, false, "SDK response too large: %w", err), nil
	}
	if errors.Is(err, exechttp.ErrUnableToReach) {
		// This is an internal error.
		return nil, nil, errs.Wrap(0, true, "Unable to reach SDK: %w", err)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, nil, errs.Wrap(0, true, "Unable to reach SDK: %w", err)
	}
	if err != nil {
		// Unknown errors here should always be internal, as user errors should be handled above.
		return nil, nil, errs.Wrap(0, true, "error executing request: %w", err)
	}
	if resp == nil {
		// err should always be non-nil in this case, so we should never hit this particular code
		// path but do this to prevent nil pointer references.
		return nil, nil, errs.Wrap(0, true, "nil response from sdk: %w", err)
	}

	// We must also assert that we had an Inngest-specific response.
	if !headers.IsSDK(resp.Header) {
		return nil, errs.WrapResponseAsUser(0, true, resp.Body, "didn't receive SDK response: %w", err), nil
	}

	// We always expect opcodes from the API endpoint.  Whenever we re-enter a sync function,
	// the API becomes, to effect, an async function and each HTTP request we make should always
	// result in well-formed ops.
	ops, userErr := parseOpcodes(resp.Body, resp.StatusCode)
	if userErr != nil {
		// Return a DriverResponse with the HTTP response data so the executor
		// can detect the error. Without this, a nil response gets converted to
		// an empty DriverResponse{} with StatusCode: 0 and Err: nil, causing
		// the executor to treat error responses (like function-rejected 400)
		// as successful completions.
		r := &state.DriverResponse{
			StatusCode:     resp.StatusCode,
			SDK:            resp.Header.Get(headers.HeaderKeySDK),
			Header:         resp.Header,
			Duration:       resp.StatResult.Total,
			RetryAt:        headers.RetryAfter(resp.Header),
			NoRetry:        headers.NoRetry(resp.Header),
			RequestVersion: headers.RequestVersion(resp.Header),
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			str := fmt.Sprintf("invalid status code: %d", resp.StatusCode)
			r.Err = &str
		}
		if r.NoRetry {
			str := "NonRetriableError"
			r.Err = &str
		}
		return r, userErr, nil
	}

	r := &state.DriverResponse{
		Generator:      ops,
		StatusCode:     resp.StatusCode,
		SDK:            resp.Header.Get(headers.HeaderKeySDK),
		Header:         resp.Header,
		Duration:       resp.StatResult.Total,
		RetryAt:        headers.RetryAfter(resp.Header),
		NoRetry:        headers.NoRetry(resp.Header),
		RequestVersion: headers.RequestVersion(resp.Header),

		Step:       inngest.Step{}, // TODO: Deprecate.  Not needed;  use opcodes.
		Output:     nil,            // TODO: Deprecate, use opcodes.
		OutputSize: 0,              // TODO: Deprecate, use opcodes.
		UserError:  nil,            // TODO: Deprecate, use opcodes.
		Err:        nil,            // TODO: Deprecate, use InternalError response
	}

	return r, nil, nil
}

func (d httpv2) async(ctx context.Context, opts driver.V2RequestOpts) (*state.DriverResponse, errs.UserError, errs.InternalError) {
	// When all SDKs implemnent RunFinished opcodes, and so on, we can essentially move to this
	// V2 driver as soon as they update their run config.
	//
	// For now, this is disabled.
	return nil, nil, errs.Wrap(0, false, "async v2 http driver not implemneted")
}

func parseOpcodes(byt []byte, status int) ([]*state.GeneratorOpcode, errs.UserError) {
	gen := []*state.GeneratorOpcode{}
	if err := json.Unmarshal(byt, &gen); err != nil {
		// TODO: ADD UNIT TESTS ASSERTING THAT THE USER ERROR CONTAINS OUR RESPONSE BODY.
		return nil, NewNonGeneratorError(byt, status)
	}

	if len(gen) == 0 {
		// Finally, if the length of resp.Generator == 0 then this is implicitly an
		// enums.OpcodeNone step.  This is added to reduce bandwidth across many
		// calls and normalize the response, such that a non-generator is
		// represented as an empty array.
		gen = []*state.GeneratorOpcode{{
			Op: enums.OpcodeNone,
		}}
	}

	// Check every op we've parsed, making sure it adheres to any limits we're
	// enforcing
	for _, op := range gen {
		if err := op.Validate(); err != nil {
			err = fmt.Errorf("error validating generator opcode %s: %w", op.ID, err)
			return nil, errs.WrapUser(0, false, "invalid opcode: %w", err)
		}

		// XXX: We may need to add `NoRetry` to `op.Error` if `item.Op == enums.OpcodeStepError`.
	}

	return gen, nil
}
