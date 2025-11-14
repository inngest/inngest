package debugapi

import (
	"context"
	"errors"
	"fmt"

	"encoding/json"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/state"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *debugAPI) GetPause(ctx context.Context, req *pb.PauseRequest) (*pb.PauseResponse, error) {
	if itemID := req.GetItemId(); itemID != "" {

		wId, err := uuid.Parse(req.WorkspaceId)
		if err != nil {
			return nil, status.Error(codes.Unknown, fmt.Errorf("error parsing workspace-id, needs UUID: %w", err).Error())
		}

		pID, err := uuid.Parse(req.ItemId)
		if err != nil {
			return nil, status.Error(codes.Unknown, fmt.Errorf("error parsing pause id, needs UUID: %w", err).Error())
		}
		index := pauses.Index{EventName: req.EventName, WorkspaceID: wId}
		pause, err := d.pm.PauseByID(ctx, index, pID)

		if err != nil {
			if errors.Is(err, state.ErrPauseNotFound) {
				return nil, status.Error(codes.NotFound, "no pause found with id")
			}
			return nil, status.Error(codes.Unknown, fmt.Errorf("error retrieving pause: %w", err).Error())
		}

		byt, err := json.Marshal(pause)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Errorf("error marshalling pause: %w", err).Error())
		}

		return &pb.PauseResponse{Data: byt}, nil
	}

	return nil, status.Error(codes.Internal, errors.New("missing pause id").Error())
}

func (d *debugAPI) GetIndex(ctx context.Context, req *pb.IndexRequest) (*pb.IndexResponse, error) {
	if req.GetEventName() == "" || req.GetWorkspaceId() == "" {
		return nil, status.Error(codes.InvalidArgument, "event_name and workspace_id are required")
	}

	wId, err := uuid.Parse(req.GetWorkspaceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workspace_id format, must be UUID")
	}

	index := pauses.Index{
		EventName:   req.GetEventName(),
		WorkspaceID: wId,
	}

	stats, err := d.pm.IndexStats(ctx, index)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get index stats: %v", err))
	}

	// Convert to protobuf response
	response := &pb.IndexResponse{
		WorkspaceId:  stats.WorkspaceID.String(),
		EventName:    stats.EventName,
		BufferLength: stats.BufferLength,
	}

	for _, blockInfo := range stats.Blocks {
		response.Blocks = append(response.Blocks, &pb.BlockInfo{
			Id:             blockInfo.ID,
			Length:         int64(blockInfo.Length),
			FirstTimestamp: blockInfo.FirstTimestamp.UnixMilli(),
			LastTimestamp:  blockInfo.LastTimestamp.UnixMilli(),
			DeleteCount:    blockInfo.DeleteCount,
		})
	}

	return response, nil
}

func (d *debugAPI) BlockPeek(ctx context.Context, req *pb.BlockPeekRequest) (*pb.BlockPeekResponse, error) {
	if req.GetEventName() == "" || req.GetWorkspaceId() == "" || req.GetBlockId() == "" {
		return nil, status.Error(codes.InvalidArgument, "event_name, workspace_id, and block_id are required")
	}

	wId, err := uuid.Parse(req.GetWorkspaceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workspace_id format, must be UUID")
	}

	blockID, err := ulid.Parse(req.GetBlockId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid block_id format, must be ULID")
	}

	index := pauses.Index{
		EventName:   req.GetEventName(),
		WorkspaceID: wId,
	}

	pauseIDs, totalCount, err := d.pm.GetBlockPauseIDs(ctx, index, blockID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get block pause IDs: %v", err))
	}

	return &pb.BlockPeekResponse{
		BlockId:    req.GetBlockId(),
		TotalCount: totalCount,
		PauseIds:   pauseIDs,
		Compacted:  int64(len(pauseIDs)) < totalCount,
	}, nil
}

func (d *debugAPI) BlockDeleted(ctx context.Context, req *pb.BlockDeletedRequest) (*pb.BlockDeletedResponse, error) {
	if req.GetEventName() == "" || req.GetWorkspaceId() == "" || req.GetBlockId() == "" {
		return nil, status.Error(codes.InvalidArgument, "event_name, workspace_id, and block_id are required")
	}

	wId, err := uuid.Parse(req.GetWorkspaceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workspace_id format, must be UUID")
	}

	blockID, err := ulid.Parse(req.GetBlockId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid block_id format, must be ULID")
	}

	index := pauses.Index{
		EventName:   req.GetEventName(),
		WorkspaceID: wId,
	}

	deletedIDs, totalCount, err := d.pm.GetBlockDeletedIDs(ctx, index, blockID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get block deleted IDs: %v", err))
	}

	return &pb.BlockDeletedResponse{
		BlockId:    req.GetBlockId(),
		TotalCount: totalCount,
		DeletedIds: deletedIDs,
		Compacted:  int64(len(deletedIDs)) < totalCount,
	}, nil
}
