package httpv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	state "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util/errs"
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
	Type string

	// URL is the URL that we're hitting.
	URL url.URL

	// Method ios the optional HTTP method to use when hitting the URL.  This is
	// only used in sync functions so that we can resume any specific sync API.
	//
	// For async functions, this is always a POST request.
	Method string
}

func (d httpv2) Name() string {
	return "httpv2"
}

// Do executes the function via an HTTP request.
func (d httpv2) Do(ctx context.Context, sl sv2.StateLoader, opts driver.V2RequestOpts) (*state.DriverResponse, errs.UserError, errs.InternalError) {
	typ, _ := opts.Fn.Driver.Metadata["type"].(string)
	if typ == "sync" {
		return d.sync(ctx, opts)
	}
	return d.async(ctx, opts)
}

// sync re-enters synchronous functions, allowing regular API endpoints to be resumed as if they're
// generic async functions.
//
// sync entry is relatively simple: we re-execute a specific API request, and we add Inngest-specific
// headers to the request.  The SDK will then fetch the requisite function state such that it can resume
// where it left off.
func (d httpv2) sync(ctx context.Context, opts driver.V2RequestOpts) (*state.DriverResponse, errs.UserError, errs.InternalError) {
	sig := Sign(ctx, opts.SigningKey, opts.Metadata.ID.RunID[:])

	method := http.MethodPost
	if m, _ := opts.Fn.Driver.Metadata["method"].(string); m != "" {
		method = m
	}

	req, err := exechttp.NewRequest(
		method,
		opts.Fn.Driver.URI,
		nil,
	)
	if err != nil {
		return nil, nil, errs.Wrap(0, true, "error creating request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Inngest-Signature", sig)
	req.Header.Add("X-Run-ID", opts.Metadata.ID.RunID.String())

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
	ops, userErr := parseOpcodes(resp.Body)
	if userErr != nil {
		return nil, userErr, nil
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

func parseOpcodes(byt []byte) ([]*state.GeneratorOpcode, errs.UserError) {
	gen := []*state.GeneratorOpcode{}
	if err := json.Unmarshal(byt, &gen); err != nil {
		return nil, errs.WrapUser(0, true, "error reading SDK responses as steps: %w", err)
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
