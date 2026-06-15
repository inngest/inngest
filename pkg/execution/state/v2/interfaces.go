package state

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
)

type CreateState struct {
	Metadata Metadata
	// Events contains a slice of JSON-encoded events.
	Events []json.RawMessage
	// Steps allows users to specify pre-defined steps to run workflows from
	// arbitrary points.
	Steps []state.MemoizedStep
	// StepInputs allows users to specify pre-defined step inputs to run
	// workflows from arbitrary points.
	StepInputs []state.MemoizedStep
}

type RunService interface {
	StateLoader

	// Create creates new state in the store for the given run ID.
	Create(ctx context.Context, s CreateState) (State, error)
	// Delete deletes state, metadata, and - when pauses are included - associated pauses
	// for the run from the store.  Nothing referencing the run should exist in the state
	// store after.
	Delete(ctx context.Context, id ID) error
	// Exists checks whether a run exists given an ID
	Exists(ctx context.Context, id ID) (bool, error)
	// Update updates configuration on the state, eg. setting the execution
	// version after communicating with the SDK.
	UpdateMetadata(ctx context.Context, id ID, config MutableConfig) error
	// SaveStep saves step output for the given run ID and step ID.
	SaveStep(ctx context.Context, id ID, stepID string, data []byte) (hasPending bool, err error)
	// SavePending saves pending step IDs for the given run ID.
	SavePending(ctx context.Context, id ID, pending []string) error

	// ConsumePause consumes a pause by its ID. It does not care about the pause's origin;
	// it only uses the pause data to populate the state of a run.
	//
	// XXX: This function does not interact with any pause backend. A pause manager is
	// expected to wrap this call and handle any required pause cleanup. As a result,
	// this is usually not the function you want to call directly.
	ConsumePause(ctx context.Context, p state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, error)

	SaveDefer(ctx context.Context, id ID, d Defer) error
	// SetDeferStatus atomically flips a Defer's ScheduleStatus. Errors when
	// no defer exists for hashedID. The Aborted transition also releases
	// the Input from the aggregate budget; the meta entry stays.
	SetDeferStatus(ctx context.Context, id ID, hashedID string, status enums.DeferStatus) error
	// SaveRejectedDefer idempotently writes a Rejected meta sentinel.
	// No-op if any defer already exists for hashedID. Returns
	// ErrDeferLimitExceeded if no room.
	SaveRejectedDefer(ctx context.Context, id ID, fnSlug string, hashedID string) error
}

// FinalizationClaim is a storage-neutral handle for a run's finish-effect
// emission claim.  State backends own the underlying storage details: Redis can
// use SET NX, Cassandra can use a conditional insert/update, and callers do not
// depend on either implementation.
type FinalizationClaim struct {
	claimed bool
	release func(context.Context) error
}

// NewFinalizationClaim constructs a finalization claim handle for a state
// backend implementation.
func NewFinalizationClaim(claimed bool, release func(context.Context) error) FinalizationClaim {
	return FinalizationClaim{claimed: claimed, release: release}
}

// Claimed returns true only for the caller allowed to emit finish effects for
// this run.  Duplicate finalizers should still clean up state, but must skip
// externally-visible finish effects.
func (c FinalizationClaim) Claimed() bool {
	return c.claimed
}

// Release clears a previously-acquired finalization claim after a publish
// failure so a later retry can emit finish effects.  Release is best effort and
// is intentionally scoped to the backend-created handle instead of requiring
// callers to know how the claim is addressed in storage.
func (c FinalizationClaim) Release(ctx context.Context) error {
	if c.release == nil {
		return nil
	}
	return c.release(ctx)
}

// FinalizationClaimAdapter is an optional state-store adapter for claiming a
// run's finish effects.  Implementations must provide first-writer-wins
// semantics without assuming multi-key transactions; the claim should be safe
// for non-transactional backends such as Cassandra.
type FinalizationClaimAdapter interface {
	// ClaimFinalization returns a backend-owned claim handle.  The handle's
	// Claimed value decides whether finalize-time effects may be emitted.
	ClaimFinalization(ctx context.Context, md Metadata) (FinalizationClaim, error)
}

// TryClaimFinalization asks the state backend to claim finish-effect emission.
// Backends without a claim adapter preserve previous behavior and allow emit.
func TryClaimFinalization(ctx context.Context, svc RunService, md Metadata) (FinalizationClaim, bool, error) {
	claimant, ok := svc.(FinalizationClaimAdapter)
	if !ok {
		return NewFinalizationClaim(true, nil), false, nil
	}

	claim, err := claimant.ClaimFinalization(ctx, md)
	return claim, true, err
}

// MetadataSizeIncrementer is an optional extension to RunService for
// implementations that support atomic metadata size tracking. Callers should
// use TryIncrementMetadataSize to safely attempt the operation.
type MetadataSizeIncrementer interface {
	// IncrementMetadataSize atomically increments the cumulative metadata size
	// counter for a run. Used by the checkpoint handler to persist metadata
	// size deltas that were tracked in-memory during span creation.
	IncrementMetadataSize(ctx context.Context, id ID, delta int) error
}

// TryIncrementMetadataSize attempts to increment the metadata size counter
// if the given RunService supports it. Returns nil if unsupported.
func TryIncrementMetadataSize(ctx context.Context, svc RunService, id ID, delta int) error {
	if inc, ok := svc.(MetadataSizeIncrementer); ok {
		return inc.IncrementMetadataSize(ctx, id, delta)
	}
	return nil
}

type LoadMetadataOpts struct {
	OmitStackAndStepMetrics bool
}

type LoadMetadataOption func(*LoadMetadataOpts)

// OmitStackAndStepMetrics skips loading Stack and step-derived metrics
// (StateSize, StepCount), avoiding an extra round trip. Those fields are
// zero-valued in the returned Metadata.
func OmitStackAndStepMetrics() LoadMetadataOption {
	return func(o *LoadMetadataOpts) { o.OmitStackAndStepMetrics = true }
}

func ApplyLoadMetadataOpts(opts []LoadMetadataOption) LoadMetadataOpts {
	var o LoadMetadataOpts
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// Staeloader defines an interface for loading the entire run state from the state store.
type StateLoader interface {
	// Metadata returns metadata for a given run
	LoadMetadata(ctx context.Context, id ID, opts ...LoadMetadataOption) (Metadata, error)
	// LoadEvents loads the triggering events for the given run.
	LoadEvents(ctx context.Context, id ID) ([]json.RawMessage, error)
	// LoadState returns all steps for a run.
	LoadSteps(ctx context.Context, id ID) (map[string]json.RawMessage, error)
	// LoadStepInputs returns only the step inputs for a run.
	LoadStepInputs(ctx context.Context, id ID) (map[string]json.RawMessage, error)
	// LoadStepsWithIDs returns a list of steps with the given IDs for a run.
	LoadStepsWithIDs(ctx context.Context, id ID, stepIDs []string) (map[string]json.RawMessage, error)
	// LoadStack returns the stack for a given run
	LoadStack(ctx context.Context, id ID) ([]string, error)

	// LoadState returns all state for a run, including steps, events, and metadata.
	LoadState(ctx context.Context, id ID) (State, error)

	LoadDefers(ctx context.Context, id ID) (map[string]Defer, error)

	// LoadDefersMeta returns each defer's metadata without loading its Input.
	// Prefer this when only FnSlug/HashedID/ScheduleStatus are needed.
	LoadDefersMeta(ctx context.Context, id ID) (map[string]DeferMeta, error)
}

//
// Re-exports for compat.
//

type (
	GeneratorOpcode = state.GeneratorOpcode
	UserError       = state.UserError
	DriverResponse  = state.DriverResponse
)

var (
	ErrRunNotFound        = state.ErrRunNotFound
	ErrIdempotentResponse = state.ErrIdempotentResponse
	ErrDuplicateResponse  = state.ErrDuplicateResponse

	// ErrDeferInputTooLarge re-exports the v1 error so v2 callers can match
	// without importing v1.
	ErrDeferInputTooLarge = state.ErrDeferInputTooLarge
)
