package connect

import (
	"context"
	"time"

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

	if ch, ok := c.wsConnections.Load(req.ConnectionID); ok {
		l.Trace("found ws connection by connectionID")
		msgChan := ch.(chan forwardMessage)

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
						"account_id", req.Data.AccountId,
						"app_id", req.Data.RunId,
						"conn_id", req.ConnectionID,
						"env_id", req.Data.EnvId,
						"fn_id", req.Data.FunctionId,
						"req_id", req.Data.RequestId,
					)
					return &pb.ForwardResponse{Success: false}, nil
				}
				return &pb.ForwardResponse{Success: true}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				l.Error("timeout waiting for websocket write confirmation",
					"account_id", req.Data.AccountId,
					"app_id", req.Data.RunId,
					"conn_id", req.ConnectionID,
					"env_id", req.Data.EnvId,
					"fn_id", req.Data.FunctionId,
					"req_id", req.Data.RequestId,
				)
				return &pb.ForwardResponse{Success: false}, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			l.Error("timeout sending message to ws channel after 5 seconds")
			return &pb.ForwardResponse{Success: false}, nil
		}
	}

	// Connection not found
	return &pb.ForwardResponse{Success: false}, nil
}

func (c *connectGatewaySvc) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		Message: "ok",
	}, nil
}
