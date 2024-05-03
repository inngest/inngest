package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	statev1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/v2"
)

type v2 struct {
	mgr mgr
}

// Create creates new state in the store for the given run ID.
func (v v2) Create(ctx context.Context, s state.CreateState) error {
	batchData := make([]map[string]any, len(s.Events))
	for n, evt := range s.Events {
		data := map[string]any{}
		if err := json.Unmarshal(evt, &data); err != nil {
			return err
		}
		batchData[n] = data

	}
	_, err := v.mgr.New(ctx, statev1.Input{
		Identifier: statev1.Identifier{
			RunID:                 s.Metadata.ID.RunID,
			WorkflowID:            s.Metadata.ID.FunctionID,
			WorkflowVersion:       0, // TODO
			EventID:               s.Metadata.Config.EventIDs[0],
			EventIDs:              s.Metadata.Config.EventIDs,
			Key:                   s.Metadata.Config.Idempotency,
			AccountID:             s.Metadata.Tenant.AccountID,
			WorkspaceID:           s.Metadata.Tenant.EnvID,
			AppID:                 s.Metadata.Tenant.AppID,
			OriginalRunID:         s.Metadata.Config.OriginalRunID,
			ReplayID:              s.Metadata.Config.ReplayID,
			PriorityFactor:        s.Metadata.Config.PriorityFactor,
			CustomConcurrencyKeys: s.Metadata.Config.CustomConcurrencyKeys,
		},
		EventBatchData: batchData,
		Context:        s.Metadata.Config.Context,
		SpanID:         s.Metadata.Config.SpanID,
	})
	return err
}

// Delete deletes state, metadata, and - when pauses are included - associated pauses
// for the run from the store.  Nothing referencing the run should exist in the state
// store after.
func (v v2) Delete(ctx context.Context, id state.ID) error {
	return v.mgr.Delete(ctx, statev1.Identifier{
		RunID:      id.RunID,
		WorkflowID: id.FunctionID,
	})
}

// LoadState returns all state for a run.
func (v v2) LoadState(ctx context.Context, id state.ID) (state.State, error) {
	// TODO: Define state.
	return nil, nil
}

// StreamState returns all state without loading in-memory
func (v v2) StreamState(ctx context.Context, id state.ID) (io.Reader, error) {
	return nil, fmt.Errorf("not implemented")
}

// Metadata returns metadata for a given run
func (v v2) LoadMetadata(ctx context.Context, id state.ID) (state.Metadata, error) {
	md, err := v.mgr.Metadata(ctx, id.RunID)
	if err != nil {
		return state.Metadata{}, err
	}

	// TODO: Run metrics.
	return state.Metadata{
		ID: state.ID{
			RunID:      md.Identifier.RunID,
			FunctionID: md.Identifier.WorkflowID,
		},
		Tenant: state.Tenant{
			AppID:     md.Identifier.AppID,
			EnvID:     md.Identifier.WorkspaceID,
			AccountID: md.Identifier.AccountID,
		},
		Config: state.Config{
			SpanID:                md.SpanID,
			StartedAt:             md.StartedAt,
			EventIDs:              md.Identifier.EventIDs,
			RequestVersion:        md.RequestVersion,
			Idempotency:           md.Identifier.Key,
			ReplayID:              md.Identifier.ReplayID,
			OriginalRunID:         md.Identifier.OriginalRunID,
			PriorityFactor:        md.Identifier.PriorityFactor,
			CustomConcurrencyKeys: md.Identifier.CustomConcurrencyKeys,
			Context:               md.Context,
			ForceStepPlan:         md.DisableImmediateExecution,
		},
	}, nil
}

// Update updates configuration on the state, eg. setting the execution
// version after communicating with the SDK.
func (v v2) UpdateMetadata(ctx context.Context, id state.ID, mutation state.MutableConfig) error {
	return v.mgr.UpdateMetadata(ctx, id.RunID, statev1.MetadataUpdate{
		DisableImmediateExecution: mutation.ForceStepPlan,
		RequestVersion:            mutation.RequestVersion,
		StartedAt:                 mutation.StartedAt,
	})
}

// SaveStep saves step output for the given run ID and step ID.
func (v v2) SaveStep(ctx context.Context, id state.ID, stepID state.StepID, data []byte) error {
	v1id := statev1.Identifier{
		RunID:      id.RunID,
		WorkflowID: id.FunctionID,
	}
	return v.mgr.SaveResponse(ctx, v1id, stepID.String(), string(data))
}
