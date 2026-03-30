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

)

func init() {
	prometheus.MustRegister(
		functionRunScheduled,
		functionRunStarted,
		functionRunEnded,
		sdkReqScheduled,
		sdkReqStarted,
		sdkReqEnded,
	)
}

// PrometheusLifecycleListener implements execution.LifecycleListener and
// records per-function Prometheus metrics matching the Inngest platform's
// metrics export format.
type PrometheusLifecycleListener struct {
	execution.NoopLifecyceListener
}

func NewPrometheusLifecycleListener() *PrometheusLifecycleListener {
	return &PrometheusLifecycleListener{}
}

func fnLabel(md statev2.Metadata) string {
	if slug := md.Config.FunctionSlug(); slug != "" {
		return slug
	}
	return md.ID.FunctionID.String()
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
	_ context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ []json.RawMessage,
) {
	functionRunStarted.WithLabelValues(fnLabel(md)).Inc()
}

func (l *PrometheusLifecycleListener) OnFunctionFinished(
	_ context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ []json.RawMessage,
	resp statev1.DriverResponse,
) {
	status := "Completed"
	if resp.Err != nil {
		status = "Failed"
	}
	functionRunEnded.WithLabelValues(fnLabel(md), status).Inc()
}

func (l *PrometheusLifecycleListener) OnFunctionCancelled(
	_ context.Context,
	md statev2.Metadata,
	_ execution.CancelRequest,
	_ []json.RawMessage,
) {
	functionRunEnded.WithLabelValues(fnLabel(md), "Cancelled").Inc()
}

func (l *PrometheusLifecycleListener) OnStepScheduled(
	_ context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ *string,
) {
	fn := fnLabel(md)
	sdkReqScheduled.WithLabelValues(fn).Inc()
}

func (l *PrometheusLifecycleListener) OnStepStarted(
	_ context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ inngest.Edge,
	_ string,
) {
	fn := fnLabel(md)
	sdkReqStarted.WithLabelValues(fn).Inc()
}

func (l *PrometheusLifecycleListener) OnStepFinished(
	_ context.Context,
	md statev2.Metadata,
	_ queue.Item,
	_ inngest.Edge,
	resp *statev1.DriverResponse,
	runErr error,
) {
	fn := fnLabel(md)

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
	sdkReqEnded.WithLabelValues(fn, status).Inc()
}

func (l *PrometheusLifecycleListener) Close(context.Context) error {
	return nil
}

// Ensure interface compliance at compile time.
var _ execution.LifecycleListener = (*PrometheusLifecycleListener)(nil)

// Unused methods are inherited from NoopLifecyceListener. We explicitly
// define Close to satisfy the interface even though it's a no-op, since
// the embedded struct's Close is sufficient but being explicit is clearer.

// The following are not overridden and fall through to the no-op:
// - OnFunctionSkipped
// - OnStepGatewayRequestFinished
// - OnWaitForEvent / OnWaitForEventResumed
// - OnInvokeFunction / OnInvokeFunctionResumed
// - OnWaitForSignal / OnWaitForSignalResumed
// - OnSleep
