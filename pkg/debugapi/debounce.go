package debugapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/oklog/ulid/v2"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

// GetDebounceInfo retrieves the currently debounced event for a function and debounce key.
func (d *debugAPI) GetDebounceInfo(ctx context.Context, req *pb.DebounceInfoRequest) (*pb.DebounceInfoResponse, error) {
	if d.debouncer == nil {
		return nil, fmt.Errorf("debouncer not configured")
	}

	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	// Use the debouncer to get debounce info
	info, err := d.debouncer.GetDebounceInfo(ctx, fnID, req.GetDebounceKey())
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce info: %w", err)
	}

	// No active debounce
	if info.DebounceID == "" || info.Item == nil {
		return &pb.DebounceInfoResponse{
			HasDebounce: info.DebounceID != "",
			DebounceId:  info.DebounceID,
		}, nil
	}

	// Convert to response format
	eventData, err := json.Marshal(info.Item.Event)
	if err != nil {
		eventData = []byte("{}")
	}

	return &pb.DebounceInfoResponse{
		HasDebounce: true,
		DebounceId:  info.DebounceID,
		EventId:     info.Item.EventID.String(),
		EventData:   eventData,
		Timeout:     info.Item.Timeout,
		AccountId:   info.Item.AccountID.String(),
		WorkspaceId: info.Item.WorkspaceID.String(),
		FunctionId:  info.Item.FunctionID.String(),
	}, nil
}

// DeleteDebounce deletes the current debounce for a function and debounce key.
func (d *debugAPI) DeleteDebounce(ctx context.Context, req *pb.DeleteDebounceRequest) (*pb.DeleteDebounceResponse, error) {
	if d.debouncer == nil {
		return nil, fmt.Errorf("debouncer not configured")
	}

	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	result, err := d.debouncer.DeleteDebounce(ctx, fnID, req.GetDebounceKey())
	if err != nil {
		return nil, fmt.Errorf("failed to delete debounce: %w", err)
	}

	return &pb.DeleteDebounceResponse{
		Deleted:    result.Deleted,
		DebounceId: result.DebounceID,
		EventId:    result.EventID,
	}, nil
}

// DeleteDebounceByID deletes debounces directly by their IDs.
func (d *debugAPI) DeleteDebounceByID(ctx context.Context, req *pb.DeleteDebounceByIDRequest) (*pb.DeleteDebounceByIDResponse, error) {
	if d.debouncer == nil {
		return nil, fmt.Errorf("debouncer not configured")
	}

	ids := req.GetDebounceIds()
	if len(ids) == 0 {
		return &pb.DeleteDebounceByIDResponse{}, nil
	}
	if len(ids) > 20 {
		return nil, fmt.Errorf("too many debounce IDs: max 20, got %d", len(ids))
	}

	parsed := make([]ulid.ULID, len(ids))
	for i, id := range ids {
		u, err := ulid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("invalid debounce ID %q: %w", id, err)
		}
		parsed[i] = u
	}

	err := d.debouncer.DeleteDebounceByID(ctx, parsed...)
	if err != nil {
		return nil, fmt.Errorf("failed to delete debounce by ID: %w", err)
	}

	return &pb.DeleteDebounceByIDResponse{
		DeletedIds: ids,
	}, nil
}

// RunDebounce schedules immediate execution of a debounce.
func (d *debugAPI) RunDebounce(ctx context.Context, req *pb.RunDebounceRequest) (*pb.RunDebounceResponse, error) {
	if d.debouncer == nil {
		return nil, fmt.Errorf("debouncer not configured")
	}

	fnID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	result, err := d.debouncer.RunDebounce(ctx, debounce.RunDebounceOpts{
		FunctionID:  fnID,
		DebounceKey: req.GetDebounceKey(),
		AccountID:   consts.DevServerAccountID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to run debounce: %w", err)
	}

	return &pb.RunDebounceResponse{
		Scheduled:  result.Scheduled,
		DebounceId: result.DebounceID,
		EventId:    result.EventID,
	}, nil
}
