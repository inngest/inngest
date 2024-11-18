package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	statev1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/v2"
)

func MustRunServiceV2(m statev1.Manager) state.RunService {
	v2, err := RunServiceV2(m)
	if err != nil {
		panic(err)
	}
	return v2
}

func RunServiceV2(m statev1.Manager) (state.RunService, error) {
	mgr, ok := m.(*mgr)
	if !ok {
		return nil, fmt.Errorf("cannot convert %T into type redis_state.*mgr", m)
	}
	return v2{mgr}, nil
}

type v2 struct {
	mgr *mgr
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
			WorkflowVersion:       s.Metadata.Config.FunctionVersion,
			EventID:               s.Metadata.Config.EventID(),
			EventIDs:              s.Metadata.Config.EventIDs,
			Key:                   s.Metadata.Config.Idempotency,
			AccountID:             s.Metadata.ID.Tenant.AccountID,
			WorkspaceID:           s.Metadata.ID.Tenant.EnvID,
			AppID:                 s.Metadata.ID.Tenant.AppID,
			OriginalRunID:         s.Metadata.Config.OriginalRunID,
			ReplayID:              s.Metadata.Config.ReplayID,
			PriorityFactor:        s.Metadata.Config.PriorityFactor,
			CustomConcurrencyKeys: s.Metadata.Config.CustomConcurrencyKeys,
			BatchID:               s.Metadata.Config.BatchID,
		},
		EventBatchData: batchData,
		Context:        s.Metadata.Config.Context,
		SpanID:         s.Metadata.Config.SpanID,
		Steps:          s.Steps,
		StepInputs:     s.StepInputs,
	})
	return err
}

// Delete deletes state, metadata, and - when pauses are included - associated pauses
// for the run from the store.  Nothing referencing the run should exist in the state
// store after.
func (v v2) Delete(ctx context.Context, id state.ID) (bool, error) {
	return v.mgr.Delete(ctx, statev1.Identifier{
		RunID:      id.RunID,
		WorkflowID: id.FunctionID,
		AccountID:  id.Tenant.AccountID,
	})
}

func (v v2) Exists(ctx context.Context, id state.ID) (bool, error) {
	return v.mgr.Exists(ctx, id.Tenant.AccountID, id.RunID)
}

// LoadEvents returns all events for a run.
func (v v2) LoadEvents(ctx context.Context, id state.ID) ([]json.RawMessage, error) {
	return v.mgr.LoadEvents(ctx, id.Tenant.AccountID, id.FunctionID, id.RunID)
}

// LoadEvents returns all events for a run.
func (v v2) LoadSteps(ctx context.Context, id state.ID) (map[string]json.RawMessage, error) {
	return v.mgr.LoadSteps(ctx, id.Tenant.AccountID, id.FunctionID, id.RunID)
}

// LoadState returns all state for a run.
func (v v2) LoadState(ctx context.Context, id state.ID) (state.State, error) {
	var (
		err   error
		state = state.State{}
	)

	if state.Metadata, err = v.LoadMetadata(ctx, id); err != nil {
		return state, err
	}

	// Reassign id since state.Metadata.ID has more complete info. Specifically,
	// it has the function ID
	id = state.Metadata.ID

	if state.Events, err = v.LoadEvents(ctx, id); err != nil {
		return state, err
	}
	if state.Steps, err = v.LoadSteps(ctx, id); err != nil {
		return state, err
	}

	return state, nil
}

// StreamState returns all state without loading in-memory
func (v v2) StreamState(ctx context.Context, id state.ID) (io.Reader, error) {
	return nil, fmt.Errorf("not implemented")
}

// Metadata returns metadata for a given run
func (v v2) LoadMetadata(ctx context.Context, id state.ID) (state.Metadata, error) {
	md, err := v.mgr.metadata(ctx, id.Tenant.AccountID, id.RunID)
	if err != nil {
		return state.Metadata{}, err
	}

	stack, err := v.mgr.stack(ctx, id.Tenant.AccountID, id.RunID)
	if err != nil {
		return state.Metadata{}, err
	}

	var startedAt time.Time
	if md.StartedAt > 0 {
		startedAt = time.UnixMilli(md.StartedAt)
	}

	result := state.Metadata{
		ID: state.ID{
			RunID:      md.Identifier.RunID,
			FunctionID: md.Identifier.WorkflowID,
			Tenant: state.Tenant{
				AppID:     md.Identifier.AppID,
				EnvID:     md.Identifier.WorkspaceID,
				AccountID: md.Identifier.AccountID,
			},
		},
		Config: *state.InitConfig(&state.Config{
			FunctionVersion:       md.Identifier.WorkflowVersion,
			SpanID:                md.SpanID,
			StartedAt:             startedAt,
			EventIDs:              md.Identifier.EventIDs,
			BatchID:               md.Identifier.BatchID,
			RequestVersion:        md.RequestVersion,
			Idempotency:           md.Identifier.Key,
			ReplayID:              md.Identifier.ReplayID,
			OriginalRunID:         md.Identifier.OriginalRunID,
			PriorityFactor:        md.Identifier.PriorityFactor,
			CustomConcurrencyKeys: md.Identifier.CustomConcurrencyKeys,
			Context:               md.Context,
			ForceStepPlan:         md.DisableImmediateExecution,
		}),
		Stack: stack,
		Metrics: state.RunMetrics{
			EventSize: md.EventSize,
			StateSize: md.StateSize,
			StepCount: md.StepCount,
		},
	}

	// initialize function trace eagerly; this needs to unmarshal the trace carrier
	_ = result.Config.FunctionTrace()

	return result, nil
}

// Update updates configuration on the state, eg. setting the execution
// version after communicating with the SDK.
func (v v2) UpdateMetadata(ctx context.Context, id state.ID, mutation state.MutableConfig) error {
	return v.mgr.UpdateMetadata(ctx, id.Tenant.AccountID, id.RunID, statev1.MetadataUpdate{
		DisableImmediateExecution: mutation.ForceStepPlan,
		RequestVersion:            mutation.RequestVersion,
		StartedAt:                 mutation.StartedAt,
	})
}

// SaveStep saves step output for the given run ID and step ID.
func (v v2) SaveStep(ctx context.Context, id state.ID, stepID string, data []byte) error {
	v1id := statev1.Identifier{
		RunID:      id.RunID,
		WorkflowID: id.FunctionID,
		AccountID:  id.Tenant.AccountID,
	}
	return v.mgr.SaveResponse(ctx, v1id, stepID, string(data))
}
