package meta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONAttrKeys(t *testing.T) {
	keys := JSONAttrKeys()
	require.Contains(t, keys, Attrs.Sessions.Key())
	require.Contains(t, keys, Attrs.ResponseHeaders.Key())
	require.Contains(t, keys, Attrs.Metadata.Key())
	// Plain string / time / status attributes aren't JSON-encoded.
	require.NotContains(t, keys, Attrs.FunctionSlug.Key())
	require.NotContains(t, keys, Attrs.QueuedAt.Key())
	require.NotContains(t, keys, Attrs.DynamicStatus.Key())
}

func TestStringAttrKeys(t *testing.T) {
	keys := StringAttrKeys()
	// String, UUID and ULID attributes, whether or not they change over a run.
	for _, k := range []string{Attrs.AccountID.Key(), Attrs.RunID.Key(), Attrs.FunctionSlug.Key(), Attrs.AppName.Key(), Attrs.StepID.Key(), Attrs.StepOutput.Key()} {
		require.Contains(t, keys, k)
	}
	// Not string-valued, or (a JsonAttr) unwrapped into JSON by the stores.
	for _, k := range []string{Attrs.QueuedAt.Key(), Attrs.EventIDs.Key(), Attrs.FunctionVersion.Key(), Attrs.Sessions.Key(), Attrs.StepAttempt.Key()} {
		require.NotContains(t, keys, k)
	}
}
