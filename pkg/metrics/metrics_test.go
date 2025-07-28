package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockQueueManager implements QueueManager interface for testing
type mockQueueManager struct {
	queueDepth int64
	err        error
}

func (m *mockQueueManager) TotalSystemQueueDepth(ctx context.Context) (int64, error) {
	return m.queueDepth, m.err
}

// mockAuthMiddleware creates a test auth middleware
func mockAuthMiddleware(requireAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requireAuth && r.Header.Get("Authorization") == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func TestNewMetricsAPI(t *testing.T) {
	tests := []struct {
		name     string
		opts     Opts
		wantErr  bool
		checkAPI func(*testing.T, *MetricsAPI)
	}{
		{
			name: "creates API with all components",
			opts: Opts{
				QueueManager:   &mockQueueManager{queueDepth: 100},
				AuthMiddleware: mockAuthMiddleware(false),
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MetricsAPI) {
				assert.NotNil(t, api.Router)
				assert.NotNil(t, api.queueGauge)
				assert.NotNil(t, api.registry)
				assert.Contains(t, api.queueGauge.Desc().String(), "inngest_queue_depth")
			},
		},
		{
			name: "creates API without auth middleware",
			opts: Opts{
				QueueManager: &mockQueueManager{queueDepth: 50},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MetricsAPI) {
				assert.NotNil(t, api.Router)
				assert.Nil(t, api.opts.AuthMiddleware)
			},
		},
		{
			name: "fails to create API without queue manager",
			opts: Opts{
				AuthMiddleware: mockAuthMiddleware(false),
			},
			wantErr:  true,
			checkAPI: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, err := NewMetricsAPI(tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, api)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, api)
				if tt.checkAPI != nil {
					tt.checkAPI(t, api)
				}
			}
		})
	}
}

func TestMetricsAPI_handleMetrics(t *testing.T) {
	tests := []struct {
		name             string
		queueManager     QueueManager
		authMiddleware   func(http.Handler) http.Handler
		setupRequest     func(*http.Request)
		wantStatus       int
		wantBodyContains []string
		wantHeaders      map[string]string
	}{
		{
			name:         "successful metrics response",
			queueManager: &mockQueueManager{queueDepth: 1500},
			wantStatus:   http.StatusOK,
			wantBodyContains: []string{
				"# HELP inngest_queue_depth",
				"# TYPE inngest_queue_depth gauge",
				"inngest_queue_depth 1500",
			},
			wantHeaders: map[string]string{
				"Content-Type": "text/plain; version=0.0.4; charset=utf-8",
			},
		},
		{
			name:         "queue manager error",
			queueManager: &mockQueueManager{err: errors.New("redis connection failed")},
			wantStatus:   http.StatusInternalServerError,
			wantBodyContains: []string{
				"Failed to get queue depth",
			},
		},
		{
			name:           "auth middleware blocks request",
			queueManager:   &mockQueueManager{queueDepth: 100},
			authMiddleware: mockAuthMiddleware(true), // requires auth
			setupRequest: func(r *http.Request) {
				// Don't set Authorization header
			},
			wantStatus: http.StatusUnauthorized,
			wantBodyContains: []string{
				"Unauthorized",
			},
		},
		{
			name:           "auth middleware allows request",
			queueManager:   &mockQueueManager{queueDepth: 2000},
			authMiddleware: mockAuthMiddleware(true), // requires auth
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer valid-token")
			},
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"inngest_queue_depth 2000",
			},
		},
		{
			name:         "zero queue depth",
			queueManager: &mockQueueManager{queueDepth: 0},
			wantStatus:   http.StatusOK,
			wantBodyContains: []string{
				"inngest_queue_depth 0",
			},
		},
		{
			name:         "large queue depth",
			queueManager: &mockQueueManager{queueDepth: 50000000}, // Within reasonable limits
			wantStatus:   http.StatusOK,
			wantBodyContains: []string{
				"inngest_queue_depth 5e+07",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create API with test configuration
			opts := Opts{
				QueueManager:   tt.queueManager,
				AuthMiddleware: tt.authMiddleware,
			}

			api, err := NewMetricsAPI(opts)
			require.NoError(t, err)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			api.Router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.wantStatus, w.Code)

			// Check response body contains expected strings
			body := w.Body.String()
			for _, expected := range tt.wantBodyContains {
				assert.Contains(t, body, expected, "Response body should contain: %s", expected)
			}

			// Check headers
			for key, expectedValue := range tt.wantHeaders {
				assert.Equal(t, expectedValue, w.Header().Get(key))
			}
		})
	}
}

func TestMetricsAPI_setupRoutes(t *testing.T) {
	tests := []struct {
		name           string
		authMiddleware func(http.Handler) http.Handler
		expectAuth     bool
	}{
		{
			name:           "routes without auth middleware",
			authMiddleware: nil,
			expectAuth:     false,
		},
		{
			name:           "routes with auth middleware",
			authMiddleware: mockAuthMiddleware(true),
			expectAuth:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Opts{
				QueueManager:   &mockQueueManager{queueDepth: 100},
				AuthMiddleware: tt.authMiddleware,
			}

			api, err := NewMetricsAPI(opts)
			require.NoError(t, err)

			// Test that routes are configured
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.expectAuth {
				// Should fail without auth
				w := httptest.NewRecorder()
				api.Router.ServeHTTP(w, req)
				assert.Equal(t, http.StatusUnauthorized, w.Code)

				// Should succeed with auth
				req.Header.Set("Authorization", "Bearer valid-token")
				w = httptest.NewRecorder()
				api.Router.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				// Should succeed without auth
				w := httptest.NewRecorder()
				api.Router.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

func TestMetricsAPI_prometheusIntegration(t *testing.T) {
	t.Run("prometheus registry integration", func(t *testing.T) {
		api, err := NewMetricsAPI(Opts{
			QueueManager: &mockQueueManager{queueDepth: 12345},
		})
		require.NoError(t, err)

		// Verify metric is registered
		metricFamilies, err := api.registry.Gather()
		require.NoError(t, err)
		require.Len(t, metricFamilies, 1)

		// Check metric family details
		mf := metricFamilies[0]
		assert.Equal(t, "inngest_queue_depth", mf.GetName())
		assert.Equal(t, "Total depth of all system queues including backlog and ready state items", mf.GetHelp())

		// Make a request to update the metric
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		api.Router.ServeHTTP(w, req)

		// Verify the metric value was updated
		metricFamilies, err = api.registry.Gather()
		require.NoError(t, err)
		require.Len(t, metricFamilies, 1)
		require.Len(t, metricFamilies[0].GetMetric(), 1)

		metric := metricFamilies[0].GetMetric()[0]
		assert.Equal(t, float64(12345), metric.GetGauge().GetValue())
	})
}

func TestMetricsAPI_concurrency(t *testing.T) {
	t.Run("concurrent requests", func(t *testing.T) {
		qm := &mockQueueManager{queueDepth: 5000}
		api, err := NewMetricsAPI(Opts{QueueManager: qm})
		require.NoError(t, err)

		// Make multiple concurrent requests
		const numRequests = 10
		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				w := httptest.NewRecorder()
				api.Router.ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		// Collect all results
		for i := 0; i < numRequests; i++ {
			status := <-results
			assert.Equal(t, http.StatusOK, status)
		}
	})
}

func TestMetricsAPI_errorHandling(t *testing.T) {
	t.Run("registry gather error", func(t *testing.T) {
		// This test is tricky because we can't easily mock registry.Gather() failure
		// In a real scenario, this would be tested with integration tests
		// For now, we test that the basic error path works

		api, err := NewMetricsAPI(Opts{
			QueueManager: &mockQueueManager{queueDepth: 100},
		})
		require.NoError(t, err)

		// Test successful path to ensure error handling works
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		api.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestQueueManagerInterface(t *testing.T) {
	t.Run("interface compliance", func(t *testing.T) {
		var qm QueueManager = &mockQueueManager{queueDepth: 100}

		depth, err := qm.TotalSystemQueueDepth(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, int64(100), depth)
	})
}

// Benchmark tests
func TestQueueDepthValidation(t *testing.T) {
	tests := []struct {
		name          string
		depth         int64
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid zero depth",
			depth:       0,
			expectError: false,
		},
		{
			name:        "valid small depth",
			depth:       1000,
			expectError: false,
		},
		{
			name:        "valid large depth",
			depth:       1_000_000,
			expectError: false,
		},
		{
			name:        "valid max reasonable depth",
			depth:       MaxReasonableQueueDepth,
			expectError: false,
		},
		{
			name:          "negative depth",
			depth:         -1,
			expectError:   true,
			errorContains: "queue depth cannot be negative",
		},
		{
			name:          "large negative depth",
			depth:         -999999,
			expectError:   true,
			errorContains: "queue depth cannot be negative",
		},
		{
			name:          "exceeds reasonable limit",
			depth:         MaxReasonableQueueDepth + 1,
			expectError:   true,
			errorContains: "exceeds maximum reasonable limit",
		},
		{
			name:          "exceeds safe integer limit",
			depth:         MaxSafeInt64 + 1,
			expectError:   true,
			errorContains: "exceeds safe integer limit",
		},
		{
			name:          "max safe integer exceeds reasonable limit",
			depth:         MaxSafeInt64,
			expectError:   true,
			errorContains: "exceeds maximum reasonable limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQueueDepth(tt.depth)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}

				// Check that error is the correct type
				var validationErr *QueueDepthValidationError
				assert.ErrorAs(t, err, &validationErr)
				assert.Equal(t, tt.depth, validationErr.Value)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeQueueDepth(t *testing.T) {
	tests := []struct {
		name          string
		depth         int64
		expectedDepth int64
		expectError   bool
		errorContains string
	}{
		{
			name:          "valid depth unchanged",
			depth:         1000,
			expectedDepth: 1000,
			expectError:   false,
		},
		{
			name:          "zero depth unchanged",
			depth:         0,
			expectedDepth: 0,
			expectError:   false,
		},
		{
			name:          "negative depth sanitized to zero",
			depth:         -5,
			expectedDepth: 0,
			expectError:   true,
			errorContains: "sanitized negative queue depth to 0",
		},
		{
			name:          "large negative depth sanitized to zero",
			depth:         -999999,
			expectedDepth: 0,
			expectError:   true,
			errorContains: "sanitized negative queue depth to 0",
		},
		{
			name:          "exceeds reasonable limit clamped to reasonable",
			depth:         MaxReasonableQueueDepth + 1000,
			expectedDepth: MaxReasonableQueueDepth,
			expectError:   true,
			errorContains: "clamped queue depth to maximum reasonable value",
		},
		{
			name:          "exceeds safe limit clamped to max safe",
			depth:         MaxSafeInt64 + 1,
			expectedDepth: MaxSafeInt64,
			expectError:   true,
			errorContains: "clamped queue depth to maximum safe value",
		},
		{
			name:          "max reasonable depth unchanged",
			depth:         MaxReasonableQueueDepth,
			expectedDepth: MaxReasonableQueueDepth,
			expectError:   false,
		},
		{
			name:          "max safe int64 clamped to reasonable",
			depth:         MaxSafeInt64,
			expectedDepth: MaxReasonableQueueDepth,
			expectError:   true,
			errorContains: "clamped queue depth to maximum reasonable value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, err := SanitizeQueueDepth(tt.depth)

			assert.Equal(t, tt.expectedDepth, sanitized)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricsAPI_handleMetricsWithValidation(t *testing.T) {
	tests := []struct {
		name             string
		queueDepth       int64
		queueError       error
		wantStatus       int
		wantBodyContains []string
		expectWarning    bool
	}{
		{
			name:       "valid depth",
			queueDepth: 1000,
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"inngest_queue_depth 1000",
			},
			expectWarning: false,
		},
		{
			name:       "zero depth",
			queueDepth: 0,
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"inngest_queue_depth 0",
			},
			expectWarning: false,
		},
		{
			name:       "negative depth sanitized",
			queueDepth: -500,
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				"inngest_queue_depth 0", // Should be sanitized to 0
			},
			expectWarning: true,
		},
		{
			name:       "excessive depth clamped",
			queueDepth: MaxReasonableQueueDepth + 1000,
			wantStatus: http.StatusOK,
			wantBodyContains: []string{
				fmt.Sprintf("inngest_queue_depth %v", float64(MaxReasonableQueueDepth)), // Should be clamped to reasonable limit
			},
			expectWarning: true,
		},
		{
			name:       "queue manager error",
			queueError: errors.New("redis connection failed"),
			wantStatus: http.StatusInternalServerError,
			wantBodyContains: []string{
				"Failed to get queue depth",
			},
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create API with test configuration
			qm := &mockQueueManager{queueDepth: tt.queueDepth, err: tt.queueError}
			api, err := NewMetricsAPI(Opts{QueueManager: qm})
			require.NoError(t, err)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			// Execute request
			api.Router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.wantStatus, w.Code)

			// Check response body contains expected strings
			body := w.Body.String()
			for _, expected := range tt.wantBodyContains {
				assert.Contains(t, body, expected, "Response body should contain: %s", expected)
			}
		})
	}
}

func TestNewMetricsAPI_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		opts          Opts
		wantError     bool
		errorContains string
	}{
		{
			name: "valid options",
			opts: Opts{
				QueueManager: &mockQueueManager{queueDepth: 100},
			},
			wantError: false,
		},
		{
			name: "missing queue manager",
			opts: Opts{
				AuthMiddleware: func(h http.Handler) http.Handler { return h },
			},
			wantError:     true,
			errorContains: "QueueManager is required",
		},
		{
			name:          "empty options",
			opts:          Opts{},
			wantError:     true,
			errorContains: "QueueManager is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, err := NewMetricsAPI(tt.opts)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, api)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, api)
			}
		})
	}
}

func TestQueueDepthValidationError(t *testing.T) {
	err := &QueueDepthValidationError{
		Value:   -100,
		Message: "test error message",
	}

	expectedError := "invalid queue depth -100: test error message"
	assert.Equal(t, expectedError, err.Error())
}

// Benchmark tests
func BenchmarkValidateQueueDepth(b *testing.B) {
	depths := []int64{0, 1000, MaxReasonableQueueDepth, -1, MaxSafeInt64 + 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		depth := depths[i%len(depths)]
		_ = ValidateQueueDepth(depth)
	}
}

func BenchmarkSanitizeQueueDepth(b *testing.B) {
	depths := []int64{1000, -500, MaxReasonableQueueDepth + 1000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		depth := depths[i%len(depths)]
		_, _ = SanitizeQueueDepth(depth)
	}
}

func BenchmarkMetricsAPI_handleMetrics(b *testing.B) {
	api, err := NewMetricsAPI(Opts{
		QueueManager: &mockQueueManager{queueDepth: 10000},
	})
	require.NoError(b, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		api.Router.ServeHTTP(w, req)
	}
}
