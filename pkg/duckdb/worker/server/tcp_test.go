package server

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/worker"
	"github.com/stretchr/testify/require"
)

func TestServerTCPAndHealthz(t *testing.T) {
	srv := New(nil, "v-test-tcp")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ctx, ln)
	}()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-serveErr)
	})

	// Dial with bare host:port AND tcp:// scheme.
	for _, addr := range []string{ln.Addr().String(), "tcp://" + ln.Addr().String()} {
		cl, err := worker.Dial(ctx, addr)
		require.NoError(t, err, "addr=%q", addr)
		ver, err := cl.Ping(ctx)
		require.NoError(t, err)
		require.Equal(t, "v-test-tcp", ver)
		require.NoError(t, cl.Close())
	}

	// Verify HTTP /healthz server reports 200 while open and 503 after Close.
	hln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	haddr := hln.Addr().String()
	require.NoError(t, hln.Close())

	stopHealth, err := startHealthServer(haddr, srv)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stopHealth()) })

	resp, err := http.Get("http://" + haddr + "/healthz")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.NoError(t, srv.Close())
	resp, err = http.Get("http://" + haddr + "/healthz")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}
