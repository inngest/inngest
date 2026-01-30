package debugapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/batch"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

// GetBatchInfo retrieves information about the current batch for a function and batch key.
func (d *debugAPI) GetBatchInfo(ctx context.Context, req *pb.BatchInfoRequest) (*pb.BatchInfoResponse, error) {
	if d.batchManager == nil {
		return nil, fmt.Errorf("batch manager not configured")
	}

	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Use the batch manager to get batch info
	info, err := d.batchManager.GetBatchInfo(ctx, fnID, req.GetBatchKey())
	if err != nil {
		return nil, fmt.Errorf("failed to get batch info: %w", err)
	}

	// Convert batch items to response format
	items := make([]*pb.BatchEventItem, 0, len(info.Items))
	for _, item := range info.Items {
		eventData, err := json.Marshal(item.Event)
		if err != nil {
			eventData = []byte("{}")
		}

		items = append(items, &pb.BatchEventItem{
			EventId:         item.EventID.String(),
			AccountId:       item.AccountID.String(),
			WorkspaceId:     item.WorkspaceID.String(),
			AppId:           item.AppID.String(),
			FunctionId:      item.FunctionID.String(),
			FunctionVersion: int32(item.FunctionVersion),
			EventData:       eventData,
		})
	}

	return &pb.BatchInfoResponse{
		BatchId:   info.BatchID,
		ItemCount: int32(len(items)),
		Items:     items,
		Status:    info.Status,
	}, nil
}

// DeleteBatch deletes the current batch for a function and batch key.
func (d *debugAPI) DeleteBatch(ctx context.Context, req *pb.DeleteBatchRequest) (*pb.DeleteBatchResponse, error) {
	if d.batchManager == nil {
		return nil, fmt.Errorf("batch manager not configured")
	}

	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	result, err := d.batchManager.DeleteBatch(ctx, fnID, req.GetBatchKey())
	if err != nil {
		return nil, fmt.Errorf("failed to delete batch: %w", err)
	}

	return &pb.DeleteBatchResponse{
		Deleted:   result.Deleted,
		BatchId:   result.BatchID,
		ItemCount: int32(result.ItemCount),
	}, nil
}

// RunBatch schedules immediate execution of a batch.
func (d *debugAPI) RunBatch(ctx context.Context, req *pb.RunBatchRequest) (*pb.RunBatchResponse, error) {
	if d.batchManager == nil {
		return nil, fmt.Errorf("batch manager not configured")
	}

	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Resolve workspace and app IDs from function
	fn, err := d.db.GetFunctionByInternalUUID(ctx, fnID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve function: %w", err)
	}

	result, err := d.batchManager.RunBatch(ctx, batch.RunBatchOpts{
		FunctionID:  fnID,
		BatchKey:    req.GetBatchKey(),
		AccountID:   consts.DevServerAccountID,
		WorkspaceID: fn.EnvID,
		AppID:       fn.AppID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to run batch: %w", err)
	}

	return &pb.RunBatchResponse{
		Scheduled: result.Scheduled,
		BatchId:   result.BatchID,
		ItemCount: int32(result.ItemCount),
	}, nil
}
