package apiv2endpoint

import (
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
)

func TestIncludeSelector(t *testing.T) {
	method := apiv2.File_api_v2_service_proto.Services().ByName("V2").Methods().ByName("ListRuns")
	selector := newIncludeSelector(method, []string{"output", "deferredFrom"})

	for value, expected := range map[string]string{
		"output":        "output",
		"deferredFrom":  "deferredFrom",
		"deferred_from": "deferredFrom",
	} {
		t.Run(value, func(t *testing.T) {
			got, err := selector.Parse(value)
			require.NoError(t, err)
			require.Equal(t, expected, got)
		})
	}

	_, err := selector.Parse("unknown")
	require.ErrorContains(t, err, `unsupported include value "unknown"`)
	_, err = selector.Parse("id")
	require.ErrorContains(t, err, `unsupported include value "id"`)
}
