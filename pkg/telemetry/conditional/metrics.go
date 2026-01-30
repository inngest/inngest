package conditional

import (
	"context"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

func init() {
	// Register the conditional metrics check with the metrics package.
	// This enables metrics.Record*Metric() to automatically check conditional scopes.
	metrics.RegisterConditionalCheck(conditionalMetricsCheck)
}

// conditionalMetricsCheck is called by metrics.Record*Metric() functions to determine
// if metrics should be recorded for the current context. Returns true if metrics should
// proceed, false if they should be skipped.
func conditionalMetricsCheck(ctx context.Context) bool {
	scope, ok := ScopeFromContext(ctx)
	if !ok {
		// No scope set, metrics proceed normally
		return true
	}
	// Scope is set, check if metrics are enabled for this scope
	return IsMetricsEnabled(ctx, scope)
}

// ConditionalCounterOpt extends metrics.CounterOpt with a Scope for conditional recording.
type ConditionalCounterOpt struct {
	metrics.CounterOpt
	Scope string
}

// ConditionalGaugeOpt extends metrics.GaugeOpt with a Scope for conditional recording.
type ConditionalGaugeOpt struct {
	metrics.GaugeOpt
	Scope string
}

// ConditionalHistogramOpt extends metrics.HistogramOpt with a Scope for conditional recording.
type ConditionalHistogramOpt struct {
	metrics.HistogramOpt
	Scope string
}

// RecordConditionalCounter records a counter metric if metrics are enabled for the given scope.
func RecordConditionalCounter(ctx context.Context, incr int64, opts ConditionalCounterOpt) {
	if !IsMetricsEnabled(ctx, opts.Scope) {
		return
	}
	metrics.RecordCounterMetric(ctx, incr, opts.CounterOpt)
}

// RecordConditionalUpDownCounter records an up-down counter metric if metrics are enabled for the given scope.
func RecordConditionalUpDownCounter(ctx context.Context, val int64, opts ConditionalCounterOpt) {
	if !IsMetricsEnabled(ctx, opts.Scope) {
		return
	}
	metrics.RecordUpDownCounterMetric(ctx, val, opts.CounterOpt)
}

// RecordConditionalGauge records a gauge metric if metrics are enabled for the given scope.
func RecordConditionalGauge(ctx context.Context, val int64, opts ConditionalGaugeOpt) {
	if !IsMetricsEnabled(ctx, opts.Scope) {
		return
	}
	metrics.RecordGaugeMetric(ctx, val, opts.GaugeOpt)
}

// RecordConditionalHistogram records a histogram metric if metrics are enabled for the given scope.
func RecordConditionalHistogram(ctx context.Context, value int64, opts ConditionalHistogramOpt) {
	if !IsMetricsEnabled(ctx, opts.Scope) {
		return
	}
	metrics.RecordIntHistogramMetric(ctx, value, opts.HistogramOpt)
}

// RegisterConditionalAsyncGauge registers an async gauge if metrics are enabled for the given scope.
// Note: This checks enablement at registration time, not at observation time.
// For dynamic enablement, use RecordConditionalGauge instead.
func RegisterConditionalAsyncGauge(ctx context.Context, opts ConditionalGaugeOpt) {
	if !IsMetricsEnabled(ctx, opts.Scope) {
		return
	}
	metrics.RegisterAsyncGauge(ctx, opts.GaugeOpt)
}

// ScopedMetrics provides a scoped wrapper for recording conditional metrics.
type ScopedMetrics struct {
	scope string
}

// NewScopedMetrics creates a new ScopedMetrics with the given scope.
func NewScopedMetrics(scope string) *ScopedMetrics {
	return &ScopedMetrics{scope: scope}
}

// Scope returns the scope of this ScopedMetrics.
func (s *ScopedMetrics) Scope() string {
	return s.scope
}

// RecordCounter records a counter metric if metrics are enabled for this scope.
func (s *ScopedMetrics) RecordCounter(ctx context.Context, incr int64, opts metrics.CounterOpt) {
	RecordConditionalCounter(ctx, incr, ConditionalCounterOpt{
		CounterOpt: opts,
		Scope:      s.scope,
	})
}

// RecordUpDownCounter records an up-down counter metric if metrics are enabled for this scope.
func (s *ScopedMetrics) RecordUpDownCounter(ctx context.Context, val int64, opts metrics.CounterOpt) {
	RecordConditionalUpDownCounter(ctx, val, ConditionalCounterOpt{
		CounterOpt: opts,
		Scope:      s.scope,
	})
}

// RecordGauge records a gauge metric if metrics are enabled for this scope.
func (s *ScopedMetrics) RecordGauge(ctx context.Context, val int64, opts metrics.GaugeOpt) {
	RecordConditionalGauge(ctx, val, ConditionalGaugeOpt{
		GaugeOpt: opts,
		Scope:    s.scope,
	})
}

// RecordHistogram records a histogram metric if metrics are enabled for this scope.
func (s *ScopedMetrics) RecordHistogram(ctx context.Context, value int64, opts metrics.HistogramOpt) {
	RecordConditionalHistogram(ctx, value, ConditionalHistogramOpt{
		HistogramOpt: opts,
		Scope:        s.scope,
	})
}
