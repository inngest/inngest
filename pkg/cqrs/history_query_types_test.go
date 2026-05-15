package cqrs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCountActiveRunsOptsValidate(t *testing.T) {
	t.Run("requires lower time without panicking", func(t *testing.T) {
		opts := CountActiveRunsOpts{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			WorkflowID:  uuid.New(),
			UpperTime:   time.Now(),
		}

		err := opts.Validate()
		require.EqualError(t, err, "lower time must be provided")
	})

	t.Run("rejects upper time before lower time", func(t *testing.T) {
		lower := time.Now()
		opts := CountActiveRunsOpts{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			WorkflowID:  uuid.New(),
			LowerTime:   &lower,
			UpperTime:   lower.Add(-time.Second),
		}

		err := opts.Validate()
		require.EqualError(t, err, "upper/end time must be after lower/start time")
	})
}
