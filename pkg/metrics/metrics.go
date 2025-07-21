package metrics

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

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
	opts         Opts
	Router       chi.Router
	queueGauge   prometheus.Gauge
	registry     *prometheus.Registry
}

// NewMetricsAPI creates a new metrics API instance with Prometheus integration
func NewMetricsAPI(opts Opts) (*MetricsAPI, error) {
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
	
	api.queueGauge.Set(float64(depth))
	
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