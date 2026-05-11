package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeferredScheduleMetadataValidate(t *testing.T) {
	valid := DeferredScheduleMetadata{
		FnSlug:       "score",
		ParentFnSlug: "app-parent-fn",
		ParentRunID:  "01ABCDEF",
		DeferID:      "deadbeef",
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
