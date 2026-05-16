package connect

import (
	"github.com/coder/websocket"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/protobuf/proto"
)

func (c *connectionHandler) handleWorkerRequestAck(msg *connectpb.ConnectMessage) *connecterrors.SocketError {
	var data connectpb.WorkerRequestAckData
	if err := proto.Unmarshal(msg.Payload, &data); err != nil {
		// This should never happen: Failing the ack means we will redeliver
		// the same request even though the worker already started processing it.
		return &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectWorkerRequestAckInvalidPayload,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "invalid payload in worker request ack",
		}
	}

	if ackCh, ok := c.pendingAcks.LoadAndDelete(data.RequestId); ok {
		close(ackCh.(chan struct{}))
	} else {
		c.log.Warn(
			"worker request ack received for unknown request ID",
			"conn_id", c.conn.ConnectionId.String(),
			"req_id", data.RequestId,
			"run_id", data.RunId,
		)
	}

	c.log.With("req_id", data.RequestId, "run_id", data.RunId, "transport", "grpc").Trace("worker acked message")

	// TODO Should we send a reverse ack to the worker to start processing the request?
	return nil
}
