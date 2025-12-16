package debugapi

import (
	"context"
	"errors"
	"fmt"

	"encoding/json"

	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *debugAPI) GetQueueItem(ctx context.Context, req *pb.QueueItemRequest) (*pb.QueueItemResponse, error) {
	opts := []redis_state.QueueOpOpt{}

	if shardName := req.GetQueueShard(); shardName != "" {
		shard, ok := d.queue.Shard(ctx, shardName)
		if ok {
			opts = append(opts, redis_state.WithQueueOpShard(shard))
		}
	}

	if itemID := req.GetItemId(); itemID != "" {
		queueItem, err := d.queue.ItemByID(ctx, itemID, opts...)
		if err != nil {
			if errors.Is(err, redis_state.ErrQueueItemNotFound) {
				return nil, status.Error(codes.NotFound, "no item found with id")
			}
			return nil, status.Error(codes.Unknown, fmt.Errorf("error retrieving queue item: %w", err).Error())
		}

		byt, err := json.Marshal(queueItem)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Errorf("error marshalling queue item: %w", err).Error())
		}

		return &pb.QueueItemResponse{Data: byt}, nil
	}

	// use runID
	var runID ulid.ULID
	{
		id, err := ulid.Parse(req.GetRunId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid ULID provided: %w", err).Error())
		}
		runID = id
	}
	items, err := d.queue.ItemsByRunID(ctx, runID, opts...)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("error retrieving queue items by runID: %w", err).Error())
	}
	if len(items) == 0 {
		return nil, status.Error(codes.NotFound, "no items found with runID")
	}

	// TODO eventually return a list
	qi := items[0]
	byt, err := json.Marshal(qi)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("error marshalling queue item: %w", err).Error())
	}

	return &pb.QueueItemResponse{
		Data: byt,
	}, nil
}
