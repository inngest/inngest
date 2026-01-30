package debugapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

// GetDebounceInfo retrieves the currently debounced event for a function and debounce key.
func (d *debugAPI) GetDebounceInfo(ctx context.Context, req *pb.DebounceInfoRequest) (*pb.DebounceInfoResponse, error) {
	if d.debouncer == nil {
		return nil, fmt.Errorf("debouncer not configured")
	}

	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Use the debouncer to get debounce info
	info, err := d.debouncer.GetDebounceInfo(ctx, fnID, req.GetDebounceKey())
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce info: %w", err)
	}

	// No active debounce
	if info.DebounceID == "" || info.Item == nil {
		return &pb.DebounceInfoResponse{
			HasDebounce: info.DebounceID != "",
			DebounceId:  info.DebounceID,
		}, nil
	}

	// Convert to response format
	eventData, err := json.Marshal(info.Item.Event)
	if err != nil {
		eventData = []byte("{}")
	}

	return &pb.DebounceInfoResponse{
		HasDebounce: true,
		DebounceId:  info.DebounceID,
		EventId:     info.Item.EventID.String(),
		EventData:   eventData,
		Timeout:     info.Item.Timeout,
		AccountId:   info.Item.AccountID.String(),
		WorkspaceId: info.Item.WorkspaceID.String(),
		FunctionId:  info.Item.FunctionID.String(),
	}, nil
}
