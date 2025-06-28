package connect

import (
	"context"

	pb "github.com/inngest/inngest/proto/gen/connect/v1"
)

func (c *connectGatewaySvc) Forward(ctx context.Context, req *pb.ForwardRequest) (*pb.ForwardResponse, error) {
	// your logic here
	return nil, nil
}
