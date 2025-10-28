package stephttp

import (
	"context"
	"net/http"

	"github.com/inngest/inngestgo/internal/fn"
	"github.com/oklog/ulid/v2"
)

type AsyncResponse interface {
	isAsyncResponse()
}

// FnOpts allows you to define function configuration options for your API-based
// Inngest function.
type FnOpts struct {
	// ID represents the function ID.  You should always set this, and must
	// set this when the URL contains values or identifiers, eg /users/123.
	ID string

	// Retries indicates the number of retries in each step.  By default,
	// for API-based synchronous functions this is zero.
	//
	// Note that retries will happen async, and the API will respond based off
	// of the AsyncResponse type defined in the function configuration.
	// Retries int32

	// OmitRequestBody prevents the incoming request from being stored every time.
	OmitRequestBody bool

	// OmitResponseBody prevents the API response from being stored every time.
	//
	// Note that you can override this by calling `stephttp.UpdateOmitResponseBody(ctx, bool)`
	// before the API responds to dynamically adjust whether the response is stored. This
	// allows you to properly store eg. errors for debugging, then only omit successful
	// responses at the end of your function.
	OmitResponseBody bool

	// AsyncResponse determines how we respond to a user when an API hits an
	// async step (eg. step.sleep, step.waitForEvent) or if a step errors.
	//
	// Mot of the time, AsyncResponseRedirect allows for seamless handling of
	// step errors and sleeping steps.
	//
	// For more information, see the Inngest docs.
	//
	// By default, this uses the AsyncResponseRedirect response type, redirecting
	// the user to a URL that will block and return the API result automatically
	// once the API finishes.
	AsyncResponse AsyncResponse
}

func Configure(ctx context.Context, opts FnOpts) {
	if _, ok := ctx.Value(fnConfigCtx).(FnOpts); ok {
		// Here, we should ideally warn, if we have a logger available.
		return
	}

	if set, ok := ctx.Value(fnSetterCtx).(func(FnOpts)); ok {
		set(opts)
	}
}

// UpdateOmitResponseBody sets whether the response body will be tracked in logs and traces.
// You can call this at any time before sending the API response and this will be respected.
func UpdateOmitResponseBody(ctx context.Context, to bool) {
	cfg := configFromContext(ctx)
	cfg.OmitResponseBody = to
	Configure(ctx, cfg)
}

func configFromContext(ctx context.Context) FnOpts {
	if get, ok := ctx.Value(fnGetterCtx).(func() FnOpts); ok {
		return get()
	}
	return FnOpts{}
}

// AsyncResponseRedirect redirects the user to a URL which will block until the async
// function completes, then return the output to the user.
//
// This is seamless, and as long as the function executes within a specific period the
// user will not know that the API finished asynchronously.
type AsyncResponseRedirect struct {
	// URL is the optional URL to redirect to.  If empty, we will redirect to
	// the Inngest API which will automatically block until the function completes.
	//
	// Note that this accepts a token which can be used to hit the Inngest API to
	// block for the API result.
	URL func(runID ulid.ULID, token string) string
}

func (a AsyncResponseRedirect) isAsyncResponse() {}

// AsyncResponseCustom allows you to configure custom HTTP responses when a sync
// function hits an async step.
//
// When you use eg. `step.sleep`, the function pauses and you must return a response
// to the user.
type AsyncResponseCustom http.HandlerFunc

func (AsyncResponseCustom) isAsyncResponse() {}

// AsyncResponseToken responds with the Token format, allowing clients to
// wait for the function to finish by hitting our V2 API with the given token.
type AsyncResponseToken struct{}

func (a AsyncResponseToken) isAsyncResponse() {}

// asyncResponseToken is the response type that we return in the AsyncResponseToken
// response method.
type asyncResponseToken struct {
	RunID ulid.ULID `json:"run_id"`
	Token string    `json:"token"`
}

//
// context setters and getters
//

type fnConfigKeyType string

const (
	fnConfigCtx = fnConfigKeyType("inngest-fn-opts")
	fnSetterCtx = fnConfigKeyType("inngest-fn-setter")
	fnGetterCtx = fnConfigKeyType("inngest-fn-getter")
)

type servableRestFn struct {
	opts FnOpts
}

func (f servableRestFn) FullyQualifiedID() string {
	return f.opts.ID
}

func (f servableRestFn) ID() string {
	return f.opts.ID
}

// Name returns the function name.
func (f servableRestFn) Name() string {
	return f.opts.ID
}

func (f servableRestFn) Config() fn.FunctionOpts {
	return fn.FunctionOpts{
		ID: f.opts.ID,
	}
}

// Trigger returns the event names or schedules that triggers the function.
func (f servableRestFn) Trigger() fn.Triggerable {
	return nil
}

// ZeroEvent returns the zero event type to marshal the event into, given an
// event name.
func (f servableRestFn) ZeroEvent() any {
	return nil
}

// Func returns the SDK function to call.  This must alawys be of type SDKFunction,
// but has an any type as we register many functions of different types into a
// type-agnostic handler; this is a generic implementation detail, unfortunately.
func (f servableRestFn) Func() any {
	return nil
}
