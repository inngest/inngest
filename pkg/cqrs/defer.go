package cqrs

import (
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// RunDefer is a single defer attached to a parent function run. Parsed from
// the parent's history opcodes (DeferAdd/DeferAbort) and joined to the
// resulting child run when one exists. Run is nil if the defer was aborted,
// the schedule event has not yet been processed, or the child trace was
// pruned.
type RunDefer struct {
	HashedDeferID string            // SHA1-hashed UserDeferID
	UserDeferID   string            // ID provided by the user, ie `defer("foo", ...)` => "foo"
	FnSlug        string            // Slug of the deferred function, "foo-defer"
	Status        enums.DeferStatus // status of the defer itself, not the potentially associated run
	Run           *TraceRun
}

type RunDeferredFrom struct {
	ParentRunID ulid.ULID
	ParentRun   *TraceRun
}

// RunInvokedFrom is the reverse-lookup for a function run that was triggered
// by a parent run's `step.invoke`. The linkage is reconstructed from the
// parent's invoke step span: an EXTEND fragment carries
// `_inngest.step.invoke.run.id` (the child run ID), and the original
// `executor.step` fragment sharing the same `dynamic_span_id` carries the
// step display name.
type RunInvokedFrom struct {
	ParentRunID ulid.ULID
	ParentRun   *TraceRun
	StepName    *string
}
