package debugapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/queue"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *debugAPI) GetQueueItem(ctx context.Context, req *pb.QueueItemRequest) (*pb.QueueItemResponse, error) {
	if itemID := req.GetItemId(); itemID != "" {
		shardName := consts.DefaultQueueShardName
		if req.QueueShard != "" {
			shardName = req.QueueShard
		}

		shard, err := d.shards.ByName(shardName)
		if err != nil {
			return nil, fmt.Errorf("could not find queue shard %q", shardName)
		}

		queueItem, err := d.queue.LoadQueueItem(ctx, shard.Name(), itemID)
		if err != nil {
			if errors.Is(err, queue.ErrQueueItemNotFound) {
				return nil, status.Error(codes.NotFound, "no item found with id")
			}
			return nil, status.Error(codes.Unknown, fmt.Errorf("error retrieving queue item: %w", err).Error())
		}

		byt, err := json.Marshal(queueItem)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Errorf("error marshalling queue item: %w", err).Error())
		}

		return &pb.QueueItemResponse{
			Data:       byt,
			QueueShard: shard.Name(),
		}, nil
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

	scope, err := debugScope(req.GetFunctionId(), req.GetAccountId(), req.GetEnvId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var shard queue.QueueShard
	if req.QueueShard != "" {
		shard, err = d.shards.ByName(req.QueueShard)
		if err != nil {
			return nil, fmt.Errorf("could not find queue shard %q", req.QueueShard)
		}
	} else {
		shard, err = d.shards.Resolve(ctx, scope, nil)
		if err != nil {
			return nil, fmt.Errorf("could not resolve queue shard for account %q: %w", scope.AccountID, err)
		}
	}

	items, err := shard.ItemsByRunID(ctx, scope, runID)
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
		Data:       byt,
		QueueShard: shard.Name(),
	}, nil
}
