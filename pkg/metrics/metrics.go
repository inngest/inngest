package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

const (
	// MaxReasonableQueueDepth defines the maximum reasonable queue depth value.
	// This helps detect potential data corruption or calculation errors.
	// Set to 1 billion items, which should handle very large production systems.
	MaxReasonableQueueDepth int64 = 1_000_000_000

	// MaxSafeInt64 is the maximum safe integer value for int64 to avoid overflow
	// when converting to float64 for Prometheus metrics.
	// This is 2^53 - 1, the largest integer that can be exactly represented as float64.
	MaxSafeInt64 int64 = 1<<53 - 1 // 9,007,199,254,740,991
)

// QueueDepthValidationError represents a queue depth validation error
type QueueDepthValidationError struct {
	Value   int64
	Message string
}

func (e *QueueDepthValidationError) Error() string {
	return fmt.Sprintf("invalid queue depth %d: %s", e.Value, e.Message)
}

// ValidateQueueDepth validates a queue depth value for reasonableness and safety
func ValidateQueueDepth(depth int64) error {
	if depth < 0 {
		return &QueueDepthValidationError{
			Value:   depth,
			Message: "queue depth cannot be negative",
		}
	}

	// Check safe integer limit first (more restrictive for very large numbers)
	if depth > MaxSafeInt64 {
		return &QueueDepthValidationError{
			Value:   depth,
			Message: fmt.Sprintf("queue depth %d exceeds safe integer limit for metrics (max: %d)", depth, MaxSafeInt64),
		}
	}

	// Check reasonable limit (for operational sanity)
	if depth > MaxReasonableQueueDepth {
		return &QueueDepthValidationError{
			Value:   depth,
			Message: fmt.Sprintf("queue depth %d exceeds maximum reasonable limit of %d", depth, MaxReasonableQueueDepth),
		}
	}

	return nil
}

// SanitizeQueueDepth validates and sanitizes a queue depth value, returning a safe value
func SanitizeQueueDepth(depth int64) (int64, error) {
	if err := ValidateQueueDepth(depth); err != nil {
		// For negative values, return 0 as a safe fallback
		if depth < 0 {
			return 0, fmt.Errorf("sanitized negative queue depth to 0: %w", err)
		}
		// For values that exceed safe integer limit, clamp to max safe
		if depth > MaxSafeInt64 {
			return MaxSafeInt64, fmt.Errorf("clamped queue depth to maximum safe value: %w", err)
		}
		// For values that exceed reasonable limit but are within safe integer range,
		// clamp to reasonable limit
		if depth > MaxReasonableQueueDepth {
			return MaxReasonableQueueDepth, fmt.Errorf("clamped queue depth to maximum reasonable value: %w", err)
		}
	}
	return depth, nil
}

// QueueManager defines the interface for accessing queue metrics
type QueueManager interface {
	TotalSystemQueueDepth(ctx context.Context) (int64, error)
}

// Opts holds the configuration options for the metrics API
type Opts struct {
	AuthMiddleware func(http.Handler) http.Handler
	QueueManager   QueueManager
}

// MetricsAPI provides Prometheus-compatible metrics endpoints
type MetricsAPI struct {
	opts       Opts
	Router     chi.Router
	queueGauge prometheus.Gauge
	registry   *prometheus.Registry
}

// NewMetricsAPI creates a new metrics API instance with Prometheus integration
func NewMetricsAPI(opts Opts) (*MetricsAPI, error) {
	// Validate required options
	if opts.QueueManager == nil {
		return nil, fmt.Errorf("QueueManager is required")
	}

	registry := prometheus.NewRegistry()

	queueGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "inngest_queue_depth",
		Help: "Total depth of all system queues including backlog and ready state items",
	})

	registry.MustRegister(queueGauge)

	api := &MetricsAPI{
		opts:       opts,
		Router:     chi.NewRouter(),
		queueGauge: queueGauge,
		registry:   registry,
	}

	api.setupRoutes()
	return api, nil
}

// setupRoutes configures the HTTP routes for the metrics API
func (api *MetricsAPI) setupRoutes() {
	handler := http.HandlerFunc(api.handleMetrics)

	if api.opts.AuthMiddleware != nil {
		handler = api.opts.AuthMiddleware(handler).ServeHTTP
	}

	api.Router.Get("/", handler)
}

// handleMetrics serves Prometheus-formatted metrics
func (api *MetricsAPI) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Update queue depth metric
	depth, err := api.opts.QueueManager.TotalSystemQueueDepth(r.Context())
	if err != nil {
		http.Error(w, "Failed to get queue depth", http.StatusInternalServerError)
		return
	}

	// Validate and sanitize the queue depth value
	sanitizedDepth, validationErr := SanitizeQueueDepth(depth)
	if validationErr != nil {
		// Log the validation warning but continue with sanitized value
		// In a production system, you might want to use a proper logger here
		fmt.Printf("Warning: Queue depth validation issue: %v\n", validationErr)
	}

	api.queueGauge.Set(float64(sanitizedDepth))

	// Gather metrics from registry
	metricFamilies, err := api.registry.Gather()
	if err != nil {
		http.Error(w, "Failed to gather metrics", http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", string(expfmt.FmtText))

	// Encode metrics in Prometheus text format
	encoder := expfmt.NewEncoder(w, expfmt.FmtText)
	for _, mf := range metricFamilies {
		if err := encoder.Encode(mf); err != nil {
			http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
			return
		}
	}
}
