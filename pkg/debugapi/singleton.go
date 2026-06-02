package debugapi

import (
	"context"
	"fmt"

	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

// GetSingletonInfo retrieves the current singleton lock status for a given key.
func (d *debugAPI) GetSingletonInfo(ctx context.Context, req *pb.SingletonInfoRequest) (*pb.SingletonInfoResponse, error) {
	scope, err := debugScope(req.GetFunctionId(), req.GetAccountId(), req.GetEnvId())
	if err != nil {
		return nil, err
	}

	// Build singleton key: function_id or function_id-suffix
	singletonKey := scope.FunctionID.String()
	if req.GetSingletonKey() != "" {
		singletonKey = scope.FunctionID.String() + "-" + req.GetSingletonKey()
	}

	shard, err := d.shards.Resolve(ctx, scope.AccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve shard: %w", err)
	}

	runID, err := shard.SingletonGetRunID(ctx, scope, singletonKey)
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
	scope, err := debugScope(req.GetFunctionId(), req.GetAccountId(), req.GetEnvId())
	if err != nil {
		return nil, err
	}

	// Build singleton key: function_id or function_id-suffix
	singletonKey := scope.FunctionID.String()
	if req.GetSingletonKey() != "" {
		singletonKey = scope.FunctionID.String() + "-" + req.GetSingletonKey()
	}

	shard, err := d.shards.Resolve(ctx, scope.AccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve shard: %w", err)
	}

	runID, err := shard.SingletonReleaseRunID(ctx, scope, singletonKey)
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
