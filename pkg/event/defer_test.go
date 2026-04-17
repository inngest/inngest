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

	t.Run("missing fn_slug", func(t *testing.T) {
		m := valid
		m.FnSlug = ""
		err := m.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "fn_slug")
	})

	t.Run("missing parent_fn_slug", func(t *testing.T) {
		m := valid
		m.ParentFnSlug = ""
		err := m.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "parent_fn_slug")
	})

	t.Run("missing parent_run_id", func(t *testing.T) {
		m := valid
		m.ParentRunID = ""
		err := m.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "parent_run_id")
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
