package connect

import (
	"context"
	"errors"
	"time"

	"github.com/coder/websocket"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const workerRequestAckNotifyTimeout = 2 * time.Second

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

	notifyCtx, notifyCancel := context.WithTimeout(context.Background(), workerRequestAckNotifyTimeout)
	defer notifyCancel()

	l := c.log.With(
		"req_id", data.RequestId,
		"run_id", data.RunId,
		"transport", "grpc",
	)

	grpcClient, err := c.svc.getOrCreateGRPCClient(notifyCtx, c.conn.EnvID, data.RequestId)
	switch {
	case err == nil:
	case errors.Is(err, state.ErrExecutorNotFound):
		l.Debug("executor not found in lease, worker ack was likely picked up by polling")
		return nil
	default:
		l.Warn("could not create grpc client to ack, executor will rely on timeout or polling", "err", err)
		return nil
	}

	reply, err := grpcClient.Ack(notifyCtx, &connectpb.AckMessage{
		RequestId: data.RequestId,
		Ts:        timestamppb.Now(),
	})
	if err != nil {
		l.Warn("could not ack message through gRPC, executor will rely on timeout or polling", "err", err)
		return nil
	}
	if reply == nil || !reply.Success {
		l.Warn("failed to ack, executor was likely done with the request")
	}
	l.Trace("worker acked message")

	// TODO Should we send a reverse ack to the worker to start processing the request?
	return nil
}
