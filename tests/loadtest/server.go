package loadtest

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/config"
	connectConfig "github.com/inngest/inngest/pkg/config/connect"
	"github.com/inngest/inngest/pkg/devserver"
)

// ServerHandle wraps a running dev server instance.
type ServerHandle struct {
	Cancel  context.CancelFunc
	BaseURL string
	Port    int
}

// StartServer launches a dev server with the given SystemConfig.
// It finds free ports, starts the server in a goroutine, and waits until
// the port is accepting connections.
func StartServer(t testing.TB, cfg SystemConfig) *ServerHandle {
	t.Helper()

	// Find free ports for the various services.
	apiPort := findFreePort(t)
	gatewayPort := findFreePort(t)
	gatewayGRPCPort := findFreePort(t)
	executorGRPCPort := findFreePort(t)
	debugPort := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())

	// Set environment variables from config.
	var envCleanups []func()
	for k, v := range cfg.EnvVars {
		key := k
		prev, had := os.LookupEnv(key)
		os.Setenv(key, v)
		if had {
			p := prev
			envCleanups = append(envCleanups, func() { os.Setenv(key, p) })
		} else {
			envCleanups = append(envCleanups, func() { os.Unsetenv(key) })
		}
	}

	// Get dev config.
	conf, err := config.Dev(ctx)
	if err != nil {
		cancel()
		t.Fatalf("loadtest: failed to get dev config: %v", err)
	}
	// The main HTTP server uses EventAPI.Port for binding.
	// CoreAPI.Port is used for internal service URLs (connect, realtime).
	conf.EventAPI.Port = apiPort
	conf.CoreAPI.Port = apiPort

	queueWorkers := cfg.QueueWorkers
	if queueWorkers == 0 {
		queueWorkers = devserver.DefaultQueueWorkers
	}
	tick := cfg.Tick
	if tick == 0 {
		tick = devserver.DefaultTickDuration
	}

	opts := devserver.StartOpts{
		Config:       *conf,
		Autodiscover: false,
		Poll:         false,
		PollInterval: devserver.DefaultPollInterval,
		QueueWorkers: queueWorkers,
		Tick:         tick,
		URLs:         []string{},
		Persist:      false,
		EventKeys:    []string{"test"},
		RequireKeys:  false,
		NoUI:         true,

		ConnectGatewayPort: gatewayPort,
		ConnectGatewayHost: conf.CoreAPI.Addr,
		ConnectGRPCConfig: connectConfig.ConnectGRPCConfig{
			Gateway:  connectConfig.GRPCConfig{Port: gatewayGRPCPort},
			Executor: connectConfig.GRPCConfig{Port: executorGRPCPort},
		},
		DebugAPIPort: debugPort,
	}

	// Start server in a goroutine.
	go func() {
		if err := devserver.New(ctx, opts); err != nil {
			// Only log if context wasn't cancelled (normal shutdown).
			if ctx.Err() == nil {
				t.Errorf("loadtest: dev server error: %v", err)
			}
		}
	}()

	// Wait for the API port to become available.
	addr := fmt.Sprintf("127.0.0.1:%d", apiPort)
	if err := waitForPort(ctx, addr, 30*time.Second); err != nil {
		cancel()
		t.Fatalf("loadtest: dev server failed to start on %s: %v", addr, err)
	}

	t.Cleanup(func() {
		cancel()
		for _, fn := range envCleanups {
			fn()
		}
		// Give the server a moment to shut down.
		time.Sleep(100 * time.Millisecond)
	})

	return &ServerHandle{
		Cancel:  cancel,
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", apiPort),
		Port:    apiPort,
	}
}

func findFreePort(t testing.TB) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("loadtest: failed to find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func waitForPort(ctx context.Context, address string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s", address)
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}
