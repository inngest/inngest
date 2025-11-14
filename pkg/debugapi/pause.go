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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *debugAPI) GetPause(ctx context.Context, req *pb.PauseRequest) (*pb.PauseResponse, error) {
	if itemID := req.GetItemId(); itemID != "" {

		wId, err := uuid.Parse(req.WorkspaceId)
		if err != nil {
			return nil, status.Error(codes.Unknown, fmt.Errorf("error parsing workspace-id, needs UUID", err).Error())
		}

		pID, err := uuid.Parse(req.ItemId)
		if err != nil {
			return nil, status.Error(codes.Unknown, fmt.Errorf("error parsing pause id, needs UUID", err).Error())
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
