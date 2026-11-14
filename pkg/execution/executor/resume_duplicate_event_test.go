package executor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestResumeDuplicateEventDoesNotEnqueueDiscovery covers the case where a
// second event matches a pause that another event already consumed.
//
// ConsumePause returns ErrDuplicateResponse when the stored payload differs
// from the one being written, which the Redis state manager maps to an empty
// ConsumePauseResult with a nil error. DidConsume is therefore false and
// HasPendingSteps is false.
//
// Resume must not enqueue a discovery item in that state. A discovery item
// enqueued against an incomplete round plans the still-pending step again,
// and when that step completes its own discovery is dropped as a duplicate,
// leaving the run with nothing to drive it forward. The run then stays
// Running forever and no timeout fires.
func TestResumeDuplicateEventDoesNotEnqueueDiscovery(t *testing.T) {
	runID := ulid.MustNew(ulid.Now(), nil)
	wsID, fnID, aID := uuid.New(), uuid.New(), uuid.New()

	md := sv2.Metadata{
		ID: sv2.ID{
			RunID:      runID,
			FunctionID: fnID,
			Tenant:     sv2.Tenant{EnvID: wsID, AccountID: aID},
		},
		Config: *sv2.InitConfig(&sv2.Config{}),
	}

	q := &stubQueue{}
	e := &executor{
		smv2: &stubRunServiceMD{md: md},
		pm: &stubPauseMgr{consumeResult: state.ConsumePauseResult{
			DidConsume:      false,
			HasPendingSteps: false,
		}},
		queue:          q,
		log:            logger.From(context.Background()),
		tracerProvider: tracing.NewNoopTracerProvider(),
	}

	pause := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: wsID,
		DataKey:     "wait-step",
		Identifier: state.PauseIdentifier{
			RunID:      runID,
			FunctionID: fnID,
			AccountID:  aID,
		},
		ParallelCoalesceKey: "abc123",
	}

	err := e.Resume(context.Background(), pause, execution.ResumeRequest{})
	require.NoError(t, err)

	// The pause was already consumed by another event, so there is nothing
	// left to discover. Enqueueing a discovery item here wedges the run.
	require.Empty(t, q.enqueued, "no discovery should be enqueued when the pause was not consumed")
}
