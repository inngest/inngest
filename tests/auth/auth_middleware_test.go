package auth

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/stretchr/testify/require"
)

func TestSigningKeyMiddlewareProtection(t *testing.T) {
	// Test 1: Server without signing key (should allow access)
	t.Run("without_signing_key", func(t *testing.T) {
		testWithoutSigningKey(t)
	})

	// Test 2: Server with signing key (should require auth)
	t.Run("with_signing_key", func(t *testing.T) {
		testWithSigningKey(t)
	})
}

func testWithoutSigningKey(t *testing.T) {
	// Start server without signing key
	cancel := startTestServer(t, nil)
	defer cancel()

	baseURL := "http://localhost:8288"

	// All protected endpoints should be accessible without auth
	endpoints := getProtectedEndpoints()

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			resp, err := makeRequest(endpoint.method, baseURL+endpoint.path, nil)
			require.NoError(t, err)
			// Should not return 401 (may return other errors for invalid data)
			require.NotEqual(t, http.StatusUnauthorized, resp.StatusCode,
				"Endpoint %s should be accessible without signing key", endpoint.path)
		})
	}
}

func testWithSigningKey(t *testing.T) {
	signingKey := "signkey-test-abc123def456"
	cancel := startTestServer(t, &signingKey)
	defer cancel()

	baseURL := "http://localhost:8288"
	endpoints := getProtectedEndpoints()

	for _, endpoint := range endpoints {
		t.Run(endpoint.name+"_no_auth", func(t *testing.T) {
			// Test without auth header - should return 401
			resp, err := makeRequest(endpoint.method, baseURL+endpoint.path, nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode,
				"Endpoint %s should require auth when signing key is configured", endpoint.path)
		})

		t.Run(endpoint.name+"_with_auth", func(t *testing.T) {
			// Test with valid auth header - should not return 401
			headers := map[string]string{
				"Authorization": "Bearer " + signingKey,
			}
			resp, err := makeRequest(endpoint.method, baseURL+endpoint.path, headers)
			require.NoError(t, err)
			require.NotEqual(t, http.StatusUnauthorized, resp.StatusCode,
				"Endpoint %s should be accessible with valid auth", endpoint.path)
		})
	}
}

type TestEndpoint struct {
	name   string
	method string
	path   string
}

func getProtectedEndpoints() []TestEndpoint {
	return []TestEndpoint{
		// DevServer API
		{"register", "POST", "/fn/register"},
		{"remove_app", "DELETE", "/fn/remove"},
		{"set_step_limit", "POST", "/fn/step-limit"},
		{"remove_step_limit", "DELETE", "/fn/step-limit"},
		{"set_state_size_limit", "POST", "/fn/state-size-limit"},
		{"remove_state_size_limit", "DELETE", "/fn/state-size-limit"},

		// API v1
		{"signals", "POST", "/v1/signals"},
		{"events", "GET", "/v1/events"},
		{"event_details", "GET", "/v1/events/test-event-id"},
		{"event_runs", "GET", "/v1/events/test-event-id/runs"},
		{"run_details", "GET", "/v1/runs/test-run-id"},
		{"cancel_run", "DELETE", "/v1/runs/test-run-id"},
		{"run_jobs", "GET", "/v1/runs/test-run-id/jobs"},
		{"app_functions", "GET", "/v1/apps/test-app/functions"},
		{"create_cancellation", "POST", "/v1/cancellations"},
		{"list_cancellations", "GET", "/v1/cancellations"},
		{"delete_cancellation", "DELETE", "/v1/cancellations/test-id"},
		{"prometheus", "GET", "/v1/prom/test-env"},
		{"userland_traces", "POST", "/v1/traces/userland"},

		// Core API
		{"core_cancel_run", "DELETE", "/v0/runs/test-run-id"},
		{"core_run_batch", "GET", "/v0/runs/test-run-id/batch"},
		{"core_run_actions", "GET", "/v0/runs/test-run-id/actions"},
		{"core_telemetry", "POST", "/v0/telemetry"},
	}
}

func startTestServer(t *testing.T, signingKey *string) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	// Get dev config
	conf, err := config.Dev(ctx)
	if err != nil {
		t.Fatalf("Failed to get dev config: %v", err)
	}

	// Create StartOpts
	opts := devserver.StartOpts{
		Config:             *conf,
		Autodiscover:       false,
		Poll:               false,
		PollInterval:       devserver.DefaultPollInterval,
		RetryInterval:      0,
		QueueWorkers:       devserver.DefaultQueueWorkers,
		Tick:               devserver.DefaultTickDuration,
		URLs:               []string{},
		ConnectGatewayPort: devserver.DefaultConnectGatewayPort,
		ConnectGatewayHost: conf.CoreAPI.Addr,
		Persist:            false,
		SigningKey:         signingKey,
		EventKeys:          []string{},
		RequireKeys:        false,
		NoUI:               false,
	}

	// Start server in goroutine
	go func() {
		if err := devserver.New(ctx, opts); err != nil {
			t.Errorf("Failed to start server: %v", err)
		}
	}()

	// Wait for server to start
	if err := waitForPort(ctx, "localhost:8288", 10*time.Second); err != nil {
		cancel()
		t.Fatalf("Server failed to start: %v", err)
	}

	return cancel
}

func waitForPort(ctx context.Context, address string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}

func makeRequest(method, url string, headers map[string]string) (*http.Response, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return client.Do(req)
}
