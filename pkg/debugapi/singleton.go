package debugapi

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/queue"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

// GetSingletonInfo retrieves the current singleton lock status for a given key.
func (d *debugAPI) GetSingletonInfo(ctx context.Context, req *pb.SingletonInfoRequest) (*pb.SingletonInfoResponse, error) {
	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Build singleton key: function_id or function_id-suffix
	singletonKey := fnID.String()
	if req.GetSingletonKey() != "" {
		singletonKey = fnID.String() + "-" + req.GetSingletonKey()
	}

	shard, err := d.shards.Resolve(ctx, consts.DevServerAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve shard: %w", err)
	}

	runID, err := shard.SingletonGetRunID(ctx, queue.Scope{
		AccountID:  consts.DevServerAccountID,
		EnvID:      consts.DevServerEnvID,
		FunctionID: fnID,
	}, singletonKey)
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
	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Build singleton key: function_id or function_id-suffix
	singletonKey := fnID.String()
	if req.GetSingletonKey() != "" {
		singletonKey = fnID.String() + "-" + req.GetSingletonKey()
	}

	shard, err := d.shards.Resolve(ctx, consts.DevServerAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve shard: %w", err)
	}

	runID, err := shard.SingletonReleaseRunID(ctx, queue.Scope{
		AccountID:  consts.DevServerAccountID,
		EnvID:      consts.DevServerEnvID,
		FunctionID: fnID,
	}, singletonKey)
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
