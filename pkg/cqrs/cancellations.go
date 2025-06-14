package cqrs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
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
	CreatedAt time.Time `json:"created_at"`
	Name      *string   `json:"name"`
	ID        ulid.ULID `json:"id"`

	// Identifiers
	AccountID   uuid.UUID `json:"account_id"`
	WorkspaceID uuid.UUID `json:"environment_id"`
	AppID       uuid.UUID `json:"app_id"`
	// FunctionID represents the function's internal ID.
	FunctionID uuid.UUID `json:"function_internal_id"`
	// FunctionSlug represents the function's external ID as defined in the SDK.
	FunctionSlug string `json:"function_id"`

	// Timing based attributes
	StartedAfter  *time.Time `json:"started_after"`
	StartedBefore time.Time  `json:"started_before"`
	If            *string    `json:"if,omitempty"`

	// Kind represents the kind of cancellation. e.g. run, backlog, etc
	Kind enums.CancellationKind `json:"kind"`
	// Type represents the type of the cancellation
	Type enums.CancellationType `json:"cause"`
	// TargetID presents the ID of the target that needs to be cancelled
	// e.g.
	// - run -> runID
	// - bulk run -> none
	// - backlog -> backlogID
	TargetID string `json:"target_id,omitempty"`

	// QueueName represents a specific queue name if known before hand.
	// NOTE: This is used mainly for cancelling items in system queues, and generally shouldn't be exposed to users
	QueueName *string `json:"queue_name"`
}
