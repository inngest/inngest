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
