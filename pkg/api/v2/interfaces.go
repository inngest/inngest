package apiv2

import (
	"context"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
)

type FunctionProvider interface {
	// GetFunction returns a function given its slug or ID.
	GetFunction(ctx context.Context, identifier string) (inngest.DeployedFunction, error)
}

type FunctionScheduler interface {
	Schedule(ctx context.Context, req execution.ScheduleRequest) (*sv2.Metadata, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event event.TrackedEvent) error
}
