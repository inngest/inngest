package grpc

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveConnectGRPCIP(t *testing.T) {
	t.Run("uses INNGEST_CONNECT_GATEWAY_GRPC_IP when set", func(t *testing.T) {
		t.Setenv("INNGEST_CONNECT_GATEWAY_GRPC_IP", "10.1.2.3")
		require.Equal(t, "10.1.2.3", resolveConnectGRPCIP())
	})

	t.Run("falls back to a routable IP when the env var is unset", func(t *testing.T) {
		// Empty means unset for our purposes; this exercises the auto-detect /
		// loopback-fallback path. Either way the result must be a valid IP.
		t.Setenv("INNGEST_CONNECT_GATEWAY_GRPC_IP", "")
		got := resolveConnectGRPCIP()
		require.NotEmpty(t, got)
		require.NotNil(t, net.ParseIP(got), "expected a valid IP, got %q", got)
	})

	t.Run("ignores an invalid override and falls through to a valid IP", func(t *testing.T) {
		t.Setenv("INNGEST_CONNECT_GATEWAY_GRPC_IP", "not-an-ip")
		got := resolveConnectGRPCIP()
		require.NotEqual(t, "not-an-ip", got)
		require.NotNil(t, net.ParseIP(got), "expected a valid IP, got %q", got)
	})
}
