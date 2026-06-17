package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestHistogramRunStateResidenceDuration(t *testing.T) {
	ctx, reader := testMeter(t)

	HistogramRunStateResidenceDuration(ctx, -time.Second, HistogramOpt{
		PkgName: "test",
		Tags: map[string]any{
			"delete_status": "failed",
			"status":        "failed",
		},
	})

	m := collectMetric(t, ctx, reader, "inngest_run_state_residence_duration")
	require.Equal(t, "ms", m.Unit)
	require.Equal(t, "Distribution of time between run creation and finalization state delete outcome", m.Description)

	h := requireHistogram(t, m)
	require.Len(t, h.DataPoints, 1)

	dp := h.DataPoints[0]
	require.Equal(t, uint64(1), dp.Count)
	require.Equal(t, int64(0), dp.Sum)
	require.Equal(t, runStateResidenceDurationBoundaries, dp.Bounds)
	require.Equal(t, []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, dp.BucketCounts)
	requireAttribute(t, dp.Attributes, "delete_status", "failed")
	requireAttribute(t, dp.Attributes, "status", "failed")
}

func TestHistogramRunStateStepCountCapsAt20(t *testing.T) {
	ctx, reader := testMeter(t)

	HistogramRunStateStepCount(ctx, 42, HistogramOpt{
		PkgName: "test",
		Tags: map[string]any{
			"status": "completed",
		},
	})

	m := collectMetric(t, ctx, reader, "inngest_run_state_step_count")
	require.Empty(t, m.Unit)
	require.Equal(t, "Distribution of completed step count per finalized run, capped at 20", m.Description)

	h := requireHistogram(t, m)
	require.Len(t, h.DataPoints, 1)

	dp := h.DataPoints[0]
	require.Equal(t, uint64(1), dp.Count)
	require.Equal(t, int64(20), dp.Sum)
	require.Equal(t, runStateStepCountBoundaries, dp.Bounds)
	require.Equal(t, []uint64{0, 0, 0, 0, 0, 1, 0}, dp.BucketCounts)
	requireAttribute(t, dp.Attributes, "status", "completed")
}

func testMeter(t *testing.T) (context.Context, *sdkmetric.ManualReader) {
	t.Helper()

	ctx := context.Background()
	originalRegistry := registry
	registry = newRegistry()
	t.Cleanup(func() {
		registry = originalRegistry
	})

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	originalMeterProvider := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(originalMeterProvider)
		require.NoError(t, provider.Shutdown(ctx))
	})

	return ctx, reader
}

func collectMetric(t *testing.T, ctx context.Context, reader *sdkmetric.ManualReader, name string) metricdata.Metrics {
	t.Helper()

	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(ctx, &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	require.Failf(t, "metric not found", "metric %q not found in collected metrics", name)
	return metricdata.Metrics{}
}

func requireHistogram(t *testing.T, m metricdata.Metrics) metricdata.Histogram[int64] {
	t.Helper()

	h, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok, "metric %q has data type %T", m.Name, m.Data)
	return h
}

func requireAttribute(t *testing.T, attrs attribute.Set, key, expected string) {
	t.Helper()

	value, ok := attrs.Value(attribute.Key(key))
	require.True(t, ok, "missing attribute %q", key)
	require.Equal(t, expected, value.AsString())
}
