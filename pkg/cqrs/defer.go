package cqrs

import (
	"encoding/json"

	"github.com/oklog/ulid/v2"
)

type RunDeferStatus string

const (
	RunDeferStatusScheduled RunDeferStatus = "SCHEDULED"
	RunDeferStatusAborted   RunDeferStatus = "ABORTED"
)

// RunDefer is a single defer attached to a parent function run. Parsed from
// the parent's history opcodes (DeferAdd/DeferAbort) and joined to the
// resulting child run when one exists. Run is nil if the defer was aborted
// or the schedule event has not yet been processed.
type RunDefer struct {
	ID     string
	FnSlug string
	Status RunDeferStatus
	Input  json.RawMessage
	Run    *TraceRun
}

// RunDeferredFrom is the parent-run linkage of a deferred (child) run,
// parsed from the child's triggering inngest/deferred.schedule event metadata.
// ParentRun is nil if the parent row could not be loaded (e.g. pruned).
type RunDeferredFrom struct {
	ParentRunID  ulid.ULID
	ParentFnSlug string
	ParentRun    *TraceRun
}
