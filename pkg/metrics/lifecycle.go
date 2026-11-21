package metrics

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	functionRunScheduled = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "inngest_function_run_scheduled_total",
		Help: "The total number of function runs scheduled",
	}, []string{"fn"})

	functionRunStarted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "inngest_function_run_started_total",
		Help: "The total number of function runs started",
	}, []string{"fn"})

	functionRunEnded = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "inngest_function_run_ended_total",
		Help: "The total number of function runs ended",
	}, []string{"fn", "status"})

	sdkReqScheduled = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "inngest_sdk_req_scheduled_total",
		Help: "The total number of SDK invocation/step execution scheduled",
	}, []string{"fn"})

	sdkReqStarted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "inngest_sdk_req_started_total",
		Help: "The total number of SDK invocation/step execution started",
	}, []string{"fn"})

	sdkReqEnded = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "inngest_sdk_req_ended_total",
		Help: "The total number of SDK invocation/step execution ended",
	}, []string{"fn", "status"})

	functionRunQueueDelay = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "inngest_function_run_queue_delay_seconds",
		Help:    "Time function runs waited in the queue between being due and starting",
		Buckets: queueDelayBuckets,
	}, []string{"fn"})

	sdkReqQueueDelay = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "inngest_sdk_req_queue_delay_seconds",
		Help:    "Time SDK invocations/step executions waited in the queue between being due and starting",
		Buckets: queueDelayBuckets,
	}, []string{"fn"})
)

// queueDelayBuckets covers delays from a few milliseconds, when the queue keeps
// up, to an hour, when runs are held back by concurrency or throttle limits.
var queueDelayBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, // < 1s
	1, 2.5, 5, 10, 30, // < 1m
	60, 300, 600, 1800, 3600, // <= 1h
}

func init() {
	registry.MustRegister(
		functionRunScheduled,
		functionRunStarted,
		functionRunEnded,
		sdkReqScheduled,
		sdkReqStarted,
		sdkReqEnded,
		functionRunQueueDelay,
		sdkReqQueueDelay,
	)
}

// PrometheusLifecycleListener implements execution.LifecycleListener and
// records per-function Prometheus metrics.
type PrometheusLifecycleListener struct {
	execution.NoopLifecyceListener
}

func NewPrometheusLifecycleListener() *PrometheusLifecycleListener {
	return &PrometheusLifecycleListener{}
}

// NewLifecycleListeners returns the Prometheus lifecycle listener if the
// experimental metrics feature is enabled, or an empty slice otherwise.
func NewLifecycleListeners() []execution.LifecycleListener {
	if !ExperimentalPromMetricsEnabled() {
		return nil
	}
	return []execution.LifecycleListener{NewPrometheusLifecycleListener()}
}

func fnLabel(md statev2.Metadata) string {
	if slug := md.Config.FunctionSlug(); slug != "" {
		return slug
	}
	return "unknown"
}

// observeQueueDelay records how long the queue item being processed waited
// between being due and starting.  Work that did not come from the queue
// carries no start latency and is not recorded.
func observeQueueDelay(ctx context.Context, h *prometheus.HistogramVec, md statev2.Metadata) {
	delay, ok := queue.GetItemStartLatency(ctx)
	if !ok {
		return
	}
	h.WithLabelValues(fnLabel(md)).Observe(delay.Seconds())
}

func (l *PrometheusLifecycleListener) OnFunctionScheduled(
	_ context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ []event.TrackedEvent,
) {
	functionRunScheduled.WithLabelValues(fnLabel(md)).Inc()
}

func (l *PrometheusLifecycleListener) OnFunctionStarted(
	ctx context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ []json.RawMessage,
) {
	functionRunStarted.WithLabelValues(fnLabel(md)).Inc()
	observeQueueDelay(ctx, functionRunQueueDelay, md)
}

func (l *PrometheusLifecycleListener) OnFunctionFinished(
	_ context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ []json.RawMessage,
	resp statev1.DriverResponse,
) {
	status := "completed"
	if resp.Err != nil {
		status = "failed"
	}
	functionRunEnded.WithLabelValues(fnLabel(md), status).Inc()
}

func (l *PrometheusLifecycleListener) OnFunctionCancelled(
	_ context.Context,
	md statev2.Metadata,
	_ execution.CancelRequest,
	_ []json.RawMessage,
) {
	functionRunEnded.WithLabelValues(fnLabel(md), "cancelled").Inc()
}

func (l *PrometheusLifecycleListener) OnStepScheduled(
	_ context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ *string,
) {
	sdkReqScheduled.WithLabelValues(fnLabel(md)).Inc()
}

func (l *PrometheusLifecycleListener) OnStepStarted(
	ctx context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ inngest.Edge,
	_ string,
) {
	sdkReqStarted.WithLabelValues(fnLabel(md)).Inc()
	observeQueueDelay(ctx, sdkReqQueueDelay, md)
}

func (l *PrometheusLifecycleListener) OnStepFinished(
	_ context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ inngest.Edge,
	resp *statev1.DriverResponse,
	runErr error,
) {
	status := "success"
	if runErr != nil {
		status = "errored"
	} else if resp != nil && resp.Err != nil {
		if resp.Retryable() {
			status = "errored"
		} else {
			status = "failed"
		}
	}
	sdkReqEnded.WithLabelValues(fnLabel(md), status).Inc()
}

// Ensure interface compliance at compile time.
var _ execution.LifecycleListener = (*PrometheusLifecycleListener)(nil)
