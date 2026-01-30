package debugapi

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/singleton"
)

// GetSingletonInfo retrieves the current singleton lock status for a given key.
func (d *debugAPI) GetSingletonInfo(ctx context.Context, req *SingletonInfoRequest) (*SingletonInfoResponse, error) {
	if d.singletonStore == nil {
		return nil, fmt.Errorf("singleton store not configured")
	}

	// Type assert to SingletonStore interface which has GetCurrentRunID
	store, ok := d.singletonStore.(singleton.SingletonStore)
	if !ok {
		return nil, fmt.Errorf("singleton store does not implement SingletonStore interface")
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	// Get the current run ID holding the singleton lock
	runID, err := store.GetCurrentRunID(ctx, req.SingletonKey, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get singleton info: %w", err)
	}

	if runID == nil {
		return &SingletonInfoResponse{
			HasLock:      false,
			CurrentRunID: "",
		}, nil
	}

	return &SingletonInfoResponse{
		HasLock:      true,
		CurrentRunID: runID.String(),
	}, nil
}
