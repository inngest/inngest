package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/publicerr"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	sdkerrors "github.com/inngest/inngestgo/errors"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"google.golang.org/protobuf/proto"
	"time"
)

const (
	ResponseAcknowlegeDeadline = time.Second * 5
)

func (h *connectHandler) handleInvokeMessage(ctx context.Context, preparedConn *connection, msg *connectproto.ConnectMessage) error {
	resp, err := h.connectInvoke(ctx, preparedConn, msg)
	if err != nil {
		h.logger.Error("failed to handle sdk request", "err", err)
		// TODO Should we drop the connection? Continue receiving messages?
		return fmt.Errorf("could not handle sdk request: %w", err)
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		h.logger.Error("failed to serialize sdk response", "err", err)
		// TODO This should never happen; Signal that we should retry
		return fmt.Errorf("could not serialize sdk response: %w", err)
	}

	responseMessage := &connectproto.ConnectMessage{
		Kind:    connectproto.GatewayMessageType_WORKER_REPLY,
		Payload: data,
	}

	// Add message to pending messages to ensure it is acknowledged by the gateway.
	// This is necessary because even if Write() below does not error, there's no guarantee
	// that the message was truly passed on to the executor _unless_ we receive the ack message.
	h.messageBuffer.addPending(ctx, resp, ResponseAcknowlegeDeadline)

	err = wsproto.Write(ctx, preparedConn.ws, responseMessage)
	if err != nil {
		h.logger.Error("failed to send sdk response", "err", err)

		// We received an error, the message definitely was not sent: Buffer message to retry
		h.messageBuffer.append(resp)

		return fmt.Errorf("could not send sdk response: %w", err)
	}

	return nil
}

// connectInvoke is the counterpart to invoke for connect
func (h *connectHandler) connectInvoke(ctx context.Context, preparedConn *connection, msg *connectproto.ConnectMessage) (*connectproto.SDKResponse, error) {
	body := connectproto.GatewayExecutorRequestData{}
	if err := proto.Unmarshal(msg.Payload, &body); err != nil {
		// TODO Should we send this back to the gateway?
		h.logger.Error("error decoding gateway request data", "error", err)
		return nil, fmt.Errorf("invalid gateway message data: %w", err)
	}

	if body.AppName == "" {
		return nil, fmt.Errorf("missing app name in executor request")
	}

	invoker, ok := h.invokers[body.AppName]
	if !ok {
		return nil, fmt.Errorf("no invoker for app name %q", body.AppName)
	}

	// Note: This still uses JSON
	// TODO Replace with Protobuf
	var request sdkrequest.Request
	if err := json.Unmarshal(body.RequestPayload, &request); err != nil {
		// TODO Should we send this back to the gateway? Previously this was a status code 400 public error with "malformed input"
		h.logger.Error("error decoding sdk request", "error", err)
		return nil, fmt.Errorf("invalid SDK request payload: %w", err)
	}

	ackPayload, err := proto.Marshal(&connectproto.WorkerRequestAckData{
		RequestId:      body.RequestId,
		AccountId:      body.AccountId,
		EnvId:          body.EnvId,
		AppId:          body.AppId,
		FunctionSlug:   body.FunctionSlug,
		StepId:         body.StepId,
		SystemTraceCtx: body.SystemTraceCtx,
		UserTraceCtx:   body.UserTraceCtx,
		RunId:          body.RunId,
	})
	if err != nil {
		h.logger.Error("error marshaling request ack", "error", err)
		return nil, publicerr.Error{
			Message: "malformed input",
			Status:  400,
		}
	}

	// Ack message
	// If we're shutting down (context is canceled) we will not ack, which is desired!
	if err := wsproto.Write(ctx, preparedConn.ws, &connectproto.ConnectMessage{
		Kind:    connectproto.GatewayMessageType_WORKER_REQUEST_ACK,
		Payload: ackPayload,
	}); err != nil {
		h.logger.Error("error sending request ack", "error", err)
		return nil, publicerr.Error{
			Message: "failed to ack worker request",
			Status:  400,
		}
	}

	// TODO Should we wait for a gateway response before starting to process? What if the gateway fails acking and we start too early?
	// This should not happen but could lead to double processing of the same message

	if request.UseAPI {
		// TODO: implement this
		// retrieve data from API
		// request.Steps =
		// request.Events =
		_ = 0 // no-op to avoid linter error
	}

	var stepId *string
	if body.StepId != nil && *body.StepId != "step" {
		stepId = body.StepId
	}

	// Set initial lease ID
	h.workerPool.inProgressLeasesLock.Lock()
	h.workerPool.inProgressLeases[body.RequestId] = body.LeaseId
	h.workerPool.inProgressLeasesLock.Unlock()

	defer func() {
		h.workerPool.inProgressLeasesLock.Lock()
		delete(h.workerPool.inProgressLeases, body.RequestId)
		h.workerPool.inProgressLeasesLock.Unlock()
	}()

	extendLeaseCtx, cancelExtendLeaseCtx := context.WithCancel(ctx)
	defer cancelExtendLeaseCtx()

	go func() {
		for {
			select {
			case <-extendLeaseCtx.Done():
				return
			case <-time.After(preparedConn.extendLeaseInterval):
			}

			h.workerPool.inProgressLeasesLock.Lock()
			currentLeaseID, ok := h.workerPool.inProgressLeases[body.RequestId]
			h.workerPool.inProgressLeasesLock.Unlock()

			if !ok {
				// If the lease is not found (e.g. due to being removed after a nack),
				// stop extending it.
				cancelExtendLeaseCtx()
				return
			}

			// TODO Extend once more before finishing (to ensure write reaches executor in time)
			extendPayload, err := proto.Marshal(&connectproto.WorkerRequestExtendLeaseData{
				RequestId:      body.RequestId,
				AccountId:      body.AccountId,
				EnvId:          body.EnvId,
				AppId:          body.AppId,
				FunctionSlug:   body.FunctionSlug,
				StepId:         body.StepId,
				SystemTraceCtx: body.SystemTraceCtx,
				UserTraceCtx:   body.UserTraceCtx,
				RunId:          body.RunId,
				LeaseId:        currentLeaseID,
			})
			if err != nil {
				h.logger.Error("error marshaling extend payload", "error", err)
				continue
			}

			if err := wsproto.Write(ctx, preparedConn.ws, &connectproto.ConnectMessage{
				Kind:    connectproto.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE,
				Payload: extendPayload,
			}); err != nil {
				h.logger.Error("error sending extend request", "error", err)
			}
		}
	}()

	// Invoke function, always complete regardless of
	resp, ops, err := invoker.InvokeFunction(context.Background(), body.FunctionSlug, stepId, request)

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
		h.logger.Error("error calling function", "error", err)
		return &connectproto.SDKResponse{
			RequestId:      body.RequestId,
			AccountId:      body.AccountId,
			EnvId:          body.EnvId,
			AppId:          body.AppId,
			Status:         connectproto.SDKResponseStatus_ERROR,
			Body:           []byte(fmt.Sprintf("error calling function: %s", err.Error())),
			NoRetry:        noRetry,
			RetryAfter:     retryAfterVal,
			SdkVersion:     fmt.Sprintf("%s:v%s", h.opts.SDKLanguage, h.opts.SDKVersion),
			RequestVersion: 0, // Go SDK currently only supports v0
			SystemTraceCtx: body.SystemTraceCtx,
			UserTraceCtx:   body.UserTraceCtx,
			RunId:          body.RunId,
		}, nil
	}

	if len(ops) > 0 {
		// Note: This still uses JSON
		// TODO Replace with Protobuf
		serializedOps, err := json.Marshal(ops)
		if err != nil {
			return nil, fmt.Errorf("could not serialize ops: %w", err)
		}

		// Return the function opcode returned here so that we can re-invoke this
		// function and manage state appropriately.  Any opcode here takes precedence
		// over function return values as the function has not yet finished.
		return &connectproto.SDKResponse{
			RequestId:  body.RequestId,
			EnvId:      body.EnvId,
			AppId:      body.AppId,
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
		return nil, fmt.Errorf("could not serialize resp: %w", err)
	}

	// Return the function response.
	return &connectproto.SDKResponse{
		RequestId:  body.RequestId,
		EnvId:      body.EnvId,
		AppId:      body.AppId,
		Status:     connectproto.SDKResponseStatus_DONE,
		Body:       serializedResp,
		NoRetry:    noRetry,
		RetryAfter: retryAfterVal,
	}, nil
}
