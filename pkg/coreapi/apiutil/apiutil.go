package apiutil

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
)

var (
	ErrRunIDInvalid = fmt.Errorf("The run ID specified is invalid")
)

// EventAPIResponse is the API response sent when responding to incoming events.
type EventAPIResponse struct {
	IDs    []string `json:"ids"`
	Status int      `json:"status"`
	Error  error    `json:"error,omitempty"`
}

// CancelRun cancels a run for a given run ID, returning consistent errors for public APIs
func CancelRun(ctx context.Context, sm state.Manager, runID ulid.ULID) error {
	if sm == nil {
		return fmt.Errorf("no state manager supplied to cancel run")
	}

	md, err := sm.Metadata(ctx, runID)
	if err != nil {
		return publicerr.Error{
			Message: "A function run with the given ID could not be found",
			Status:  404,
		}
	}
	switch md.Status {
	case enums.RunStatusFailed, enums.RunStatusCompleted, enums.RunStatusOverflowed:
		return publicerr.Error{
			Message: "This function has already completed",
			Status:  409,
		}
	case enums.RunStatusCancelled:
		return nil
	}
	if err := sm.SetStatus(ctx, md.Identifier, enums.RunStatusCancelled); err != nil {
		return publicerr.Error{
			Message: "There was an error cancelling your function",
			Err:     err,
			Status:  500,
		}
	}
	return nil
}
