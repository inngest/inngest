package debugapi

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/singleton"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

// GetSingletonInfo retrieves the current singleton lock status for a given key.
func (d *debugAPI) GetSingletonInfo(ctx context.Context, req *pb.SingletonInfoRequest) (*pb.SingletonInfoResponse, error) {
	if d.singletonStore == nil {
		return nil, fmt.Errorf("singleton store not configured")
	}

	// Type assert to SingletonStore interface which has GetCurrentRunID
	store, ok := d.singletonStore.(singleton.SingletonStore)
	if !ok {
		return nil, fmt.Errorf("singleton store does not implement SingletonStore interface")
	}

	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Build singleton key: function_id or function_id-suffix
	singletonKey := fnID.String()
	if req.GetSingletonKey() != "" {
		singletonKey = fnID.String() + "-" + req.GetSingletonKey()
	}

	// Get the current run ID holding the singleton lock
	runID, err := store.GetCurrentRunID(ctx, singletonKey, consts.DevServerAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get singleton info: %w", err)
	}

	if runID == nil {
		return &pb.SingletonInfoResponse{
			HasLock:      false,
			CurrentRunId: "",
		}, nil
	}

	return &pb.SingletonInfoResponse{
		HasLock:      true,
		CurrentRunId: runID.String(),
	}, nil
}

// DeleteSingletonLock removes an existing singleton lock.
func (d *debugAPI) DeleteSingletonLock(ctx context.Context, req *pb.DeleteSingletonLockRequest) (*pb.DeleteSingletonLockResponse, error) {
	if d.singletonStore == nil {
		return nil, fmt.Errorf("singleton store not configured")
	}

	// Type assert to SingletonStore interface which has ReleaseSingleton
	store, ok := d.singletonStore.(singleton.SingletonStore)
	if !ok {
		return nil, fmt.Errorf("singleton store does not implement SingletonStore interface")
	}

	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Build singleton key: function_id or function_id-suffix
	singletonKey := fnID.String()
	if req.GetSingletonKey() != "" {
		singletonKey = fnID.String() + "-" + req.GetSingletonKey()
	}

	// Release the singleton lock (which deletes it)
	runID, err := store.ReleaseSingleton(ctx, singletonKey, consts.DevServerAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete singleton lock: %w", err)
	}

	if runID == nil {
		return &pb.DeleteSingletonLockResponse{
			Deleted: false,
			RunId:   "",
		}, nil
	}

	return &pb.DeleteSingletonLockResponse{
		Deleted: true,
		RunId:   runID.String(),
	}, nil
}
