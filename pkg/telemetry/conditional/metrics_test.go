package conditional

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/stretchr/testify/require"
)

func TestRecordConditionalCounter(t *testing.T) {
	defer ClearFeatureFlag()

	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("records when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		// This should not panic - we can't easily verify the metric was recorded
		// without setting up a full OTel pipeline, but we verify it doesn't error
		RecordConditionalCounter(ctx, 1, ConditionalCounterOpt{
			CounterOpt: metrics.CounterOpt{
				PkgName:    "test",
				MetricName: "test_counter",
			},
			Scope: "test.Scope",
		})
	})

	t.Run("does not record when disabled", func(t *testing.T) {
		RegisterFeatureFlag(NeverEnabled)

		// This should return early without calling metrics
		RecordConditionalCounter(ctx, 1, ConditionalCounterOpt{
			CounterOpt: metrics.CounterOpt{
				PkgName:    "test",
				MetricName: "test_counter",
			},
			Scope: "test.Scope",
		})
	})
}

func TestRecordConditionalUpDownCounter(t *testing.T) {
	defer ClearFeatureFlag()

	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("records when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		RecordConditionalUpDownCounter(ctx, 5, ConditionalCounterOpt{
			CounterOpt: metrics.CounterOpt{
				PkgName:    "test",
				MetricName: "test_updown_counter",
			},
			Scope: "test.Scope",
		})
	})

	t.Run("does not record when disabled", func(t *testing.T) {
		RegisterFeatureFlag(NeverEnabled)

		RecordConditionalUpDownCounter(ctx, 5, ConditionalCounterOpt{
			CounterOpt: metrics.CounterOpt{
				PkgName:    "test",
				MetricName: "test_updown_counter",
			},
			Scope: "test.Scope",
		})
	})
}

func TestRecordConditionalGauge(t *testing.T) {
	defer ClearFeatureFlag()

	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("records when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		RecordConditionalGauge(ctx, 42, ConditionalGaugeOpt{
			GaugeOpt: metrics.GaugeOpt{
				PkgName:    "test",
				MetricName: "test_gauge",
			},
			Scope: "test.Scope",
		})
	})

	t.Run("does not record when disabled", func(t *testing.T) {
		RegisterFeatureFlag(NeverEnabled)

		RecordConditionalGauge(ctx, 42, ConditionalGaugeOpt{
			GaugeOpt: metrics.GaugeOpt{
				PkgName:    "test",
				MetricName: "test_gauge",
			},
			Scope: "test.Scope",
		})
	})
}

func TestRecordConditionalHistogram(t *testing.T) {
	defer ClearFeatureFlag()

	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("records when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		RecordConditionalHistogram(ctx, 100, ConditionalHistogramOpt{
			HistogramOpt: metrics.HistogramOpt{
				PkgName:    "test",
				MetricName: "test_histogram",
				Boundaries: []float64{10, 50, 100, 500, 1000},
			},
			Scope: "test.Scope",
		})
	})

	t.Run("does not record when disabled", func(t *testing.T) {
		RegisterFeatureFlag(NeverEnabled)

		RecordConditionalHistogram(ctx, 100, ConditionalHistogramOpt{
			HistogramOpt: metrics.HistogramOpt{
				PkgName:    "test",
				MetricName: "test_histogram",
			},
			Scope: "test.Scope",
		})
	})
}

func TestScopedMetrics(t *testing.T) {
	defer ClearFeatureFlag()

	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))
	sm := NewScopedMetrics("test.Scope")

	require.Equal(t, "test.Scope", sm.Scope())

	t.Run("RecordCounter when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		sm.RecordCounter(ctx, 1, metrics.CounterOpt{
			PkgName:    "test",
			MetricName: "scoped_counter",
		})
	})

	t.Run("RecordUpDownCounter when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		sm.RecordUpDownCounter(ctx, 5, metrics.CounterOpt{
			PkgName:    "test",
			MetricName: "scoped_updown_counter",
		})
	})

	t.Run("RecordGauge when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		sm.RecordGauge(ctx, 42, metrics.GaugeOpt{
			PkgName:    "test",
			MetricName: "scoped_gauge",
		})
	})

	t.Run("RecordHistogram when enabled", func(t *testing.T) {
		RegisterFeatureFlag(AlwaysEnabled)

		sm.RecordHistogram(ctx, 100, metrics.HistogramOpt{
			PkgName:    "test",
			MetricName: "scoped_histogram",
		})
	})

	t.Run("methods do nothing when disabled", func(t *testing.T) {
		RegisterFeatureFlag(NeverEnabled)

		// These should all return early without recording
		sm.RecordCounter(ctx, 1, metrics.CounterOpt{PkgName: "test", MetricName: "c"})
		sm.RecordUpDownCounter(ctx, 1, metrics.CounterOpt{PkgName: "test", MetricName: "u"})
		sm.RecordGauge(ctx, 1, metrics.GaugeOpt{PkgName: "test", MetricName: "g"})
		sm.RecordHistogram(ctx, 1, metrics.HistogramOpt{PkgName: "test", MetricName: "h"})
	})
}

func TestConditionalMetricsWithTags(t *testing.T) {
	defer ClearFeatureFlag()
	RegisterFeatureFlag(AlwaysEnabled)

	ctx := WithContext(context.Background(), WithAccountID(uuid.New()))

	t.Run("counter with tags", func(t *testing.T) {
		RecordConditionalCounter(ctx, 1, ConditionalCounterOpt{
			CounterOpt: metrics.CounterOpt{
				PkgName:     "test",
				MetricName:  "tagged_counter",
				Description: "A test counter with tags",
				Tags: map[string]any{
					"status": "success",
					"type":   "test",
				},
			},
			Scope: "test.Scope",
		})
	})

	t.Run("histogram with boundaries and tags", func(t *testing.T) {
		RecordConditionalHistogram(ctx, 250, ConditionalHistogramOpt{
			HistogramOpt: metrics.HistogramOpt{
				PkgName:     "test",
				MetricName:  "latency_histogram",
				Description: "Latency in milliseconds",
				Unit:        "ms",
				Boundaries:  []float64{10, 50, 100, 250, 500, 1000},
				Tags: map[string]any{
					"endpoint": "/api/test",
				},
			},
			Scope: "test.Scope",
		})
	})
}
