package step

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngestgo/errors"
)

type InferFormat string

const (
	// FormatOpenAIChat represents the default OpenAI chat completion request.
	InferFormatOpenAIChat InferFormat = "openai-chat"
	InferFormatAnthropic  InferFormat = "anthropic"
	InferFormatGemini     InferFormat = "gemini"
	InferFormatBedrock    InferFormat = "bedrock"
)

// StepRun runs any code reliably, with retries, returning the resulting data.  If this
// fails the function stops.
func Infer[InputT any, OutputT any](
	ctx context.Context,
	id string,
	in InferOpts[InputT],
) (out OutputT, err error) {

	mgr := preflight(ctx)
	op := mgr.NewOp(enums.OpcodeAIGateway, id, nil)
	hashedID := op.MustHash()

	if val, ok := mgr.Step(op); ok {
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
				_ = json.Unmarshal(err.Data, out)
				return out, err
			}
			// If there's an error, assume that val is already of type T without wrapping
			// in the 'data' object as per the SDK spec.  Here, if this succeeds we can be
			// sure that we're wrapping the data in a compliant way.
			if len(unwrapped.Data) > 0 {
				val = unwrapped.Data
			}
		}

		// If we're not unmarshalling, return the raw data.  This uses some type foo to make
		// things work correctly.
		v := reflect.New(reflect.TypeOf(out)).Interface()
		switch v.(type) {
		case json.RawMessage:
			val, _ := reflect.ValueOf(unwrapped.Data).Elem().Interface().(OutputT)
			return val, nil
		case []byte:
			val, _ := reflect.ValueOf([]byte(unwrapped.Data)).Elem().Interface().(OutputT)
			return val, nil
		case string:
			val, _ := reflect.ValueOf(string(unwrapped.Data)).Elem().Interface().(OutputT)
			return val, nil
		}

		// Check to see if we were passed a pointer or not. If not, we must make this a pointer.
		if reflect.TypeOf(out).Kind() != reflect.Ptr {
			v := reflect.New(reflect.TypeOf(out)).Interface()
			err := json.Unmarshal(val, v)
			return reflect.ValueOf(v).Elem().Interface().(OutputT), err
		}

		// NOTE: API responses may change, so return both the val and the error.
		v = reflect.New(reflect.TypeOf(out).Elem()).Interface()

		err := json.Unmarshal(val, v)
		res := reflect.ValueOf(v).Interface()
		out, _ = res.(OutputT)

		return out, err
	}

	reqBytes, err := json.Marshal(in.Body)
	if err != nil {
		mgr.SetErr(fmt.Errorf("error unmarshalling state for step '%s': %w", id, err))
		panic(ControlHijack{})
	}

	mgr.AppendOp(state.GeneratorOpcode{
		ID:   hashedID,
		Op:   enums.OpcodeAIGateway,
		Name: id,
		Opts: inferOpcodeOpts{
			URL:     in.Opts.URL,
			Headers: in.Opts.Headers,
			AuthKey: in.Opts.AuthKey,
			Format:  in.Opts.Format,
			Body:    in.Body,
		},
		Data: reqBytes,
	})
	panic(ControlHijack{})
}

type inferOpcodeOpts struct {
	// URL is the provider URL which is used when making the request.
	URL string `json:"url"`
	// Headers represent additional headers to send in the request.
	Headers map[string]string `json:"headers,omitempty"`
	// AuthKey is your API key.  This will be added to the inference request depending
	// on the API format chosen in Format.
	//
	// This is NEVER logged or kept.
	AuthKey string `json:"auth_key"`
	// Format represents the format for the API request and response.  Infer allows
	// the use of common formats, and we create the request and infer metadata based
	// off of the API format.  Note that many providers support an open OpenAI-like
	// format.
	Format InferFormat `json:"format"`
	// Body represents the raw request.
	Body any `json:"body"`
}

type InferRequestOpts struct {
	// URL is the provider URL which is used when making the request.
	URL string `json:"url"`
	// Headers represent additional headers to send in the request.
	Headers map[string]string `json:"headers,omitempty"`
	// AuthKey is your API key.  This will be added to the inference request depending
	// on the API format chosen in Format.
	//
	// This is NEVER logged or kept.
	AuthKey string `json:"auth_key"`
	// Format represents the format for the API request and response.  Infer allows
	// the use of common formats, and we create the request and infer metadata based
	// off of the API format.  Note that many providers support an open OpenAI-like
	// format.
	Format InferFormat `json:"format"`
	// AutoToolCall bool `json:"auto_tool_call"`
}

type InferOpts[RequestT any] struct {
	// Opts represents the Inngest-specific step and request opts
	Opts InferRequestOpts
	// Body is the raw request type, eg. the Anthropic or OpenAI request.
	Body RequestT
}

// InferOpenAIOpts is a helper function for generating OpenAI opts.
func InferOpenAIOpts(key *string, baseURL *string) InferRequestOpts {
	api := os.Getenv("OPENAI_API_KEY")
	if key != nil {
		api = *key
	}

	base := "https://api.openai.com"
	if baseURL != nil {
		base = *baseURL
	}

	return InferRequestOpts{
		URL:     base + "/v1/chat/completions",
		AuthKey: api,
		Format:  InferFormatOpenAIChat,
	}
}
