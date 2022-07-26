package inmemory

import (
	"testing"

	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/testharness"
)

func TestStateHarness(t *testing.T) {
	testharness.CheckState(t, func() (state.Manager, func()) {
		return NewStateManager(), func() {}
	})
}
