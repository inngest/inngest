package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

// TestHandleGeneratorGroup_StepErrorHonorsRetryAt mirrors what the SDK sends
// when a step throws RetryAfterError under AsyncCheckpointing:
//
//   - HTTP 206 with header `retry-after: <seconds>` (parsed by httpdriver
//     into DriverResponse.RetryAt)
//   - one OpcodeStepError opcode in the body, with a retryable UserError
//
// handleGeneratorGroup is the function the bug reporter labeled "looks
// correct in code, but observed behavior shows it isn't taking effect."
// This test pins down the behavior they expected: when resp.RetryAt is
// set and HandleGenerator returns ErrHandledStepError, the wrapper must
// produce an error that the queue can read via AsRetryAtError to override
// the default backoff schedule.
func TestHandleGeneratorGroup_StepErrorHonorsRetryAt(t *testing.T) {
	ctx := context.Background()

	// SDK sent retry-after: 600. The httpdriver would parse this into a
	// time ~600s in the future and put it on resp.RetryAt.
	wantRetryAt := time.Now().Add(600 * time.Second).Truncate(time.Second)

	maxAttempts := 5
	jobID := "test-job"
	gen := &state.GeneratorOpcode{
		Op:   enums.OpcodeStepError,
		ID:   "step-1",
		Name: "rate-limited-step",
		Error: &state.UserError{
			Name:    "RetryAfterError",
			Message: "rate limited",
			// NoRetry is false: the SDK also sends x-inngest-no-retry: false,
			// because the user wants a retry — just on a delay.
			NoRetry: false,
		},
	}

	resp := &state.DriverResponse{
		Step:      inngest.Step{ID: "step-1", Name: "rate-limited-step"},
		RetryAt:   &wantRetryAt,
		Generator: []*state.GeneratorOpcode{gen},
	}

	// Minimal runInstance. handleStepError needs:
	//   - r.resp set (for ShouldRetry)
	//   - r.item.Attempt + GetMaxAttempts (for ShouldRetry's attempt+1<max check)
	//   - r.item.Payload as a queue.PayloadEdge (HandleGenerator type-asserts this)
	i := &runInstance{
		resp: resp,
		f:    inngest.Function{},
		item: queue.Item{
			JobID:       &jobID,
			Kind:        queue.KindEdge,
			Attempt:     0,
			MaxAttempts: &maxAttempts,
			Payload:     queue.PayloadEdge{Edge: inngest.SourceEdge},
		},
		c: clockwork.NewRealClock(),
	}

	e := &executor{
		log: logger.From(ctx),
		// no lifecycles, no state store: the retryable branch of
		// handleStepError doesn't touch them.
	}

	group := OpcodeGroup{
		Opcodes:                 []*state.GeneratorOpcode{gen},
		ShouldStartHistoryGroup: false,
	}

	err := e.handleGeneratorGroup(ctx, i, group, resp)

	// We expect a non-nil error: handleStepError returns ErrHandledStepError
	// for retryable step errors, and handleGeneratorGroup wraps it.
	require.Error(t, err, "expected handleGeneratorGroup to return an error for a retryable step error")

	// The queue process loop reads err via AsRetryAtError to override
	// q.backoffFunc(qi.Data.Attempt). That call must find the retry-at
	// for retry-after to be honored. This is the load-bearing assertion
	// for the user's bug.
	specifier := queue.AsRetryAtError(err)
	require.NotNil(t, specifier,
		"AsRetryAtError(err) must find the retry-at; otherwise the queue falls back to BackoffTable",
	)

	gotRetryAt := specifier.NextRetryAt()
	require.NotNil(t, gotRetryAt, "NextRetryAt() must return the retry time, not nil")
	require.True(t, gotRetryAt.Equal(wantRetryAt),
		"NextRetryAt() must equal the SDK-supplied retry-after time:\n  want: %s\n  got:  %s",
		wantRetryAt, *gotRetryAt,
	)

	// service.handleQueueItem dispatches on errors.Is(err, ErrHandledStepError)
	// before the queue's process loop sees it. If the wrap broke that check,
	// the service layer would treat it as a generic error and skip the
	// retry-at extraction entirely. Pin that down too.
	require.ErrorIs(t, err, ErrHandledStepError,
		"the wrapped error must still satisfy errors.Is(err, ErrHandledStepError); service.handleQueueItem branches on this",
	)
}

// TestHandleGeneratorGroup_StepErrorHonorsRetryAt_AfterFmtErrorfWrap is the
// same assertion but applied to err after HandleResponse re-wraps it with
// fmt.Errorf("error handling generator response: %w", serr). That wrap is
// at executor.go:1966 in the path:
//
//	handleGeneratorGroup -> HandleGeneratorResponse -> HandleResponse
//
// %w preserves Unwrap, so AsRetryAtError should still find it. If this
// test fails while the previous one passes, the regression is somewhere
// in the wrap chain rather than in handleGeneratorGroup itself.
func TestHandleGeneratorGroup_StepErrorHonorsRetryAt_AfterFmtErrorfWrap(t *testing.T) {
	ctx := context.Background()
	wantRetryAt := time.Now().Add(600 * time.Second).Truncate(time.Second)

	maxAttempts := 5
	jobID := "test-job"
	gen := &state.GeneratorOpcode{
		Op:   enums.OpcodeStepError,
		ID:   "step-1",
		Name: "rate-limited-step",
		Error: &state.UserError{
			Name:    "RetryAfterError",
			Message: "rate limited",
			NoRetry: false,
		},
	}

	resp := &state.DriverResponse{
		Step:      inngest.Step{ID: "step-1", Name: "rate-limited-step"},
		RetryAt:   &wantRetryAt,
		Generator: []*state.GeneratorOpcode{gen},
	}

	i := &runInstance{
		resp: resp,
		f:    inngest.Function{},
		item: queue.Item{
			JobID:       &jobID,
			Kind:        queue.KindEdge,
			Attempt:     0,
			MaxAttempts: &maxAttempts,
			Payload:     queue.PayloadEdge{Edge: inngest.SourceEdge},
		},
		c: clockwork.NewRealClock(),
	}

	e := &executor{log: logger.From(ctx)}

	group := OpcodeGroup{
		Opcodes:                 []*state.GeneratorOpcode{gen},
		ShouldStartHistoryGroup: false,
	}

	innerErr := e.handleGeneratorGroup(ctx, i, group, resp)
	require.Error(t, innerErr)

	// Re-wrap exactly as HandleResponse does at executor.go:1966.
	wrapped := fmt.Errorf("error handling generator response: %w", innerErr)

	specifier := queue.AsRetryAtError(wrapped)
	require.NotNil(t, specifier, "AsRetryAtError must traverse the fmt.Errorf %%w chain")
	require.True(t, specifier.NextRetryAt().Equal(wantRetryAt))
	require.ErrorIs(t, wrapped, ErrHandledStepError)
}
