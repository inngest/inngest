package connect

import (
	"context"

	"github.com/coder/websocket"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	// This will be sent exactly once, as the router selected this gateway to
	// handle the request. Even if the gateway is draining, we should ack the
	// message; the SDK will buffer messages and use a new connection for
	// results.
	grpcClient, err := c.svc.getOrCreateGRPCClient(context.Background(), c.conn.EnvID, data.RequestId)
	if err != nil {
		return &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "could not create grpc client to ack",
		}
	}

	reply, err := grpcClient.Ack(context.Background(), &connectpb.AckMessage{
		RequestId: data.RequestId,
		Ts:        timestamppb.Now(),
	})
	if err != nil {
		// This should never happen: Failing the ack means we will redeliver
		// the same request even though the worker already started processing it.
		c.log.ReportError(err, "failed to ack message through gRPC")
		return &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "could not ack message through gRPC",
		}
	}
	if !reply.Success {
		c.log.Warn("failed to ack, executor was likely done with the request")
	}
	c.log.Trace("worker acked message",
		"req_id", data.RequestId,
		"run_id", data.RunId,
		"transport", "grpc",
	)

	// TODO Should we send a reverse ack to the worker to start processing the request?
	return nil
}
