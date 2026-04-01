package connect

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	pb "github.com/inngest/inngest/proto/gen/connect/v1"
)

// forwardMessage pairs a request with a result channel so `Forward()` can block
// until the WebSocket write completes. Without this, the Gateway believes
// delivery succeeded while the write can still fail (e.g. during drain),
// silently dropping the request.
type forwardMessage struct {
	Data   *pb.GatewayExecutorRequestData
	Result chan error
}

func (c *connectGatewaySvc) Forward(ctx context.Context, req *pb.ForwardRequest) (*pb.ForwardResponse, error) {
	l := logger.StdlibLogger(ctx)
	l.Trace("received grpc message from executor")

	if val, ok := c.wsConnections.Load(req.ConnectionID); ok {
		handler := val.(*connectionHandler)

		if handler.draining.Load() {
			// Block until DRAINING is written to Redis, so the proxy
			// re-route finds the new connection as READY.

			accID, _ := uuid.Parse(req.Data.AccountId)

			l.Optional(accID, "connect").Debug("forward blocked, connection draining",
				"conn_id", req.ConnectionID,
				"req_id", req.Data.RequestId,
				"run_id", req.Data.RunId,
			)
			<-handler.stopForwarding
			l.Optional(accID, "connect").Debug("forward released after drain complete",
				"conn_id", req.ConnectionID,
				"req_id", req.Data.RequestId,
				"run_id", req.Data.RunId,
			)
			return &pb.ForwardResponse{Success: false}, nil
		}

		l.Trace("found ws connection by connectionID")
		msgChan := handler.messageChan

		resultCh := make(chan error, 1)
		msg := forwardMessage{
			Data:   req.Data,
			Result: resultCh,
		}

		select {
		case msgChan <- msg:
			// Block until the WebSocket write completes. Only ack success to
			// the executor after confirmed delivery to the worker.
			select {
			case err := <-resultCh:
				if err != nil {
					l.Error("failed to write message to websocket",
						"err", err,
						"conn_id", req.ConnectionID,
						"req_id", req.Data.RequestId,
						"run_id", req.Data.RunId,
						"fn_slug", req.Data.FunctionSlug,
					)
					return &pb.ForwardResponse{Success: false}, nil
				}
				return &pb.ForwardResponse{Success: true}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(consts.ConnectWorkerRequestLeaseDuration + consts.ConnectWorkerRequestGracePeriod + 5*time.Second):
				l.Error("timeout waiting for worker ACK",
					"conn_id", req.ConnectionID,
					"req_id", req.Data.RequestId,
					"run_id", req.Data.RunId,
					"fn_slug", req.Data.FunctionSlug,
				)
				return &pb.ForwardResponse{Success: false}, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			l.Error("timeout sending message to ws channel after 5 seconds",
				"conn_id", req.ConnectionID,
				"req_id", req.Data.RequestId,
				"run_id", req.Data.RunId,
				"fn_slug", req.Data.FunctionSlug,
			)
			return &pb.ForwardResponse{Success: false}, nil
		}
	}

	// Connection not found
	l.Warn("connection not found in wsConnections",
		"conn_id", req.ConnectionID,
		"req_id", req.Data.RequestId,
		"run_id", req.Data.RunId,
	)
	return &pb.ForwardResponse{Success: false}, nil
}

func (c *connectGatewaySvc) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		Message: "ok",
	}, nil
}
