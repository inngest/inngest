package conformance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRuntimeDefaultsToServeAndDerivedURLs(t *testing.T) {
	t.Parallel()

	rt, err := (Config{
		SDK: SDKConfig{
			URL: "http://127.0.0.1:3000/api/inngest",
		},
		Dev: DevConfig{
			URL:        "http://127.0.0.1:8288",
			SigningKey: "signkey-test-123",
		},
	}).Runtime()
	require.NoError(t, err)
	require.Equal(t, TransportServe, rt.Transport)
	require.Equal(t, "http://127.0.0.1:3000/api/introspect", rt.IntrospectURL.String())
	require.Equal(t, "http://127.0.0.1:8288", rt.APIURL.String())
	require.Equal(t, "http://127.0.0.1:8288", rt.EventURL.String())
	require.Equal(t, "test", rt.EventKey)
	require.Equal(t, 60*time.Second, rt.Timeout)
}

func TestRuntimeSupportsExplicitOverridePaths(t *testing.T) {
	t.Parallel()

	rt, err := (Config{
		Transport: TransportServe,
		Timeout:   "15s",
		SDK: SDKConfig{
			URL:            "http://127.0.0.1:3000/custom",
			IntrospectPath: "/meta/introspect",
		},
		Dev: DevConfig{
			URL:      "http://127.0.0.1:8288",
			APIURL:   "http://127.0.0.1:8388",
			EventURL: "http://127.0.0.1:8488",
			EventKey: "evt-key",
		},
	}).Runtime()
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:3000/meta/introspect", rt.IntrospectURL.String())
	require.Equal(t, "http://127.0.0.1:8388", rt.APIURL.String())
	require.Equal(t, "http://127.0.0.1:8488", rt.EventURL.String())
	require.Equal(t, "evt-key", rt.EventKey)
	require.Equal(t, 15*time.Second, rt.Timeout)
}
