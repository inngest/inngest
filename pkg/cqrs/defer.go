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
// The struct exposes only the parent's identifiers. Consumers that need the
// parent run or function fetch them lazily (via GraphQL resolvers, etc.) so
// the read path doesn't pay for joins the caller may not use.
type RunDeferredFrom struct {
	RunID  ulid.ULID
	FnSlug string
}
