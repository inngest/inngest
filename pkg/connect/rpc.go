package connect

import (
	"context"

	pb "github.com/inngest/inngest/proto/gen/connect/v1"
)

func (c *connectGatewaySvc) Forward(ctx context.Context, req *pb.ForwardRequest) (*pb.ForwardResponse, error) {
	return nil, nil
}


func (c *connectGatewaySvc) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		Message: "ok",
	}, nil
}

