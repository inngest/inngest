package cqrs

import (
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// RunDefer is a single defer attached to a parent function run. RunID is nil
// when the deferred child has not been scheduled yet (parent still running).
// The child function slug is always populated; consumers resolve the child
// function lazily via the slug.
type RunDefer struct {
	HashedDeferID   string            // SHA1-hashed UserlandDeferID
	UserlandDeferID string            // ID provided by the userland SDK caller, ie `defer("foo", ...)` => "foo"
	FnSlug          string            // Slug of the deferred function, "foo-defer"
	Status          enums.DeferStatus // Status of the defer itself, not its associated run
	RunID           *ulid.ULID        // Scheduled child run ID, nil when the child hasn't been scheduled.
}

// RunDeferredFrom describes a parent run that scheduled a function run via
// `defer`. A child can have multiple parents when batching collapses several
// deferred.schedule events into one Schedule call.
//
// FnName and FnSlug ride along on the child's executor.run span so the
// run-list GraphQL resolver can return the parent's Function shape without
// issuing one DB lookup per row. FnSlug is always populated; FnName may be
// empty when scheduling didn't carry a name, in which case the UI falls back
// to FnSlug.
type RunDeferredFrom struct {
	RunID  ulid.ULID
	FnSlug string
	FnName string
}
