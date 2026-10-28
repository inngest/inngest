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
	infos, err := d.batchManager.GetBatchInfo(ctx, fnID, req.GetBatchKey())
	if err != nil {
		return nil, fmt.Errorf("failed to get batch info: %w", err)
	}

	return BatchInfoResponse(infos), nil
}

// BatchInfoResponse converts per-store batch info into a response. The
// top-level fields describe the first store that holds items, or else the
// first store with a batch, for clients that read a single batch.
func BatchInfoResponse(infos []batch.BatchInfo) *pb.BatchInfoResponse {
	resp := &pb.BatchInfoResponse{}
	var summary *pb.BatchStoreInfo
	for _, info := range infos {
		store := &pb.BatchStoreInfo{
			Store:     info.Store,
			BatchId:   info.BatchID,
			ItemCount: int32(len(info.Items)),
			Items:     batchEventItems(info.Items),
			Status:    info.Status,
		}
		resp.Stores = append(resp.Stores, store)
		if store.BatchId == "" {
			continue
		}
		if summary == nil || (summary.ItemCount == 0 && store.ItemCount > 0) {
			summary = store
		}
	}
	if summary != nil {
		resp.BatchId = summary.BatchId
		resp.ItemCount = summary.ItemCount
		resp.Items = summary.Items
		resp.Status = summary.Status
	}
	return resp
}

func batchEventItems(batchItems []batch.BatchItem) []*pb.BatchEventItem {
	items := make([]*pb.BatchEventItem, 0, len(batchItems))
	for _, item := range batchItems {
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
	return items
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

	results, err := d.batchManager.DeleteBatch(ctx, fnID, req.GetBatchKey())
	if err != nil {
		return nil, fmt.Errorf("failed to delete batch: %w", err)
	}

	return DeleteBatchResponse(results), nil
}

// DeleteBatchResponse converts per-store delete results into a response. The
// top-level fields summarize every store: deleted is true if any store deleted
// a batch, batch_id is the first deleted batch, and item_count is the total.
func DeleteBatchResponse(results []batch.DeleteBatchResult) *pb.DeleteBatchResponse {
	resp := &pb.DeleteBatchResponse{}
	for _, result := range results {
		resp.Stores = append(resp.Stores, &pb.DeleteBatchStoreResult{
			Store:     result.Store,
			Deleted:   result.Deleted,
			BatchId:   result.BatchID,
			ItemCount: int32(result.ItemCount),
		})
		if !result.Deleted {
			continue
		}
		if !resp.Deleted {
			resp.BatchId = result.BatchID
		}
		resp.Deleted = true
		resp.ItemCount += int32(result.ItemCount)
	}
	return resp
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

	results, err := d.batchManager.RunBatch(ctx, batch.RunBatchOpts{
		FunctionID:  fnID,
		BatchKey:    req.GetBatchKey(),
		AccountID:   consts.DevServerAccountID,
		WorkspaceID: fn.EnvID,
		AppID:       fn.AppID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to run batch: %w", err)
	}

	return RunBatchResponse(results), nil
}

// RunBatchResponse converts per-store run results into a response. The
// top-level fields summarize every store: scheduled is true if any store
// scheduled a batch, batch_id is the first scheduled batch, and item_count is
// the total.
func RunBatchResponse(results []batch.RunBatchResult) *pb.RunBatchResponse {
	resp := &pb.RunBatchResponse{}
	for _, result := range results {
		resp.Stores = append(resp.Stores, &pb.RunBatchStoreResult{
			Store:     result.Store,
			Scheduled: result.Scheduled,
			BatchId:   result.BatchID,
			ItemCount: int32(result.ItemCount),
		})
		if !result.Scheduled {
			continue
		}
		if !resp.Scheduled {
			resp.BatchId = result.BatchID
		}
		resp.Scheduled = true
		resp.ItemCount += int32(result.ItemCount)
	}
	return resp
}
