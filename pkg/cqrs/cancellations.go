package cqrs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type CancellationReadWriter interface {
	CancellationReader
	CancellationWriter
}

// CancellationReader loads cancellations from a backing store.
//
// Note that this is intended for non-critical-path reads, eg. for dashboards
// and management.  Critical path reader interfaces are defined within the
// cancellation package.
type CancellationReader interface {
	// Cancellations returns all cancellations for a given workspace.
	Cancellations(ctx context.Context, wsID uuid.UUID) ([]Cancellation, error)
	Cancellation(ctx context.Context, wsID uuid.UUID, id ulid.ULID) (*Cancellation, error)
	CancellationsByFunction(ctx context.Context, wsID uuid.UUID, fnID uuid.UUID) ([]Cancellation, error)
}

type CancellationWriter interface {
	// CreateCancellation writes a cancellation to the backing store.
	CreateCancellation(ctx context.Context, c Cancellation) error
	// DeleteCancellation deletes a cancellation, immediately preventing the cancellation
	// from stopping functions.
	DeleteCancellation(ctx context.Context, c Cancellation) error
}

// Cancellation represents a cancellation of many function runs during the time specified.
type Cancellation struct {
	CreatedAt   time.Time `json:"created_at"`
	Name        *string   `json:"name"`
	ID          ulid.ULID `json:"id"`
	WorkspaceID uuid.UUID `json:"environment_id"`
	// FunctionID represents the function's internal ID.
	FunctionID uuid.UUID `json:"function_internal_id"`
	// FunctionSlug represents the function's external ID as defined in the SDK.
	FunctionSlug  string     `json:"function_id"`
	StartedAfter  *time.Time `json:"started_after"`
	StartedBefore time.Time  `json:"started_before"`
	If            *string    `json:"if,omitempty"`
	// XXX (tonyhb): We can eventually add a  "kind" field: an enum allowing
	// you to cancel only the backlog of unstarted functions or every function.
}
