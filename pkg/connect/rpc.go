package connect

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	pb "github.com/inngest/inngest/proto/gen/connect/v1"
)

func (c *connectGatewaySvc) Forward(ctx context.Context, req *pb.ForwardRequest) (*pb.ForwardResponse, error) {
	l := logger.StdlibLogger(ctx)
	l.Trace("received grpc message from executor")

	if ch, ok := c.wsConnections.Load(req.ConnectionID); ok {
		l.Trace("found ws connection by connectionID")
		msgChan := ch.(chan *pb.GatewayExecutorRequestData)

		select {
		case msgChan <- req.Data:
			// XXX: Should we ack after the ws write or it's fine to ack just
			// after the message is consumed.

			return &pb.ForwardResponse{Success: true}, nil
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
