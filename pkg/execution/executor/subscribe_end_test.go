package executor

import (
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/require"
)

// A read ends in one of three ways, and its row says which: a message answered
// it (completed), its timeout passed (timed out, and the wait expired), or its
// inbox was closed with nothing to read (cancelled, which is what a read that
// lost a race ends as).
func TestInboxReadEndState(t *testing.T) {
	t.Run("a message completes the read", func(t *testing.T) {
		status, expired := inboxReadEnd(sv2.InboxWake{})
		require.Equal(t, enums.StepStatusCompleted, status)
		require.False(t, expired)
	})

	t.Run("a timeout times the read out", func(t *testing.T) {
		status, expired := inboxReadEnd(sv2.InboxWake{TimedOut: true})
		require.Equal(t, enums.StepStatusTimedOut, status)
		require.True(t, expired)
	})

	t.Run("a close cancels the read", func(t *testing.T) {
		status, expired := inboxReadEnd(sv2.InboxWake{Closed: true})
		require.Equal(t, enums.StepStatusCancelled, status)
		require.False(t, expired)
	})
}
