package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	sdkerrors "github.com/inngest/inngestgo/errors"
	"github.com/inngest/inngestgo/internal/connectworker"
	"github.com/inngest/inngestgo/internal/sdkrequest"
)

// invokerHandler serves each request with the FunctionInvoker registered for
// the request's app.
//
// this is used during regular Connect() execution and calls functions directly
// from inbound requests.
type invokerHandler struct {
	invokers   map[string]FunctionInvoker
	opts       Opts
	logger     *slog.Logger
	httpClient *http.Client
}

// Admin does some basic checks to ensure the functtion exists for the given request
func (i invokerHandler) Admit(req *connectproto.GatewayExecutorRequestData) error {
	if req.AppName == "" {
		return fmt.Errorf("missing app name in executor request")
	}
	if _, ok := i.invokers[req.AppName]; !ok {
		return fmt.Errorf("no invoker for app name %q", req.AppName)
	}
	return nil
}

// Done is a noop with regular connect
func (i invokerHandler) Done(*connectproto.GatewayExecutorRequestData) {}

func (i invokerHandler) Serve(_ context.Context, req *connectproto.GatewayExecutorRequestData) (connectworker.Response, error) {
	l := i.logger.With(
		"request_id", req.RequestId,
		"run_id", req.RunId,
		"app_id", req.AppId,
		"function_slug", req.FunctionSlug,
	)
	if req.JobId != "" {
		l = l.With("job_id", req.JobId)
	}

	invoker, ok := i.invokers[req.AppName]
	if !ok {
		return connectworker.Response{}, fmt.Errorf("no invoker for app name %q", req.AppName)
	}

	// Note: This still uses JSON
	// TODO Replace with Protobuf
	var request sdkrequest.Request
	err := json.Unmarshal(req.RequestPayload, &request)
	if err != nil {
		l.Error("error decoding sdk request", "error", err)
		err = fmt.Errorf("invalid SDK request payload: %w", err)
	}
	request.CallCtx.RequestID = req.RequestId
	request.CallCtx.JobID = req.JobId

	var stepId *string
	if req.StepId != nil && *req.StepId != "step" {
		stepId = req.StepId
	}

	// Invoke function, always complete regardless of connection cancellation once
	// the request has been acknowledged.
	var (
		resp any
		ops  []sdkrequest.GeneratorOpcode
	)
	if err == nil && request.UsesAPI() {
		loadOpts := sdkrequest.LoadFromAPIOpts{
			APIBaseURL:        i.opts.APIBaseURL,
			AuthToken:         string(i.opts.HashedSigningKey),
			AuthTokenFallback: string(i.opts.HashedSigningKeyFallback),
			HTTPClient:        i.httpClient,
		}
		err = sdkrequest.LoadFromAPI(context.Background(), &request, loadOpts)
		if err != nil {
			err = fmt.Errorf("error loading function state from API: %w", err)
		}
	}
	if err == nil {
		resp, ops, err = invoker.InvokeFunction(context.Background(), req.FunctionSlug, stepId, request)
	}

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

	// These may be added even for 2xx codes with step errors.
	var retryAfterVal *string
	if retryAt != nil {
		formatted := retryAt.Format(time.RFC3339)
		retryAfterVal = &formatted
	}

	if err != nil {
		l.Error("error calling function", "error", err)
		return connectworker.Response{
			Status:         connectproto.SDKResponseStatus_ERROR,
			Body:           []byte(fmt.Sprintf("error calling function: %s", err.Error())),
			NoRetry:        noRetry,
			RetryAfter:     retryAfterVal,
			SDKVersion:     fmt.Sprintf("%s:v%s", i.opts.SDKLanguage, i.opts.SDKVersion),
			RequestVersion: 0, // Go SDK currently only supports v0
		}, nil
	}

	if len(ops) > 0 {
		// Note: This still uses JSON
		// TODO Replace with Protobuf
		serializedOps, err := json.Marshal(ops)
		if err != nil {
			return connectworker.Response{}, fmt.Errorf("could not serialize ops: %w", err)
		}

		// Return the function opcode returned here so that we can re-invoke this
		// function and manage state appropriately.  Any opcode here takes precedence
		// over function return values as the function has not yet finished.
		return connectworker.Response{
			Status:     connectproto.SDKResponseStatus_NOT_COMPLETED,
			Body:       serializedOps,
			NoRetry:    noRetry,
			RetryAfter: retryAfterVal,
		}, nil
	}

	// Note: This still uses JSON
	// TODO Replace with Protobuf
	serializedResp, err := json.Marshal(resp)
	if err != nil {
		return connectworker.Response{}, fmt.Errorf("could not serialize resp: %w", err)
	}

	// Return the function response.
	return connectworker.Response{
		Status:     connectproto.SDKResponseStatus_DONE,
		Body:       serializedResp,
		NoRetry:    noRetry,
		RetryAfter: retryAfterVal,
	}, nil
}
