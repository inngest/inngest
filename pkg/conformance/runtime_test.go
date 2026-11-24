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
	require.Equal(t, 3*time.Minute, rt.Timeout)
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

func TestRuntimeHonorsExplicitTimeout(t *testing.T) {
	t.Parallel()

	for _, timeout := range []string{"50ms", "60s", "2m", "5m"} {
		t.Run(timeout, func(t *testing.T) {
			rt, err := (Config{Timeout: timeout}).Runtime()
			require.NoError(t, err)
			expected, err := time.ParseDuration(timeout)
			require.NoError(t, err)
			require.Equal(t, expected, rt.Timeout)
		})
	}
}

func TestRuntimeDefaultCoversRetryBudget(t *testing.T) {
	t.Parallel()

	rt, err := (Config{}).Runtime()
	require.NoError(t, err)
	// retry-basic permits two retry waits and six other request/response
	// waits. Trigger submission can also use the client's 30-second timeout.
	retryWaits := 2 * 45 * time.Second
	requestAndResponseWaits := 6 * 5 * time.Second
	triggerTimeout := 30 * time.Second
	require.Greater(t, rt.Timeout, retryWaits+requestAndResponseWaits+triggerTimeout)
}
