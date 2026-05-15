package event

import (
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeferredScheduleMetadataValidate(t *testing.T) {
	valid := DeferredScheduleMetadata{
		FnSlug:       "score",
		ParentFnSlug: "app-parent-fn",
		ParentRunID:  "01ABCDEF",
		HashedDeferID: "deadbeef",
	}

	t.Run("valid", func(t *testing.T) {
		m := valid
		require.NoError(t, m.Validate())
	})

	t.Run("reports all missing fields at once", func(t *testing.T) {
		m := DeferredScheduleMetadata{}
		err := m.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "fn_slug")
		require.Contains(t, err.Error(), "parent_fn_slug")
		require.Contains(t, err.Error(), "parent_run_id")
		require.Contains(t, err.Error(), "defer_id")
	})
}

// TestEventDeferredScheduleMetadata covers the typed-vs-untyped envelope
// branches in (*Event).DeferredScheduleMetadata so any future change to how
// metadata is round-tripped (e.g. through Redis/event JSON) is caught.
func TestEventDeferredScheduleMetadata(t *testing.T) {
	wantMeta := DeferredScheduleMetadata{
		FnSlug:       "child-fn",
		ParentFnSlug: "parent-fn",
		ParentRunID:  "01ABCDEF",
		HashedDeferID: "deadbeef",
	}

	t.Run("missing _inngest prefix returns an error", func(t *testing.T) {
		e := Event{Data: map[string]any{"x": 1}}
		got, err := e.DeferredScheduleMetadata()
		require.Error(t, err)
		assert.Contains(t, err.Error(), consts.InngestEventDataPrefix)
		assert.Nil(t, got)
	})

	t.Run("decodes a map[string]any payload", func(t *testing.T) {
		e := Event{Data: map[string]any{
			consts.InngestEventDataPrefix: map[string]any{
				"fn_slug":        wantMeta.FnSlug,
				"parent_fn_slug": wantMeta.ParentFnSlug,
				"parent_run_id":  wantMeta.ParentRunID,
				"defer_id":       wantMeta.HashedDeferID,
			},
		}}
		got, err := e.DeferredScheduleMetadata()
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, wantMeta, *got)
	})

	t.Run("decodes a DeferredScheduleMetadata value (typed envelope)", func(t *testing.T) {
		e := Event{Data: map[string]any{consts.InngestEventDataPrefix: wantMeta}}
		got, err := e.DeferredScheduleMetadata()
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, wantMeta, *got)
	})

	t.Run("decodes a *DeferredScheduleMetadata pointer", func(t *testing.T) {
		m := wantMeta // copy
		e := Event{Data: map[string]any{consts.InngestEventDataPrefix: &m}}
		got, err := e.DeferredScheduleMetadata()
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, wantMeta, *got)
	})

	t.Run("bad payload returns an error", func(t *testing.T) {
		// json.Marshal succeeds on any value but Unmarshal into the struct
		// fails when the source is not an object (e.g. a string).
		e := Event{Data: map[string]any{consts.InngestEventDataPrefix: "not-an-object"}}
		got, err := e.DeferredScheduleMetadata()
		require.Error(t, err)
		assert.Nil(t, got)
	})
}
