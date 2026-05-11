package cqrs

import (
	"context"

	"github.com/oklog/ulid/v2"
)

// DeferStore writes parent-run → child-run defer linkages so the dev-server
// GraphQL can expose deferred runs after the parent's state has been deleted.
//
// deferID is the SHA1 hash of the user-supplied defer id (the join key with
// UpdateRunDeferChildRunID). userDeferID preserves the original string so the
// UI can show the id the SDK caller typed.
type DeferStore interface {
	InsertRunDefer(ctx context.Context, parentRunID ulid.ULID, deferID, userDeferID, fnSlug string, status RunDeferStatus) error
	UpdateRunDeferChildRunID(ctx context.Context, parentRunID ulid.ULID, deferID string, childRunID ulid.ULID) error
}

type RunDeferStatus string

const (
	RunDeferStatusScheduled RunDeferStatus = "SCHEDULED"
	RunDeferStatusAborted   RunDeferStatus = "ABORTED"
)

// RunDefer is a single defer attached to a parent function run. Parsed from
// the parent's history opcodes (DeferAdd/DeferAbort) and joined to the
// resulting child run when one exists. Run is nil if the defer was aborted,
// the schedule event has not yet been processed, or the child trace was
// pruned.
//
// ID is the SHA1-hashed defer id used as the join key. UserDeferID is the
// original SDK-caller-supplied id (the first arg to `defer("foo", ...)`) and
// is what the dev-server UI shows. The SDK always emits userland.id, so
// UserDeferID is expected to be non-empty for any defer this server records.
type RunDefer struct {
	ID          string
	UserDeferID string
	FnSlug      string
	Status      RunDeferStatus
	Run         *TraceRun
}

// RunDeferredFrom is the parent-run linkage of a deferred (child) run,
// parsed from the child's triggering inngest/deferred.schedule event metadata.
// ParentRun is nil if the parent row could not be loaded (e.g. pruned).
type RunDeferredFrom struct {
	ParentRunID  ulid.ULID
	ParentFnSlug string
	ParentRun    *TraceRun
}
