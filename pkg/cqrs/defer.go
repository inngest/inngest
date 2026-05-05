package cqrs

import (
	"encoding/json"

	"github.com/oklog/ulid/v2"
)

// RunDeferStatus represents the status of a defer attached to a function run.
//
// Values mirror the GraphQL enum constants in
// pkg/coreapi/graph/models/models_gen.go (RunDeferStatus*).
type RunDeferStatus string

const (
	RunDeferStatusScheduled RunDeferStatus = "SCHEDULED"
	RunDeferStatusAborted   RunDeferStatus = "ABORTED"
)

// RunDefer is the structured representation of a single defer associated with
// a parent function run. It is parsed from history opcodes (DeferAdd/
// DeferCancel) and joined to the resulting child run when one exists.
type RunDefer struct {
	// ID is the user-provided defer ID (Userland.ID on the opcode).
	ID string
	// FnSlug is the deferred function slug (DeferAddOpts.FnSlug).
	FnSlug string
	// Status is the resolved status after folding any DeferCancel into the
	// matching DeferAdd.
	Status RunDeferStatus
	// Input is the raw JSON input payload from the DeferAdd opcode. nil if
	// absent.
	Input json.RawMessage
	// Run is the child function run triggered by this defer's deterministic
	// schedule event. nil if the defer was aborted, or the event has not
	// yet been processed.
	Run *FunctionRun
}

// RunDeferredFrom is the inverse of RunDefer: it points a deferred (child) run
// back to the run that scheduled it. Populated by parsing the child run's
// triggering inngest/deferred.schedule event metadata.
type RunDeferredFrom struct {
	ParentRunID  ulid.ULID
	ParentFnSlug string
	// ParentRun is the parent function run; nil if it could not be loaded
	// (e.g., orphaned because the parent row was pruned).
	ParentRun *FunctionRun
}
