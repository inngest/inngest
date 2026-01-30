package debugapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// GetBatchInfo retrieves information about the current batch for a function and batch key.
func (d *debugAPI) GetBatchInfo(ctx context.Context, req *BatchInfoRequest) (*BatchInfoResponse, error) {
	if d.batchManager == nil {
		return nil, fmt.Errorf("batch manager not configured")
	}

	fnID, err := uuid.Parse(req.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Use the batch manager to get batch info
	info, err := d.batchManager.GetBatchInfo(ctx, fnID, req.BatchKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch info: %w", err)
	}

	// Convert batch items to response format
	items := make([]*BatchEventItem, 0, len(info.Items))
	for _, item := range info.Items {
		eventData, err := json.Marshal(item.Event)
		if err != nil {
			eventData = []byte("{}")
		}

		items = append(items, &BatchEventItem{
			EventID:         item.EventID.String(),
			AccountID:       item.AccountID.String(),
			WorkspaceID:     item.WorkspaceID.String(),
			AppID:           item.AppID.String(),
			FunctionID:      item.FunctionID.String(),
			FunctionVersion: item.FunctionVersion,
			EventData:       eventData,
		})
	}

	return &BatchInfoResponse{
		BatchID:   info.BatchID,
		ItemCount: int32(len(items)),
		Items:     items,
		Status:    info.Status,
	}, nil
}
