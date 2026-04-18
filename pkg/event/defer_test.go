package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeferredStartMetadataValidate(t *testing.T) {
	valid := DeferredStartMetadata{
		FnSlug:       "score",
		ParentFnSlug: "app-parent-fn",
		ParentRunID:  "01ABCDEF",
	}

	t.Run("valid", func(t *testing.T) {
		m := valid
		require.NoError(t, m.Validate())
	})

	t.Run("reports all missing fields at once", func(t *testing.T) {
		m := DeferredStartMetadata{}
		err := m.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "fn_slug")
		require.Contains(t, err.Error(), "parent_fn_slug")
		require.Contains(t, err.Error(), "parent_run_id")
	})
}
