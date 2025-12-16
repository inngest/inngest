package pauses

import (
	"errors"

	"github.com/inngest/inngest/pkg/execution/state"
)

func WritePauseRetryableError(err error) bool {
	switch {
	case errors.Is(err, state.ErrSignalConflict):
		return false
	case errors.Is(err, state.ErrPauseAlreadyExists):
		return false
	}

	return true
}
