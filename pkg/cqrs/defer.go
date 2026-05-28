package cqrs

import (
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// RunDefer is a single defer attached to a parent function run. Run is nil
// if the child run has not yet been recorded.
type RunDefer struct {
	HashedDeferID   string            // SHA1-hashed UserlandDeferID
	UserlandDeferID string            // ID provided by the userland SDK caller, ie `defer("foo", ...)` => "foo"
	FnSlug          string            // Slug of the deferred function, "foo-defer"
	Status          enums.DeferStatus // Status of the defer itself, not its associated run
	Run             *TraceRun
}

// RunDeferredFrom describes the parent run that scheduled a function run via
// `defer`.
type RunDeferredFrom struct {
	ParentRunID ulid.ULID
	ParentRun   *TraceRun
}

// RunInvokedFrom describes the parent run that triggered a function run via
// `step.invoke`.
type RunInvokedFrom struct {
	ParentRunID ulid.ULID
	ParentRun   *TraceRun
	StepName    *string
}
