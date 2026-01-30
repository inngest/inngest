package debugapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/redis/rueidis"
)

// GetDebounceInfo retrieves the currently debounced event for a function and debounce key.
func (d *debugAPI) GetDebounceInfo(ctx context.Context, req *DebounceInfoRequest) (*DebounceInfoResponse, error) {
	if d.debounceClient == nil {
		return nil, fmt.Errorf("debounce client not configured")
	}

	fnID, err := uuid.Parse(req.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Get the debounce key - if not provided, use the function ID
	debounceKey := req.DebounceKey
	if debounceKey == "" {
		debounceKey = fnID.String()
	}

	// Get the debounce pointer (which contains the debounce ID)
	debouncePointerKey := d.debounceClient.KeyGenerator().DebouncePointer(ctx, fnID, debounceKey)

	// Read the debounce ID from the pointer
	debounceIDStr, err := d.debounceClient.Client().Do(ctx, d.debounceClient.Client().B().Get().Key(debouncePointerKey).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// No active debounce
			return &DebounceInfoResponse{
				HasDebounce: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get debounce pointer: %w", err)
	}

	// Get the debounce item from the hash
	debounceHashKey := d.debounceClient.KeyGenerator().Debounce(ctx)
	itemBytes, err := d.debounceClient.Client().Do(ctx, d.debounceClient.Client().B().Hget().Key(debounceHashKey).Field(debounceIDStr).Build()).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// Debounce ID exists in pointer but item not found in hash
			return &DebounceInfoResponse{
				HasDebounce: true,
				DebounceID:  debounceIDStr,
			}, nil
		}
		return nil, fmt.Errorf("failed to get debounce item: %w", err)
	}

	// Parse the debounce item
	var di debounce.DebounceItem
	if err := json.Unmarshal(itemBytes, &di); err != nil {
		return nil, fmt.Errorf("failed to decode debounce item: %w", err)
	}

	eventData, err := json.Marshal(di.Event)
	if err != nil {
		eventData = []byte("{}")
	}

	return &DebounceInfoResponse{
		HasDebounce: true,
		DebounceID:  debounceIDStr,
		EventID:     di.EventID.String(),
		EventData:   eventData,
		Timeout:     di.Timeout,
		AccountID:   di.AccountID.String(),
		WorkspaceID: di.WorkspaceID.String(),
		FunctionID:  di.FunctionID.String(),
	}, nil
}
