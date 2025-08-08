package debugapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrPartitionNotAvailable = fmt.Errorf("partition not available")
)

func (d *debugAPI) GetPartition(ctx context.Context, req *pb.PartitionRequest) (*pb.PartitionResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		// not a user based function, could be system queues

		return &pb.PartitionResponse{
			Id: req.GetId(),
			Tenant: &pb.PartitionTenant{
				AccountId: consts.DevServerAccountID.String(),
				EnvId:     consts.DevServerEnvID.String(),
			},
		}, nil
	}

	wf, err := d.db.GetFunctionByInternalUUID(ctx, consts.DevServerEnvID, id)
	if err != nil {
		return nil, fmt.Errorf("error retrieving function: %w", err)
	}

	shard, err := d.findShard(ctx, consts.DevServerAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("error finding shard: %w", err)
	}

	return &pb.PartitionResponse{
		Id:   req.GetId(),
		Slug: wf.Slug,
		Tenant: &pb.PartitionTenant{
			AccountId: consts.DevServerAccountID.String(),
			EnvId:     consts.DevServerEnvID.String(),
			AppId:     wf.AppID.String(),
		},
		Config: wf.Config,
		QueueShard: &pb.QueueShard{
			Name: shard.Name,
			Kind: shard.Kind,
		},
	}, nil
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
		if errors.Is(err, redis_state.ErrPartitionNotFound) {
			return nil, status.Error(codes.NotFound, redis_state.ErrPartitionNotFound.Error())
		}

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
