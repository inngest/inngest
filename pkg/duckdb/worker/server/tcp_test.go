package server

import (
	"context"
	"net"
	"net/http"
	"os"
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

func TestServerTCPNonLoopbackRequiresAuthToken(t *testing.T) {
	srv := New(nil, "v-test-tcp")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, addr := range []string{":0", "0.0.0.0:0", "tcp://0.0.0.0:9090", "tcp://[::]:9090"} {
		err := srv.ListenAndServe(ctx, addr)
		require.Error(t, err, "addr=%q", addr)
		require.Contains(t, err.Error(), "--auth-token")
	}

	require.True(t, isLoopbackHost("127.0.0.1"))
	require.True(t, isLoopbackHost("::1"))
	require.True(t, isLoopbackHost("localhost"))
	require.True(t, isLoopbackHost("LOCALHOST"))
	require.False(t, isLoopbackHost(""))
	require.False(t, isLoopbackHost("0.0.0.0"))
	require.False(t, isLoopbackHost("::"))
	require.False(t, isLoopbackHost("192.168.1.10"))
}

func TestServerTCPAuthToken(t *testing.T) {
	srv := New(nil, "v-test-auth")
	srv.SetAuthToken("super-secret")
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

	addr := ln.Addr().String()

	_, err = worker.Dial(ctx, addr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")

	_, err = worker.DialWithToken(ctx, addr, "wrong-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")

	cl, err := worker.DialWithToken(ctx, addr, "super-secret")
	require.NoError(t, err)
	ver, err := cl.Ping(ctx)
	require.NoError(t, err)
	require.Equal(t, "v-test-auth", ver)
	require.NoError(t, cl.Close())
}

func TestServerPingReturnsPID(t *testing.T) {
	srv := New(nil, "v-test-pid")
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

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, worker.WriteFrame(conn, worker.Request{Op: worker.OpPing}))
	var resp worker.Response
	require.NoError(t, worker.ReadFrame(conn, &resp))
	require.Equal(t, worker.KindOK, resp.Kind)
	require.Equal(t, "v-test-pid", resp.Version)
	require.Equal(t, os.Getpid(), resp.PID)
}
