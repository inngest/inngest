package connect

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/publicerr"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
)

func (h *connectHandler) writeRequestAck(ctx context.Context, preparedConn *connection, ackPayload []byte, l *slog.Logger) error {
	// ACK is the request ownership boundary. If this generation cannot ACK,
	// do not invoke; the gateway/executor can retry or reroute the request.
	if !preparedConn.canWriteRequestAck() {
		return errConnectionRetired
	}

	if err := ctx.Err(); err != nil {
		writeErr := fmt.Errorf("could not write message to websocket: %w", err)
		preparedConn.retire("request ack write failed", "err", writeErr)
		l.Error("error sending request ack", "error", writeErr, "phase", preparedConn.phase())
		return publicerr.Wrap(writeErr, 400, "failed to ack worker request")
	}

	if err := wsproto.Write(ctx, preparedConn.ws, &connectproto.ConnectMessage{
		Kind:    connectproto.GatewayMessageType_WORKER_REQUEST_ACK,
		Payload: ackPayload,
	}); err != nil {
		preparedConn.retire("request ack write failed", "err", err)
		l.Error("error sending request ack", "error", err, "phase", preparedConn.phase())
		return publicerr.Wrap(err, 400, "failed to ack worker request")
	}

	return nil
}

func (h *connectHandler) writeReply(ctx context.Context, preparedConn *connection, resp *connectproto.SDKResponse, responseMessage *connectproto.ConnectMessage) error {
	if !preparedConn.canWriteReply() {
		h.bufferReply(resp)
		h.logger.Debug(
			"buffering sdk response because connection phase does not allow reply write",
			"request_id", resp.RequestId,
			"phase", preparedConn.phase(),
			"connection_id", preparedConn.connectionId,
		)
		return nil
	}

	if err := wsproto.Write(ctx, preparedConn.ws, responseMessage); err != nil {
		// ACKed work has already completed. Preserve the existing policy:
		// buffer the reply for API flush and do not retire on reply write
		// failure.
		h.bufferReply(resp)
		h.logger.Debug(
			"buffering sdk response after websocket write failure",
			"err", err,
			"request_id", resp.RequestId,
			"phase", preparedConn.phase(),
			"connection_id", preparedConn.connectionId,
		)
		return nil
	}

	return nil
}

func (h *connectHandler) bufferReply(resp *connectproto.SDKResponse) {
	h.messageBuffer.append(resp)

	// A reply can be buffered after the connection lifecycle notification was
	// already consumed, so appending flushable work must also wake the manager.
	if h.notifyFlushChan == nil {
		return
	}
	select {
	case h.notifyFlushChan <- struct{}{}:
	default:
	}
}

func canExtendRequestLease(preparedConn *connection) bool {
	return preparedConn.canWriteExtendLease()
}
