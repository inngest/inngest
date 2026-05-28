package event

import (
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeferredScheduleMetadataValidate(t *testing.T) {
	valid := DeferredScheduleMetadata{
		FnSlug:           "score",
		ParentFnSlug:     "app-parent-fn",
		ParentRunID:      "01ABCDEF",
		ParentFunctionID: "00000000-0000-0000-0000-000000000001",
		HashedDeferID:    "deadbeef",
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
		require.Contains(t, err.Error(), "parent_function_id")
		require.Contains(t, err.Error(), "hashed_defer_id")
	})

	t.Run("rejects an unparseable parent_function_id", func(t *testing.T) {
		// Without this, a malformed UUID would slip through and be persisted
		// on the parent-side defer span, breaking per-function span queries.
		m := valid
		m.ParentFunctionID = "not-a-uuid"
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parent_function_id")
	})

	t.Run("rejects the zero uuid for parent_function_id", func(t *testing.T) {
		// uuid.Parse accepts "00000000-...000" as valid; we reject it
		// explicitly so a zero-value FunctionID can't stamp uuid.Nil onto
		// the persisted span (where it would disappear from indexes).
		m := valid
		m.ParentFunctionID = "00000000-0000-0000-0000-000000000000"
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parent_function_id")
	})
}

func TestEventDeferredScheduleMetadata(t *testing.T) {
	wantMeta := DeferredScheduleMetadata{
		FnSlug:           "child-fn",
		ParentFnSlug:     "parent-fn",
		ParentRunID:      "01ABCDEF",
		ParentFunctionID: "00000000-0000-0000-0000-000000000001",
		HashedDeferID:    "deadbeef",
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
				"fn_slug":             wantMeta.FnSlug,
				"parent_fn_slug":      wantMeta.ParentFnSlug,
				"parent_run_id":       wantMeta.ParentRunID,
				"parent_function_id":  wantMeta.ParentFunctionID,
				"hashed_defer_id":     wantMeta.HashedDeferID,
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

	t.Run("bad payload returns an error", func(t *testing.T) {
		// json.Marshal succeeds on any value but Unmarshal into the struct
		// fails when the source is not an object.
		e := Event{Data: map[string]any{consts.InngestEventDataPrefix: "not-an-object"}}
		got, err := e.DeferredScheduleMetadata()
		require.Error(t, err)
		assert.Nil(t, got)
	})
}
