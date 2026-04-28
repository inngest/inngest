package metrics

import (
	"context"
	"testing"

	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// counterValue reads the current value of a counter from the package-level
// registry by metric name and label values.
func counterValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	for _, f := range families {
		if f.GetName() != name {
			continue
		}
		for _, m := range f.GetMetric() {
			match := true
			for _, lp := range m.GetLabel() {
				if v, ok := labels[lp.GetName()]; !ok || v != lp.GetValue() {
					match = false
					break
				}
			}
			if match && len(m.GetLabel()) == len(labels) {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func mdWithSlug(slug string) statev2.Metadata {
	md := statev2.Metadata{}
	statev2.InitConfig(&md.Config)
	if slug != "" {
		md.Config.SetFunctionSlug(slug)
	}
	return md
}

func TestFnLabel(t *testing.T) {
	t.Run("returns slug when present", func(t *testing.T) {
		md := mdWithSlug("app-my-function")
		assert.Equal(t, "app-my-function", fnLabel(md))
	})

	t.Run("returns unknown when slug is empty", func(t *testing.T) {
		md := mdWithSlug("")
		assert.Equal(t, "unknown", fnLabel(md))
	})
}

func TestPrometheusLifecycleListener_FunctionRun(t *testing.T) {
	l := NewPrometheusLifecycleListener()
	ctx := context.Background()
	slug := "test-fn-run"
	md := mdWithSlug(slug)
	labels := map[string]string{"fn": slug}

	// Capture baselines (counters accumulate across tests in the same process).
	baseScheduled := counterValue(t, "inngest_function_run_scheduled_total", labels)
	baseStarted := counterValue(t, "inngest_function_run_started_total", labels)
	baseCompleted := counterValue(t, "inngest_function_run_ended_total", map[string]string{"fn": slug, "status": "Completed"})
	baseFailed := counterValue(t, "inngest_function_run_ended_total", map[string]string{"fn": slug, "status": "Failed"})

	l.OnFunctionScheduled(ctx, md, queue.Item{}, nil)
	assert.Equal(t, baseScheduled+1, counterValue(t, "inngest_function_run_scheduled_total", labels))

	l.OnFunctionStarted(ctx, md, queue.Item{}, nil)
	assert.Equal(t, baseStarted+1, counterValue(t, "inngest_function_run_started_total", labels))

	// Completed
	l.OnFunctionFinished(ctx, md, queue.Item{}, nil, statev1.DriverResponse{})
	assert.Equal(t, baseCompleted+1, counterValue(t, "inngest_function_run_ended_total", map[string]string{"fn": slug, "status": "Completed"}))

	// Failed
	errStr := "something went wrong"
	l.OnFunctionFinished(ctx, md, queue.Item{}, nil, statev1.DriverResponse{Err: &errStr})
	assert.Equal(t, baseFailed+1, counterValue(t, "inngest_function_run_ended_total", map[string]string{"fn": slug, "status": "Failed"}))
}

func TestPrometheusLifecycleListener_FunctionCancelled(t *testing.T) {
	l := NewPrometheusLifecycleListener()
	ctx := context.Background()
	slug := "test-fn-cancel"
	md := mdWithSlug(slug)
	labels := map[string]string{"fn": slug, "status": "Cancelled"}

	base := counterValue(t, "inngest_function_run_ended_total", labels)
	l.OnFunctionCancelled(ctx, md, execution.CancelRequest{}, nil)
	assert.Equal(t, base+1, counterValue(t, "inngest_function_run_ended_total", labels))
}

func TestPrometheusLifecycleListener_StepLifecycle(t *testing.T) {
	l := NewPrometheusLifecycleListener()
	ctx := context.Background()
	slug := "test-step-lc"
	md := mdWithSlug(slug)
	fnLabels := map[string]string{"fn": slug}

	baseScheduled := counterValue(t, "inngest_sdk_req_scheduled_total", fnLabels)
	baseStarted := counterValue(t, "inngest_sdk_req_started_total", fnLabels)
	baseSuccess := counterValue(t, "inngest_sdk_req_ended_total", map[string]string{"fn": slug, "status": "success"})
	baseErrored := counterValue(t, "inngest_sdk_req_ended_total", map[string]string{"fn": slug, "status": "errored"})
	baseFailed := counterValue(t, "inngest_sdk_req_ended_total", map[string]string{"fn": slug, "status": "failed"})

	l.OnStepScheduled(ctx, md, queue.Item{}, nil)
	assert.Equal(t, baseScheduled+1, counterValue(t, "inngest_sdk_req_scheduled_total", fnLabels))

	l.OnStepStarted(ctx, md, queue.Item{}, inngest.Edge{}, "http://localhost")
	assert.Equal(t, baseStarted+1, counterValue(t, "inngest_sdk_req_started_total", fnLabels))

	// success: no error
	l.OnStepFinished(ctx, md, queue.Item{}, inngest.Edge{}, &statev1.DriverResponse{}, nil)
	assert.Equal(t, baseSuccess+1, counterValue(t, "inngest_sdk_req_ended_total", map[string]string{"fn": slug, "status": "success"}))

	// errored: runErr set
	l.OnStepFinished(ctx, md, queue.Item{}, inngest.Edge{}, nil, assert.AnError)
	assert.Equal(t, baseErrored+1, counterValue(t, "inngest_sdk_req_ended_total", map[string]string{"fn": slug, "status": "errored"}))

	// errored: retryable response error
	errStr := "temporary"
	l.OnStepFinished(ctx, md, queue.Item{}, inngest.Edge{}, &statev1.DriverResponse{
		Err:     &errStr,
		NoRetry: false,
	}, nil)
	assert.Equal(t, baseErrored+2, counterValue(t, "inngest_sdk_req_ended_total", map[string]string{"fn": slug, "status": "errored"}))

	// failed: non-retryable response error
	l.OnStepFinished(ctx, md, queue.Item{}, inngest.Edge{}, &statev1.DriverResponse{
		Err:     &errStr,
		NoRetry: true,
	}, nil)
	assert.Equal(t, baseFailed+1, counterValue(t, "inngest_sdk_req_ended_total", map[string]string{"fn": slug, "status": "failed"}))
}
