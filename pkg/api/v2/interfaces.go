package apiv2

import (
	"context"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

type FunctionProvider interface {
	// GetFunction returns a function given its slug OR ID.
	GetFunction(ctx context.Context, identifier string) (inngest.DeployedFunction, error)
}

type FunctionScheduler interface {
	// Schedule initializes a new function run, ensuring that the function will be
	// executed via our async execution engine as quickly as possible.
	//
	// This returns a run ID, metadata for the run, and any errors scheduling.
	//
	// If the run was impacted by flow control (idempotency, rate limiting, debounce, etc.),
	// metadata will be nil.  This will return the original run ID if runs were skipped due
	// to idemptoency.
	Schedule(ctx context.Context, req execution.ScheduleRequest) (ulid.ULID, *sv2.Metadata, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event event.TrackedEvent) error
}
