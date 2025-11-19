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

func MustRunServiceV2(m statev1.Manager, opts ...MgrV2Opt) state.RunService {
	o := &mgrV2Opts{}
	for _, apply := range opts {
		apply(o)
	}

	v2, err := runServiceV2(m, *o)
	if err != nil {
		panic(err)
	}

	return v2
}

func runServiceV2(m statev1.Manager, opts mgrV2Opts) (state.RunService, error) {
	mgr, ok := m.(*mgr)
	if !ok {
		return nil, fmt.Errorf("cannot convert %T into type redis_state.*mgr", m)
	}

	v2 := v2{mgr: mgr, disabledRetries: opts.disabledRetries}
	return v2, nil
}

type (
	MgrV2Opt  func(o *mgrV2Opts)
	mgrV2Opts struct {
		disabledRetries bool
	}
)

func WithDisabledRetries() MgrV2Opt {
	return func(o *mgrV2Opts) {
		o.disabledRetries = true
	}
}

type v2 struct {
	mgr             *mgr
	disabledRetries bool
}

// Create creates new state in the store for the given run ID.
func (v v2) Create(ctx context.Context, s state.CreateState) (state.State, error) {
	batchData := make([]map[string]any, len(s.Events))
	for n, evt := range s.Events {
		data := map[string]any{}
		if err := json.Unmarshal(evt, &data); err != nil {
			return state.State{}, err
		}
		batchData[n] = data

	}
	st, err := v.mgr.New(ctx, statev1.Input{
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
	switch err {
	case nil:
		// no-op continue
	case statev1.ErrIdentifierExists:
		s.Metadata.ID.RunID = st.RunID()
		// NOTE:  Idempotency keys are non-transactional, so we want to retry this LoadState
		// call up to 3 times, to ensure that the original thread between saving idempotency
		// keys and saving state is set.
		st, err := util.WithRetry(
			ctx,
			"load-state",
			func(ctx context.Context) (state.State, error) {
				return v.LoadState(ctx, s.Metadata.ID)
			},
			util.RetryConf{
				MaxAttempts:    3,
				InitialBackoff: 25 * time.Millisecond,
				MaxBackoff:     150 * time.Millisecond,
			},
		)
		if err != nil {
			return state.State{}, err
		}
		return st, statev1.ErrIdentifierExists
	default:
		return state.State{}, err
	}

	// XXX: We do the exact same size calculations done in `mgr.New` to return a v2 state without changing the v1 interface.
	var stepsByt []byte
	if len(s.Steps) > 0 {
		stepsByt, err = json.Marshal(s.Steps)
		if err != nil {
			return state.State{}, fmt.Errorf("error storing run state in redis when marshalling steps: %w", err)
		}
	}

	var stepInputsByt []byte
	if len(s.StepInputs) > 0 {
		stepInputsByt, err = json.Marshal(s.StepInputs)
		if err != nil {
			return state.State{}, fmt.Errorf("error storing run state in redis when marshalling step inputs: %w", err)
		}
	}

	events, err := json.Marshal(batchData)
	if err != nil {
		return state.State{}, fmt.Errorf("error storing run state in redis when marshalling batchData: %w", err)
	}

	metadata := s.Metadata
	metadata.ID = state.ID{
		// Set the returned run ID from the state manager
		RunID:      st.RunID(),
		FunctionID: s.Metadata.ID.FunctionID,
		Tenant: state.Tenant{
			AppID:     s.Metadata.ID.Tenant.AppID,
			EnvID:     s.Metadata.ID.Tenant.EnvID,
			AccountID: s.Metadata.ID.Tenant.AccountID,
		},
	}
	stateSize := len(events) + len(stepsByt) + len(stepInputsByt)
	metadata.Metrics = state.RunMetrics{
		EventSize: len(events),
		StateSize: stateSize,
		StepCount: len(s.Steps),
	}

	metrics.IncrStateWrittenCounter(ctx, stateSize, metrics.CounterOpt{
		PkgName: "redis_state",
		Tags: map[string]any{
			"account_id": s.Metadata.ID.Tenant.AccountID,
		},
	})

	steps := make(map[string]json.RawMessage)
	for _, step := range s.Steps {
		if data, err := json.Marshal(step.Data); err == nil {
			steps[step.ID] = json.RawMessage(data)
		}
	}

	return state.State{Metadata: metadata, Events: s.Events, Steps: steps}, nil
}

// Delete deletes state, metadata, and - when pauses are included - associated pauses
// for the run from the store.  Nothing referencing the run should exist in the state
// store after.
func (v v2) Delete(ctx context.Context, id state.ID) error {
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

// LoadSteps returns all steps for a run.
func (v v2) LoadSteps(ctx context.Context, id state.ID) (map[string]json.RawMessage, error) {
	return v.mgr.LoadSteps(ctx, id.Tenant.AccountID, id.FunctionID, id.RunID)
}

// LoadStepInputs returns only the step inputs for a run.
func (v v2) LoadStepInputs(ctx context.Context, id state.ID) (map[string]json.RawMessage, error) {
	return v.mgr.LoadStepInputs(ctx, id.Tenant.AccountID, id.FunctionID, id.RunID)
}

// LoadStepsWithIDs returns a list of steps with the given IDs for a run.
func (v v2) LoadStepsWithIDs(ctx context.Context, id state.ID, stepIDs []string) (map[string]json.RawMessage, error) {
	return v.mgr.LoadStepsWithIDs(ctx, id.Tenant.AccountID, id.FunctionID, id.RunID, stepIDs)
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

// LoadStack returns the current stack for a run.
func (v v2) LoadStack(ctx context.Context, id state.ID) ([]string, error) {
	return v.mgr.stack(ctx, id.Tenant.AccountID, id.RunID)
}

// Update updates configuration on the state, eg. setting the execution
// version after communicating with the SDK.
func (v v2) UpdateMetadata(ctx context.Context, id state.ID, mutation state.MutableConfig) error {
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
		v.retryPolicy(),
	)

	return err
}

// SaveStep saves step output for the given run ID and step ID.
func (v v2) SaveStep(ctx context.Context, id state.ID, stepID string, data []byte) (bool, error) {
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
		v.retryPolicy(
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
		v.retryPolicy(),
	)

	return err
}

func (v v2) retryPolicy(opts ...util.RetryConfSetting) util.RetryConf {
	if v.disabledRetries {
		opts = append(opts, util.WithRetryConfMaxAttempts(1))
	}
	return util.NewRetryConf(opts...)
}

// determine what errors are retriable
func (v v2) retryableError(err error) bool {
	switch {
	case errors.Is(err, statev1.ErrIdempotentResponse):
		return false
	case errors.Is(err, statev1.ErrDuplicateResponse):
		return false
	}

	return true
}
