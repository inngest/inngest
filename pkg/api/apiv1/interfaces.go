package apiv1

import (
	"context"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

// EventReader represents an event reader for the API.
type EventReader interface {
	// WorkspaceEvents returns the latest events for a given workspace.
	WorkspaceEvents(ctx context.Context, workspaceID uuid.UUID, opts *cqrs.WorkspaceEventsOpts) ([]cqrs.Event, error)
	// Find returns a specific event given an ID.
	FindEvent(ctx context.Context, workspaceID uuid.UUID, id ulid.ULID) (*cqrs.Event, error)
}
