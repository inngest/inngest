package debugapi

import (
	"context"
	"fmt"

	"cuelang.org/go/pkg/uuid"
	"github.com/inngest/inngest/pkg/consts"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func (d *debugAPI) GetPartition(ctx context.Context, req *pb.PartitionRequest) (*pb.PartitionResponse, error) {
	return nil, errNotImplemented
}

func (d *debugAPI) GetPartitionStatus(ctx context.Context, req *pb.PartitionRequest) (*pb.PartitionStatusResponse, error) {
	var queueName *string
	if _, err := uuid.Parse(req.GetId()); err != nil {
		queueName = &req.Id
	}

	shard, err := d.findShard(ctx, consts.DevServerAccountID, queueName)
	if err != nil {
		return nil, fmt.Errorf("error finding shard for GetPartition: %w", err)
	}

	pt, err := d.queue.PartitionByID(ctx, shard, req.GetId())
	if err != nil {
		return nil, fmt.Errorf("error retrieving partition: %w", err)
	}

	return &pb.PartitionStatusResponse{
		Id:      req.GetId(),
		Paused:  pt.Paused,
		Migrate: pt.Migrate,

		AccountActive:     int64(pt.AccountActive),
		AccountInProgress: int64(pt.AccountInProgress),
		Ready:             int64(pt.Ready),
		InProgress:        int64(pt.InProgress),
		Active:            int64(pt.Active),
		Future:            int64(pt.Future),
		Backlogs:          int64(pt.Backlogs),
	}, nil
}
