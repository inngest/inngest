package debugapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

// GetBatchInfo retrieves information about the current batch for a function and batch key.
func (d *debugAPI) GetBatchInfo(ctx context.Context, req *BatchInfoRequest) (*BatchInfoResponse, error) {
	if d.batchClient == nil {
		return nil, fmt.Errorf("batch client not configured")
	}

	fnID, err := uuid.Parse(req.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Get the batch pointer key
	batchKey := req.BatchKey
	if batchKey == "" {
		batchKey = "default"
	}

	// Hash the batch key the same way the batch manager does
	hashedBatchKey := sha256.Sum256([]byte(batchKey))
	encodedBatchKey := base64.StdEncoding.EncodeToString(hashedBatchKey[:])

	// Get the batch pointer (which contains the batch ID)
	batchPointerKey := d.batchClient.KeyGenerator().BatchPointerWithKey(ctx, fnID, encodedBatchKey)

	// Read the batch ID from the pointer using RetriableClient interface
	batchIDStr, err := d.batchClient.Client().Do(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Get().Key(batchPointerKey).Build()
	}).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// No active batch
			return &BatchInfoResponse{
				BatchID:   "",
				ItemCount: 0,
				Items:     []*BatchEventItem{},
				Status:    "none",
			}, nil
		}
		return nil, fmt.Errorf("failed to get batch pointer: %w", err)
	}

	batchID, err := ulid.Parse(batchIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid batch ID in pointer: %w", err)
	}

	// Get the batch items
	batchListKey := d.batchClient.KeyGenerator().Batch(ctx, fnID, batchID)
	itemStrList, err := d.batchClient.Client().Do(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Lrange().Key(batchListKey).Start(0).Stop(-1).Build()
	}).AsStrSlice()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return &BatchInfoResponse{
				BatchID:   batchIDStr,
				ItemCount: 0,
				Items:     []*BatchEventItem{},
				Status:    "empty",
			}, nil
		}
		return nil, fmt.Errorf("failed to retrieve batch items: %w", err)
	}

	// Parse the items
	items := make([]*BatchEventItem, 0, len(itemStrList))
	for _, str := range itemStrList {
		item := &batch.BatchItem{}
		if err := json.Unmarshal([]byte(str), item); err != nil {
			return nil, fmt.Errorf("failed to decode batch item: %w", err)
		}

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

	// Get the batch metadata (status)
	metadataKey := d.batchClient.KeyGenerator().BatchMetadata(ctx, fnID, batchID)
	status, err := d.batchClient.Client().Do(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Hget().Key(metadataKey).Field("status").Build()
	}).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			status = "pending"
		} else {
			status = "unknown"
		}
	}

	return &BatchInfoResponse{
		BatchID:   batchIDStr,
		ItemCount: int32(len(items)),
		Items:     items,
		Status:    status,
	}, nil
}
