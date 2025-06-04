package step

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo/errors"
	"github.com/inngest/inngestgo/internal/sdkrequest"
)

// FetchOpts configures the HTTP call made from Inngest.  Note that the HTTP call
// respects all step limits:  the response will be capped at the maximum step size,
// and the HTTP request will terminate after the maximum step timeout.
//
// These are currently 4MB and 2 hours respectively, but may change without SDK version
// updates on the execution side.
type FetchOpts struct {
	// URL is the full endpoint that we're sending the request to.  This must
	// always be provided by our SDKs.
	URL string `json:"url,omitempty"`
	// Headers represent additional headers to send in the request.
	Headers map[string]string `json:"headers,omitempty"`
	// Body indicates the raw content of the request, as a slice of JSON bytes.
	// It's expected that this comes from our SDKs directly.
	Body string `json:"body"`
	// Method is the HTTP method to use for the request.  This is almost always
	// POST for AI requests, but can be specified too.
	Method string `json:"method,omitempty"`
}

type FetchResponse[InputT any] struct {
	Body       InputT            `json:"body"`
	Headers    map[string]string `json:"headers"`
	StatusCode int               `json:"status_code"`
	URL        string            `json:"url"`
}

// Fetch offloads the request to Inngest and continues execution with the response when complete
func Fetch[OutputT any](
	ctx context.Context,
	id string,
	in FetchOpts,
) (out FetchResponse[OutputT], err error) {
	mgr := preflight(ctx)
	op := mgr.NewOp(enums.OpcodeGateway, id, nil)
	hashedID := op.MustHash()

	if val, ok := mgr.Step(ctx, op); ok {
		// This step has already ran as we have state for it. Unmarshal the JSON into type T
		unwrapped := response{}

		if err := json.Unmarshal(val, &unwrapped); err == nil {
			// Check for step errors first.
			if len(unwrapped.Error) > 0 {
				err := errors.StepError{}
				if err := json.Unmarshal(unwrapped.Error, &err); err != nil {
					mgr.SetErr(fmt.Errorf("error unmarshalling error for step '%s': %w", id, err))
					panic(ControlHijack{})
				}

				// See if we have any data for multiple returns in the error type.
				_ = json.Unmarshal(err.Data, &out)
				return out, err
			}
			// If there's an error, assume that val is already of type T without wrapping
			// in the 'data' object as per the SDK spec.  Here, if this succeeds we can be
			// sure that we're wrapping the data in a compliant way.
			if len(unwrapped.Data) > 0 {
				val = unwrapped.Data
			}
		}

		// NOTE: The executor ALWAYS embeds the actual FetchResponse in type Data.
		err := json.Unmarshal(val, &out)
		return out, err
	}

	mgr.AppendOp(sdkrequest.GeneratorOpcode{
		ID:   hashedID,
		Op:   enums.OpcodeGateway,
		Name: id,
		Opts: in,
	})
	panic(ControlHijack{})
}
