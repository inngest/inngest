package debugapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/queue"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *debugAPI) GetShadowPartition(ctx context.Context, req *pb.ShadowPartitionRequest) (*pb.ShadowPartitionResponse, error) {
	shard, err := d.findShard(ctx, consts.DevServerAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("error finding shard: %w", err)
	}

	pt, err := d.queue.PartitionByID(ctx, shard, req.GetPartitionId())
	if err != nil {
		if errors.Is(err, queue.ErrPartitionNotFound) {
			return nil, status.Error(codes.NotFound, queue.ErrPartitionNotFound.Error())
		}
		return nil, fmt.Errorf("error retrieving partition: %w", err)
	}

	sp := pt.QueueShadowPartition
	if sp == nil {
		return nil, status.Error(codes.NotFound, "shadow partition not found")
	}

	resp := &pb.ShadowPartitionResponse{
		PartitionId:     sp.PartitionID,
		FunctionVersion: int32(sp.FunctionVersion),
		BacklogCount:    int64(pt.Backlogs),
	}
	if sp.LeaseID != nil {
		resp.LeaseId = sp.LeaseID.String()
	}
	if sp.FunctionID != nil {
		resp.FunctionId = sp.FunctionID.String()
	}
	if sp.EnvID != nil {
		resp.EnvId = sp.EnvID.String()
	}
	if sp.AccountID != nil {
		resp.AccountId = sp.AccountID.String()
	}
	if sp.SystemQueueName != nil {
		resp.SystemQueueName = *sp.SystemQueueName
	}

	return resp, nil
}

func (d *debugAPI) GetBacklogs(ctx context.Context, req *pb.BacklogsRequest) (*pb.BacklogsResponse, error) {
	shard, err := d.findShard(ctx, consts.DevServerAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("error finding shard: %w", err)
	}

	until := time.Now().Add(365 * 24 * time.Hour)
	iter, err := d.queue.BacklogsByPartition(ctx, shard, req.GetPartitionId(), time.Time{}, until)
	if err != nil {
		return nil, fmt.Errorf("error listing backlogs: %w", err)
	}

	limit := req.GetLimit()
	if limit <= 0 {
		limit = 1000 // default limit
	}

	var backlogs []*pb.BacklogInfo
	var count int64
	for bl := range iter {
		count++
		if int64(len(backlogs)) >= limit {
			continue // keep counting but stop collecting
		}

		size, _ := d.queue.BacklogSize(ctx, shard, bl.BacklogID)
		backlogs = append(backlogs, mapBacklogToProto(bl, size))
	}

	return &pb.BacklogsResponse{
		Backlogs:   backlogs,
		TotalCount: count,
	}, nil
}

func (d *debugAPI) GetBacklogSize(ctx context.Context, req *pb.BacklogSizeRequest) (*pb.BacklogSizeResponse, error) {
	shard, err := d.findShard(ctx, consts.DevServerAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("error finding shard: %w", err)
	}

	size, err := d.queue.BacklogSize(ctx, shard, req.GetBacklogId())
	if err != nil {
		return nil, fmt.Errorf("error getting backlog size: %w", err)
	}

	resp := &pb.BacklogSizeResponse{
		BacklogId: req.GetBacklogId(),
		ItemCount: size,
	}

	bl, err := d.queue.BacklogByID(ctx, shard, req.GetBacklogId())
	if err == nil {
		resp.Backlog = mapBacklogToProto(bl, size)
	}

	return resp, nil
}

func mapBacklogToProto(bl *queue.QueueBacklog, itemCount int64) *pb.BacklogInfo {
	info := &pb.BacklogInfo{
		BacklogId:               bl.BacklogID,
		ShadowPartitionId:       bl.ShadowPartitionID,
		EarliestFunctionVersion: int32(bl.EarliestFunctionVersion),
		Start:                   bl.Start,
		ItemCount:               itemCount,
	}
	for _, ck := range bl.ConcurrencyKeys {
		info.ConcurrencyKeys = append(info.ConcurrencyKeys, &pb.BacklogConcurrencyKeyInfo{
			CanonicalKeyId:      ck.CanonicalKeyID,
			Scope:               ck.Scope.String(),
			EntityId:            ck.EntityID.String(),
			HashedKeyExpression: ck.HashedKeyExpression,
			HashedValue:         ck.HashedValue,
			UnhashedValue:       ck.UnhashedValue,
			ConcurrencyMode:     ck.ConcurrencyMode.String(),
		})
	}
	if bl.Throttle != nil {
		info.Throttle = &pb.BacklogThrottleInfo{
			ThrottleKey:               bl.Throttle.ThrottleKey,
			ThrottleKeyRawValue:       bl.Throttle.ThrottleKeyRawValue,
			ThrottleKeyExpressionHash: bl.Throttle.ThrottleKeyExpressionHash,
		}
	}
	return info
}
