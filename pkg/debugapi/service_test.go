package debugapi

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

// TestServerRunStopsWithContext checks that Run returns when its context is
// cancelled.  the stop of the app waits for Run to return.
func TestServerRunStopsWithContext(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())

	server := NewDebugAPI(Opts{
		Log:  logger.VoidLogger(),
		Port: port,
	})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", l.Addr().String())
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 5*time.Second, 10*time.Millisecond)

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after the context was cancelled")
	}
}
