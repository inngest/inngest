package cqrs

import (
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// RunDefer is a single defer attached to a parent function run. RunID is nil
// when the deferred child has not been scheduled yet (parent still running)
// or when the defer was aborted before scheduling. The child function slug is
// always populated; consumers resolve the child function lazily via the slug.
type RunDefer struct {
	// Hashed `RunDeferResolver`
	HashedDeferID string

	// Defer ID passed to `defer()` call in the Inngest function
	UserlandDeferID string

	// Deferred function slug
	FnSlug string

	// Status of the defer (not the child run)
	Status enums.DeferStatus

	// Scheduled child run ID, nil when the child hasn't been scheduled.
	RunID *ulid.ULID
}
