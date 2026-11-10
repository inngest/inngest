package connectworker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/publicerr"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/protobuf/proto"
)

const (
	ResponseAcknowlegeDeadline = time.Second * 5
)

// errConnectionRetired is only returned before WORKER_REQUEST_ACK is sent.
// At that point the SDK has not claimed ownership of the request, so skipping
// invocation lets the gateway/executor retry or reroute it instead of risking
// duplicate execution.
var errConnectionRetired = errors.New("connection retired")

func (h *connectHandler) handleInvokeMessage(ctx context.Context, preparedConn *connection, req *connectproto.GatewayExecutorRequestData) error {
	resp, err := h.connectInvoke(ctx, preparedConn, req)
	if err != nil {
		if errors.Is(err, errConnectionRetired) {
			// This is a pre-ACK skip. Once ACK succeeds, connectInvoke invokes
			// the function with context.Background() and must let it finish.
			logAttrs := append(preparedConn.logAttrs(), "phase", preparedConn.phase())
			h.logger.Debug(
				"skipping sdk request because connection phase does not allow request ack",
				logAttrs...,
			)
			return nil
		}
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

	return h.writeReply(ctx, preparedConn, resp, responseMessage)
}

// connectInvoke ACKs the request, extends its lease while the request handler
// serves it, and returns the reply.
func (h *connectHandler) connectInvoke(ctx context.Context, preparedConn *connection, body *connectproto.GatewayExecutorRequestData) (*connectproto.SDKResponse, error) {
	logAttrs := []any{
		"request_id", body.RequestId,
		"run_id", body.RunId,
		"app_id", body.AppId,
		"function_slug", body.FunctionSlug,
	}
	logAttrs = append(logAttrs, preparedConn.logAttrs()...)
	l := h.logger.With(logAttrs...)
	if body.JobId != "" {
		l = l.With("job_id", body.JobId)
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
		l.Error("error marshaling request ack", "error", err)
		return nil, publicerr.Wrap(err, 400, "malformed input")
	}

	if err := h.writeRequestAck(ctx, preparedConn, ackPayload, l); err != nil {
		return nil, err
	}

	// TODO Should we wait for a gateway response before starting to process? What if the gateway fails acking and we start too early?
	// This should not happen but could lead to double processing of the same message

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
			if !canExtendRequestLease(preparedConn) {
				// ACKed work remains owned through Draining and Closing, so this
				// only stops at the hard no-write boundary: Retired or Closed.
				l.Debug("stopping lease extension because connection phase does not allow lease extension", "phase", preparedConn.phase())
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
				l.Error("error marshaling extend payload", "error", err)
				continue
			}

			if err := wsproto.Write(ctx, preparedConn.ws, &connectproto.ConnectMessage{
				Kind:    connectproto.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE,
				Payload: extendPayload,
			}); err != nil {
				l.Error("error sending extend request", "error", err)
			}
		}
	}()

	resp, err := h.handler.Serve(ctx, body)
	if err != nil {
		return nil, err
	}

	return newSDKResponse(body, resp), nil
}
