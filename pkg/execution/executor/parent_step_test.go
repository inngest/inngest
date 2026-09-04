package executor

import (
	"testing"

	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

// DiscoveredAfter is how the SDK tells us which steps this one waited for. It
// rides the free-form opts, so these cover the shapes that field arrives in.
func TestDiscoveredAfter(t *testing.T) {
	t.Run("reads the join from opts", func(t *testing.T) {
		op := state.GeneratorOpcode{ID: "c", Opts: map[string]any{
			"discoveredAfter": []any{"a", "b"},
		}}
		require.Equal(t, []string{"a", "b"}, op.DiscoveredAfter())
	})

	t.Run("reads it from opts already marshalled to bytes", func(t *testing.T) {
		op := state.GeneratorOpcode{ID: "c", Opts: []byte(`{"discoveredAfter":["a","b"]}`)}
		require.Equal(t, []string{"a", "b"}, op.DiscoveredAfter())
	})

	// An SDK from before the join was observable reported one ID, not a list.
	// Reading both shapes keeps those runs rendering rather than losing their
	// lineage entirely.
	t.Run("accepts a bare string from an older SDK", func(t *testing.T) {
		op := state.GeneratorOpcode{ID: "b", Opts: map[string]any{"discoveredAfter": "a"}}
		require.Equal(t, []string{"a"}, op.DiscoveredAfter())
	})

	t.Run("empty when the step waits on the trigger rather than a step", func(t *testing.T) {
		// The first fan-out of a run: the SDK reports no parent for these.
		op := state.GeneratorOpcode{ID: "a", Opts: map[string]any{}}
		require.Empty(t, op.DiscoveredAfter())
	})

	t.Run("empty for an SDK that does not report it", func(t *testing.T) {
		op := state.GeneratorOpcode{ID: "a", Opts: map[string]any{"parallelMode": "race"}}
		require.Empty(t, op.DiscoveredAfter())
	})

	t.Run("empty for nil opts", func(t *testing.T) {
		require.Empty(t, (&state.GeneratorOpcode{ID: "a"}).DiscoveredAfter())
	})

	t.Run("ignores malformed opts rather than failing the run", func(t *testing.T) {
		require.Empty(t, (&state.GeneratorOpcode{ID: "a", Opts: []byte("not json")}).DiscoveredAfter())
		require.Empty(t, (&state.GeneratorOpcode{ID: "a", Opts: map[string]any{"discoveredAfter": 42}}).DiscoveredAfter())
		require.Empty(t, (&state.GeneratorOpcode{ID: "a", Opts: map[string]any{"discoveredAfter": ""}}).DiscoveredAfter())
	})

	t.Run("drops non-string members rather than the whole list", func(t *testing.T) {
		op := state.GeneratorOpcode{ID: "c", Opts: map[string]any{
			"discoveredAfter": []any{"a", 42, "", "b"},
		}}
		require.Equal(t, []string{"a", "b"}, op.DiscoveredAfter())
	})
}

// Alternates are the losing side of a race: real orderings, but not
// dependencies. They are reported separately so a race is not drawn as a join.
func TestDiscoveredAfterAlternates(t *testing.T) {
	t.Run("reads the losers of a race", func(t *testing.T) {
		op := state.GeneratorOpcode{ID: "r3", Opts: map[string]any{
			"discoveredAfter":           []any{"r1"},
			"discoveredAfterAlternates": []any{"r2"},
		}}
		require.Equal(t, []string{"r1"}, op.DiscoveredAfter())
		require.Equal(t, []string{"r2"}, op.DiscoveredAfterAlternates())
	})

	t.Run("empty for a plain join", func(t *testing.T) {
		op := state.GeneratorOpcode{ID: "c", Opts: map[string]any{
			"discoveredAfter": []any{"a", "b"},
		}}
		require.Empty(t, op.DiscoveredAfterAlternates())
	})
}
