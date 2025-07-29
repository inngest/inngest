package debugapi

import (
	"context"

	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func (d *debugAPI) GetPartition(ctx context.Context, req *pb.PartitionRequest) (*pb.PartitionResponse, error) {
	return nil, errNotImplemented
}

func (d *debugAPI) GetPartitionStatus(ctx context.Context, req *pb.PartitionRequest) (*pb.PartitionStatusResponse, error) {
	return nil, errNotImplemented
}
