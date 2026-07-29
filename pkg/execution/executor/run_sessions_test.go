package executor

import (
	"fmt"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRunSessions(t *testing.T) {
	t.Run("returns nil for no pairs", func(t *testing.T) {
		sessions, dropped := normalizeRunSessions(meta.EventSessions{})
		require.Nil(t, sessions)
		require.Zero(t, dropped)
	})

	t.Run("dedupes identical pairs across events", func(t *testing.T) {
		sessions, dropped := normalizeRunSessions(meta.EventSessions{
			{Key: "conversation_id", ID: "conv_a"},
			{Key: "conversation_id", ID: "conv_a"},
		})
		require.Equal(t, meta.EventSessions{
			{Key: "conversation_id", ID: "conv_a"},
		}, sessions)
		require.Zero(t, dropped)
	})

	t.Run("keeps two IDs under the same key", func(t *testing.T) {
		sessions, dropped := normalizeRunSessions(meta.EventSessions{
			{Key: "conversation_id", ID: "conv_b"},
			{Key: "conversation_id", ID: "conv_a"},
		})
		require.Equal(t, meta.EventSessions{
			{Key: "conversation_id", ID: "conv_a"},
			{Key: "conversation_id", ID: "conv_b"},
		}, sessions)
		require.Zero(t, dropped)
	})

	t.Run("sorts deterministically by key then ID", func(t *testing.T) {
		sessions, _ := normalizeRunSessions(meta.EventSessions{
			{Key: "tenant", ID: "t_1"},
			{Key: "conversation_id", ID: "conv_a"},
			{Key: "workflow_id", ID: "wf_1"},
		})
		require.Equal(t, meta.EventSessions{
			{Key: "conversation_id", ID: "conv_a"},
			{Key: "tenant", ID: "t_1"},
			{Key: "workflow_id", ID: "wf_1"},
		}, sessions)
	})

	t.Run("caps at the per-run limit and reports dropped", func(t *testing.T) {
		pairs := meta.EventSessions{}
		for i := 0; i < consts.MaxRunSessions+3; i++ {
			pairs = append(pairs, meta.EventSession{
				Key: "conversation_id",
				ID:  fmt.Sprintf("conv_%03d", i),
			})
		}

		sessions, dropped := normalizeRunSessions(pairs)
		require.Len(t, sessions, consts.MaxRunSessions)
		require.Equal(t, 3, dropped)
	})
}
