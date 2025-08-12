package redis_state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	statev1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
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
func (v v2) Create(ctx context.Context, s state.CreateState) (statev1.State, error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"op": "create"}})
	}()

	batchData := make([]map[string]any, len(s.Events))
	for n, evt := range s.Events {
		data := map[string]any{}
		if err := json.Unmarshal(evt, &data); err != nil {
			return nil, err
		}
		batchData[n] = data

	}
	state, err := v.mgr.New(ctx, statev1.Input{
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
	if err == nil {
		metrics.IncrStateWrittenCounter(ctx, len(batchData), metrics.CounterOpt{
			PkgName: "redis_state",
			Tags: map[string]any{
				"account_id": s.Metadata.ID.Tenant.AccountID,
			},
		})
	}
	return state, err
}

// Delete deletes state, metadata, and - when pauses are included - associated pauses
// for the run from the store.  Nothing referencing the run should exist in the state
// store after.
func (v v2) Delete(ctx context.Context, id state.ID) (bool, error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"op": "delete"}})
	}()

	return v.mgr.Delete(ctx, statev1.Identifier{
		RunID:      id.RunID,
		WorkflowID: id.FunctionID,
		AccountID:  id.Tenant.AccountID,
	})
}

func (v v2) Exists(ctx context.Context, id state.ID) (bool, error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"op": "exists"}})
	}()

	return v.mgr.Exists(ctx, id.Tenant.AccountID, id.RunID)
}

// LoadEvents returns all events for a run.
func (v v2) LoadEvents(ctx context.Context, id state.ID) ([]json.RawMessage, error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"op": "loadevents"}})
	}()

	return v.mgr.LoadEvents(ctx, id.Tenant.AccountID, id.FunctionID, id.RunID)
}

// LoadEvents returns all events for a run.
func (v v2) LoadSteps(ctx context.Context, id state.ID) (map[string]json.RawMessage, error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"op": "loadsteps"}})
	}()

	return v.mgr.LoadSteps(ctx, id.Tenant.AccountID, id.FunctionID, id.RunID)
}

// LoadState returns all state for a run.
func (v v2) LoadState(ctx context.Context, id state.ID) (state.State, error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"op": "loadstate"}})
	}()

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
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"op": "loadmetadata"}})
	}()

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
			HasAI:                 md.HasAI,
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
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"op": "updatemetadata"}})
	}()

	_, err := util.WithRetry(
		ctx,
		"state.UpdateMetadata",
		func(ctx context.Context) (bool, error) {
			err := v.mgr.UpdateMetadata(ctx, id.Tenant.AccountID, id.RunID, statev1.MetadataUpdate{
				DisableImmediateExecution: mutation.ForceStepPlan,
				RequestVersion:            mutation.RequestVersion,
				StartedAt:                 mutation.StartedAt,
				HasAI:                     mutation.HasAI,
			})

			return false, err
		},
		util.NewRetryConf(),
	)

	return err
}

// SaveStep saves step output for the given run ID and step ID.
func (v v2) SaveStep(ctx context.Context, id state.ID, stepID string, data []byte) (bool, error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"op": "savestep"}})
	}()

	v1id := statev1.Identifier{
		RunID:      id.RunID,
		WorkflowID: id.FunctionID,
		AccountID:  id.Tenant.AccountID,
	}

	attempt := 0
	hasPending, err := util.WithRetry(
		ctx,
		"state.SaveStep",
		func(ctx context.Context) (bool, error) {
			attempt++
			return v.mgr.SaveResponse(ctx, v1id, stepID, string(data))
		},
		util.NewRetryConf(
			util.WithRetryConfRetryableErrors(v.retryableError),
			util.WithRetryConfMaxBackoff(10*time.Second),
			util.WithRetryConfMaxAttempts(10),
		),
	)

	if errors.Is(err, statev1.ErrIdempotentResponse) {
		// This step data for this step ID has already been saved exactly as before.
		logger.StdlibLogger(ctx).Warn(
			"swallowing idempotent step response",
			"attempt", attempt,
			"run_id", id.RunID,
			"step_id", stepID,
		)
		// NOTE: hasPending should be accurate in this case.
		return hasPending, nil
	}

	if errors.Is(err, statev1.ErrDuplicateResponse) && attempt > 1 {
		// Swallow the error. Since the 2nd attempt has a "duplicate response"
		// (i.e. already exists in Redis), we can assume that the first attempt
		// successfully updated Redis despite the retry. This can happen if we
		// get a context timeout in Go code but Redis actually completed the
		// operation.
		logger.StdlibLogger(ctx).Warn(
			"swallowing duplicate response",
			"attempt", attempt,
			"run_id", id.RunID,
			"step_id", stepID,
		)
		return false, nil
	}

	// We only record the number of bytes written after handling idempotent and
	// duplicate errors;  those don't count towards backing state store growth.
	metrics.IncrStateWrittenCounter(ctx, len(data), metrics.CounterOpt{
		PkgName: "redis_state",
		Tags: map[string]any{
			"account_id": id.Tenant.AccountID,
		},
	})

	return hasPending, err
}

// SavePending saves pending step IDs for the given run ID.
func (v v2) SavePending(ctx context.Context, id state.ID, pending []string) error {
	start := time.Now()
	defer func() {
		dur := time.Since(start).Milliseconds()
		metrics.HistogramStateStoreOperationDuration(ctx, dur, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"op": "savepending"}})
	}()

	v1id := statev1.Identifier{
		RunID:      id.RunID,
		WorkflowID: id.FunctionID,
		AccountID:  id.Tenant.AccountID,
	}

	_, err := util.WithRetry(
		ctx,
		"state.SavePending",
		func(ctx context.Context) (bool, error) {
			err := v.mgr.SavePending(ctx, v1id, pending)
			return false, err
		},
		util.NewRetryConf(),
	)

	return err
}

// determine what errors are retriable
func (v v2) retryableError(err error) bool {
	switch {
	case errors.Is(err, statev1.ErrDuplicateResponse):
		return false
	}

	return true
}
