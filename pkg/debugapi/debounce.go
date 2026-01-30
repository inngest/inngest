package debugapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// GetDebounceInfo retrieves the currently debounced event for a function and debounce key.
func (d *debugAPI) GetDebounceInfo(ctx context.Context, req *DebounceInfoRequest) (*DebounceInfoResponse, error) {
	if d.debouncer == nil {
		return nil, fmt.Errorf("debouncer not configured")
	}

	fnID, err := uuid.Parse(req.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Use the debouncer to get debounce info
	info, err := d.debouncer.GetDebounceInfo(ctx, fnID, req.DebounceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce info: %w", err)
	}

	// No active debounce
	if info.DebounceID == "" || info.Item == nil {
		return &DebounceInfoResponse{
			HasDebounce: info.DebounceID != "",
			DebounceID:  info.DebounceID,
		}, nil
	}

	// Convert to response format
	eventData, err := json.Marshal(info.Item.Event)
	if err != nil {
		eventData = []byte("{}")
	}

	return &DebounceInfoResponse{
		HasDebounce: true,
		DebounceID:  info.DebounceID,
		EventID:     info.Item.EventID.String(),
		EventData:   eventData,
		Timeout:     info.Item.Timeout,
		AccountID:   info.Item.AccountID.String(),
		WorkspaceID: info.Item.WorkspaceID.String(),
		FunctionID:  info.Item.FunctionID.String(),
	}, nil
}
