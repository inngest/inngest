package apiv2endpoint_test

import (
	"testing"

	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/stretchr/testify/require"
)

func TestIncludeSelector(t *testing.T) {
	selector := apiv2endpoint.NewIncludeSelector(map[string]string{
		"deferredFrom":  "deferredFrom",
		"deferred_from": "deferredFrom",
	})

	for _, value := range []string{"deferredFrom", "deferred_from"} {
		got, err := selector.Parse(value)
		require.NoError(t, err)
		require.Equal(t, "deferredFrom", got)
	}

	_, err := selector.Parse("unknown")
	require.ErrorContains(t, err, `unsupported include value "unknown"`)
}
