package metrics

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/authn"
)

var testSigningKey = "signkey-test-abc123def456"

// integrationQueueManager is a test queue manager for integration tests
type integrationQueueManager struct {
	queueDepth int64
}

func (m *integrationQueueManager) TotalSystemQueueDepth(ctx context.Context) (int64, error) {
	return atomic.LoadInt64(&m.queueDepth), nil
}

func (m *integrationQueueManager) SetQueueDepth(depth int64) {
	atomic.StoreInt64(&m.queueDepth, depth)
}

func TestMetricsAPIIntegration(t *testing.T) {
	tests := []struct {
		name           string
		queueDepth     int64
		withAuth       bool
		authHeader     string
		expectedStatus int
		checkResponse  func(t *testing.T, body string, status int)
	}{
		{
			name:           "successful metrics with empty queue",
			queueDepth:     0,
			withAuth:       false,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, status int) {
				assert.Contains(t, body, "# HELP inngest_queue_depth")
				assert.Contains(t, body, "# TYPE inngest_queue_depth gauge")
				assert.Contains(t, body, "inngest_queue_depth 0")
			},
		},
		{
			name:           "successful metrics with populated queue",
			queueDepth:     5,
			withAuth:       false,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, status int) {
				assert.Contains(t, body, "inngest_queue_depth 5")
			},
		},
		{
			name:           "unauthorized access with signing key",
			queueDepth:     0,
			withAuth:       true,
			authHeader:     "", // No auth header
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string, status int) {
				assert.Contains(t, body, "Authentication failed")
			},
		},
		{
			name:           "successful auth with valid signing key",
			queueDepth:     10,
			withAuth:       true,
			authHeader:     generateValidAuthHeader(testSigningKey),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, status int) {
				assert.Contains(t, body, "inngest_queue_depth 10")
			},
		},
		{
			name:           "invalid auth header format",
			queueDepth:     0,
			withAuth:       true,
			authHeader:     "invalid-header-format",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string, status int) {
				// Should contain auth error
			},
		},
		{
			name:           "large queue depth",
			queueDepth:     1000,
			withAuth:       false,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, status int) {
				assert.Contains(t, body, "inngest_queue_depth 1000")
			},
		},
		{
			name:           "negative queue depth sanitized",
			queueDepth:     -100,
			withAuth:       false,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, status int) {
				// Should be sanitized to 0
				assert.Contains(t, body, "inngest_queue_depth 0")
			},
		},
		{
			name:           "excessive queue depth clamped",
			queueDepth:     MaxReasonableQueueDepth + 5000,
			withAuth:       false,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, status int) {
				// Should be clamped to reasonable limit
				assert.Contains(t, body, fmt.Sprintf("inngest_queue_depth %v", float64(MaxReasonableQueueDepth)))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			ctx := context.Background()

			// Create queue manager with test data
			queueManager := &integrationQueueManager{}
			queueManager.SetQueueDepth(tt.queueDepth)

			// Create metrics API
			opts := Opts{
				QueueManager: queueManager,
			}

			if tt.withAuth {
				opts.AuthMiddleware = authn.SigningKeyMiddleware(&testSigningKey)
			}

			api, err := NewMetricsAPI(opts)
			require.NoError(t, err)

			// Start test server
			server := &http.Server{
				Handler: api.Router,
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)

			port := listener.Addr().(*net.TCPAddr).Port
			baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

			go func() {
				_ = server.Serve(listener)
			}()
			defer func() {
				_ = server.Shutdown(ctx)
			}()

			// Wait for server to start
			time.Sleep(100 * time.Millisecond)

			// Make request
			req, err := http.NewRequest(http.MethodGet, baseURL+"/", nil)
			require.NoError(t, err)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Read response
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// Check status
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Check response body
			if tt.checkResponse != nil {
				tt.checkResponse(t, string(body), resp.StatusCode)
			}

			// Check content type for successful responses
			if resp.StatusCode == http.StatusOK {
				contentType := resp.Header.Get("Content-Type")
				assert.Contains(t, contentType, "text/plain")
			}
		})
	}
}

func TestMetricsAPIPerformance(t *testing.T) {
	t.Run("response time under load", func(t *testing.T) {
		ctx := context.Background()

		// Create queue manager with large queue
		queueManager := &integrationQueueManager{}
		queueManager.SetQueueDepth(10000)

		// Create API
		api, err := NewMetricsAPI(Opts{
			QueueManager: queueManager,
		})
		require.NoError(t, err)

		// Start server
		server := &http.Server{Handler: api.Router}
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		port := listener.Addr().(*net.TCPAddr).Port
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

		go func() {
			_ = server.Serve(listener)
		}()
		defer func() {
			_ = server.Shutdown(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		// Test multiple concurrent requests
		const numRequests = 10
		results := make(chan time.Duration, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				start := time.Now()

				resp, err := http.Get(baseURL + "/")
				if err != nil {
					results <- time.Duration(-1)
					return
				}
				defer resp.Body.Close()

				duration := time.Since(start)
				results <- duration
			}()
		}

		// Collect results
		var totalDuration time.Duration
		successCount := 0

		for i := 0; i < numRequests; i++ {
			duration := <-results
			if duration > 0 {
				totalDuration += duration
				successCount++
			}
		}

		require.Greater(t, successCount, 0)
		avgDuration := totalDuration / time.Duration(successCount)

		// Response should be reasonably fast (under 1 second)
		assert.Less(t, avgDuration, time.Second, "Average response time should be under 1 second")
	})
}

func TestMetricsAPIPrometheusFormat(t *testing.T) {
	t.Run("prometheus format validation", func(t *testing.T) {
		ctx := context.Background()

		// Create queue manager with test data
		queueManager := &integrationQueueManager{}
		queueManager.SetQueueDepth(42)

		api, err := NewMetricsAPI(Opts{
			QueueManager: queueManager,
		})
		require.NoError(t, err)

		// Start server
		server := &http.Server{Handler: api.Router}
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		port := listener.Addr().(*net.TCPAddr).Port
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

		go func() {
			_ = server.Serve(listener)
		}()
		defer func() {
			_ = server.Shutdown(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		// Make request
		resp, err := http.Get(baseURL + "/")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		bodyStr := string(body)

		// Validate Prometheus format
		lines := strings.Split(bodyStr, "\n")

		// Should have HELP line
		helpFound := false
		typeFound := false
		metricFound := false

		for _, line := range lines {
			if strings.HasPrefix(line, "# HELP inngest_queue_depth") {
				helpFound = true
				assert.Contains(t, line, "Total depth of all system queues")
			} else if strings.HasPrefix(line, "# TYPE inngest_queue_depth") {
				typeFound = true
				assert.Contains(t, line, "gauge")
			} else if strings.HasPrefix(line, "inngest_queue_depth ") {
				metricFound = true
				assert.Contains(t, line, "42")
			}
		}

		assert.True(t, helpFound, "Should contain HELP comment")
		assert.True(t, typeFound, "Should contain TYPE comment")
		assert.True(t, metricFound, "Should contain metric value")

		// Check content type
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
	})
}

func generateValidAuthHeader(signingKey string) string {
	// This is a simplified auth header generation
	// In reality, you'd need to create a proper JWT or signature
	// For this test, we'll use a simple Bearer token format
	return fmt.Sprintf("Bearer %s", signingKey)
}
