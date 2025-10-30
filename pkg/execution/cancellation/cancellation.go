package cancellation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/oklog/ulid/v2"
)

// Checker checks whether the given function run is cancelled.
type Checker interface {
	// IsCancelled returns the ID of a CancelRequest if the function is cancelled.  If
	// the function is not cancelled the UUID returned will be nil.
	//
	// The event data specified is expected to be the event that triggers a function.
	IsCancelled(ctx context.Context, wsID uuid.UUID, fnID uuid.UUID, runID ulid.ULID, event map[string]any) (*cqrs.Cancellation, error)
}

// Reader returns all cancellations that are valid for the given point in time.
type Reader interface {
	// ReadAt returns cancellations which may cancel functions at the given point in time,
	// for a specific workspace/function.
	ReadAt(ctx context.Context, wsID uuid.UUID, fnID uuid.UUID, at time.Time) ([]cqrs.Cancellation, error)
}

// Writer manages writing cancellations to one or more datastores.
type Writer interface {
	// Write stores new cancellations into a datastore.
	Write(ctx context.Context, c cqrs.Cancellation) error
}

// NewChecker returns a new cancellation checker given a reader.
func NewChecker(r Reader) Checker {
	return checker{r}
}

// checker implements the default checking logic.
type checker struct{ r Reader }

func (c checker) IsCancelled(ctx context.Context, wsID, fnID uuid.UUID, runID ulid.ULID, event map[string]any) (*cqrs.Cancellation, error) {
	if c.r == nil {
		return nil, fmt.Errorf("no cancel loader specified")
	}

	at := ulid.Time(runID.Time())

	all, err := c.r.ReadAt(ctx, wsID, fnID, at)
	if err != nil {
		return nil, err
	}

	for _, i := range all {
		cancel := i

		if at.After(cancel.StartedBefore) || (cancel.StartedAfter != nil && at.Before(*cancel.StartedAfter)) {
			// The loader may have messed up times, so verify that the cancellation includes this
			// run ID as within bounds.
			continue
		}

		if cancel.If == nil {
			return &cancel, nil
		}

		// This cancellation has an expression, and we should only cancel the function
		// if the event that initialized the function matches the expression.
		ok, err := expressions.EvaluateBoolean(ctx, *cancel.If, map[string]any{
			"event": event,
		})
		if err != nil {
			return nil, err
		}
		if ok {
			return &cancel, nil
		}
	}

	// None of the cancellations match.
	return nil, nil
}
