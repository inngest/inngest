package metrics

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/inngest/inngest/pkg/inngest/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	prefix = "inngest"
)

var (
	registry = newRegistry()
)

//
// NOTE:
// Most of these maps probably can be done with generics.
// Too lazy to try to get it work that way right now
//

// ========================
//
//	Counters
//
// ========================

// NOTE: these can probably be simplified by generics
type counterMap struct {
	rw sync.RWMutex
	m  map[string]metric.Int64Counter
}

func (c *counterMap) Get(name string) (metric.Int64Counter, bool) {
	c.rw.RLock()
	defer c.rw.RUnlock()
	v, ok := c.m[name]
	return v, ok
}

func (c *counterMap) Add(name string, m metric.Int64Counter) {
	c.rw.Lock()
	defer c.rw.Unlock()
	c.m[name] = m
}

type upDownCounterMap struct {
	rw sync.RWMutex
	m  map[string]metric.Int64UpDownCounter
}

func (c *upDownCounterMap) Get(name string) (metric.Int64UpDownCounter, bool) {
	c.rw.RLock()
	defer c.rw.RUnlock()
	v, ok := c.m[name]
	return v, ok
}

func (c *upDownCounterMap) Add(name string, m metric.Int64UpDownCounter) {
	c.rw.Lock()
	defer c.rw.Unlock()
	c.m[name] = m
}

type CounterOpt struct {
	PkgName     string
	Description string
	Meter       metric.Meter
	MetricName  string
	Tags        map[string]any
	Unit        string
}

// RecordCounterMetric increments the counter by the provided value.
// The meter used can either be passed in or is the global meter
func RecordCounterMetric(ctx context.Context, incr int64, opts CounterOpt) {
	attrs := []attribute.KeyValue{}
	if opts.Tags != nil {
		attrs = append(attrs, parseAttributes(opts.Tags)...)
	}

	metricName := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	c, err := registry.getCounter(ctx, opts)
	if err != nil {
		log.From(ctx).Error().Err(err).Str("metric_name", metricName).Msg("error accessing counter metric")
		return
	}

	c.Add(ctx, incr, metric.WithAttributes(attrs...))
}

func RecordUpDownCounterMetric(ctx context.Context, val int64, opts CounterOpt) {
	attrs := []attribute.KeyValue{}
	if opts.Tags != nil {
		attrs = append(attrs, parseAttributes(opts.Tags)...)
	}

	metricName := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	c, err := registry.getUpDownCounter(ctx, opts)
	if err != nil {
		log.From(ctx).Error().Err(err).Str("metric_name", metricName).Msg("error accessing counter metric")
		return
	}

	c.Add(ctx, val, metric.WithAttributes(attrs...))
}

// ========================
//
//	Gauges
//
// ========================
type asyncGaugeMap struct {
	rw sync.RWMutex
	m  map[string]metric.Int64ObservableGauge
}

func (g *asyncGaugeMap) Get(name string) (metric.Int64ObservableGauge, bool) {
	g.rw.RLock()
	defer g.rw.RUnlock()
	v, ok := g.m[name]
	return v, ok
}

func (g *asyncGaugeMap) Add(name string, m metric.Int64ObservableGauge) {
	g.rw.Lock()
	defer g.rw.Unlock()
	g.m[name] = m
}

type gaugeMap struct {
	rw sync.RWMutex
	m  map[string]metric.Int64Gauge
}

func (g *gaugeMap) Get(name string) (metric.Int64Gauge, bool) {
	g.rw.RLock()
	defer g.rw.RUnlock()
	v, ok := g.m[name]
	return v, ok
}

func (g *gaugeMap) Add(name string, m metric.Int64Gauge) {
	g.rw.Lock()
	defer g.rw.Unlock()
	g.m[name] = m
}

type GaugeOpt struct {
	PkgName     string
	Description string
	MetricName  string
	Meter       metric.Meter
	Tags        map[string]any
	Unit        string
	Callback    GaugeCallback
}

type GaugeCallback func(ctx context.Context) (int64, error)

// RecordGaugeMetric records the gauge value via a callback.
// The callback needs to be passed in so it doesn't get captured as a closure when instrumenting the value
func RegisterAsyncGauge(ctx context.Context, opts GaugeOpt) {
	metricName := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	_, err := registry.setAsyncGauge(ctx, opts)
	if err != nil {
		log.From(ctx).Error().Err(err).Str("metric_name", metricName).Msg("error setting async gauge")
	}
}

func RecordGaugeMetric(ctx context.Context, val int64, opts GaugeOpt) {
	attrs := []attribute.KeyValue{}
	if opts.Tags != nil {
		attrs = append(attrs, parseAttributes(opts.Tags)...)
	}

	metricName := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	c, err := registry.getGauge(ctx, opts)
	if err != nil {
		log.From(ctx).Error().Err(err).Str("metric_name", metricName).Msg("error accessing counter metric")
		return
	}

	c.Record(ctx, val, metric.WithAttributes(attrs...))
}

// ========================
//
//	Histogram
//
// ========================

type histogramMap struct {
	rw sync.RWMutex
	m  map[string]metric.Int64Histogram
}

func (h *histogramMap) Get(name string) (metric.Int64Histogram, bool) {
	h.rw.RLock()
	defer h.rw.RUnlock()
	v, ok := h.m[name]
	return v, ok
}

func (h *histogramMap) Add(name string, m metric.Int64Histogram) {
	h.rw.Lock()
	defer h.rw.Unlock()
	h.m[name] = m
}

type HistogramOpt struct {
	PkgName     string
	Description string
	Meter       metric.Meter
	MetricName  string
	Tags        map[string]any
	Unit        string
	Boundaries  []float64
}

// RecordIntHistogramMetric records the observed value for distributions.
// Bucket can be provided
func RecordIntHistogramMetric(ctx context.Context, value int64, opts HistogramOpt) {
	metricName := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	h, err := registry.getHistogram(ctx, opts)
	if err != nil {
		log.From(ctx).Error().Err(err).Str("metric_name", metricName).Msg("error accessing histogram metric")
		return
	}

	attrs := []attribute.KeyValue{}
	if opts.Tags != nil {
		attrs = append(attrs, parseAttributes(opts.Tags)...)
	}
	h.Record(ctx, value, metric.WithAttributes(attrs...))
}

// ========================
//
//	Helpers
//
// ========================
type metricsRegistry struct {
	mu sync.RWMutex

	counters       *counterMap
	updownCounters *upDownCounterMap
	asyncGauges    *asyncGaugeMap
	gauges         *gaugeMap
	histograms     *histogramMap
}

func newRegistry() *metricsRegistry {
	return &metricsRegistry{
		counters:       &counterMap{m: map[string]metric.Int64Counter{}},
		updownCounters: &upDownCounterMap{m: map[string]metric.Int64UpDownCounter{}},
		asyncGauges:    &asyncGaugeMap{m: map[string]metric.Int64ObservableGauge{}},
		gauges:         &gaugeMap{m: map[string]metric.Int64Gauge{}},
		histograms:     &histogramMap{m: map[string]metric.Int64Histogram{}},
	}
}

func (r *metricsRegistry) getCounter(ctx context.Context, opts CounterOpt) (metric.Int64Counter, error) {
	name := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	if c, ok := r.counters.Get(name); ok {
		return c, nil
	}

	// use the global one by default
	meter := otel.Meter(opts.PkgName)
	if opts.Meter != nil {
		meter = opts.Meter
	}

	c, err := meter.Int64Counter(
		name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
	)
	if err == nil {
		r.counters.Add(name, c)
	}
	return c, err
}

func (r *metricsRegistry) getUpDownCounter(ctx context.Context, opts CounterOpt) (metric.Int64UpDownCounter, error) {
	name := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	if c, ok := r.updownCounters.Get(name); ok {
		return c, nil
	}

	// use the global one by default
	meter := otel.Meter(opts.PkgName)
	if opts.Meter != nil {
		meter = opts.Meter
	}

	c, err := meter.Int64UpDownCounter(
		name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
	)
	if err == nil {
		r.updownCounters.Add(name, c)
	}
	return c, err
}

func (r *metricsRegistry) setAsyncGauge(ctx context.Context, opts GaugeOpt) (metric.Int64ObservableGauge, error) {
	name := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	if g, ok := r.asyncGauges.Get(name); ok {
		return g, nil
	}

	// use the global one by default
	meter := otel.Meter(opts.PkgName)
	if opts.Meter != nil {
		meter = opts.Meter
	}

	g, err := meter.Int64ObservableGauge(
		name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			value, err := opts.Callback(ctx)
			if err != nil {
				return err
			}
			attrs := []attribute.KeyValue{}
			if opts.Tags != nil {
				attrs = append(attrs, parseAttributes(opts.Tags)...)
			}
			o.Observe(value, metric.WithAttributes(attrs...))
			return nil
		}),
	)
	if err == nil {
		r.asyncGauges.Add(name, g)
	}
	return g, err
}

func (r *metricsRegistry) getGauge(ctx context.Context, opts GaugeOpt) (metric.Int64Gauge, error) {
	name := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	if g, ok := r.gauges.Get(name); ok {
		return g, nil
	}

	meter := otel.Meter(opts.PkgName)
	if opts.Meter != nil {
		meter = opts.Meter
	}

	g, err := meter.Int64Gauge(
		name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
	)
	if err == nil {
		r.gauges.Add(name, g)
	}
	return g, err
}

func (r *metricsRegistry) getHistogram(ctx context.Context, opts HistogramOpt) (metric.Int64Histogram, error) {
	name := fmt.Sprintf("%s_%s", prefix, opts.MetricName)
	if h, ok := r.histograms.Get(name); ok {
		return h, nil
	}

	// use the global one by default
	meter := otel.Meter(opts.PkgName)
	if opts.Meter != nil {
		meter = opts.Meter
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := meter.Int64Histogram(
		name,
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
		metric.WithExplicitBucketBoundaries(opts.Boundaries...),
	)
	if err == nil {
		r.histograms.Add(name, m)
	}
	return m, err
}

// parseAttributes parses the attribute map into otel compatible attributes
func parseAttributes(attrs map[string]any) []attribute.KeyValue {
	result := make([]attribute.KeyValue, 0)

	for k, v := range attrs {
		attr := attribute.KeyValue{Key: attribute.Key(k)}

		isStringer, ok := v.(fmt.Stringer)
		if ok {
			attr.Value = attribute.StringValue(isStringer.String())
			result = append(result, attr)
			continue
		}

		t := reflect.TypeOf(v)
		switch t.Kind() {
		case reflect.String:
			attr.Value = attribute.StringValue(v.(string))
		case reflect.Int:
			attr.Value = attribute.IntValue(v.(int))
		case reflect.Int32:
			attr.Value = attribute.Int64Value(int64(v.(int32)))
		case reflect.Int64:
			attr.Value = attribute.Int64Value(v.(int64))
		case reflect.Uint:
			attr.Value = attribute.Int64Value(int64(v.(uint)))
		case reflect.Uint32:
			attr.Value = attribute.Int64Value(int64(v.(uint32)))
		case reflect.Uint64:
			attr.Value = attribute.Int64Value(int64(v.(uint64)))
		case reflect.Float32:
			attr.Value = attribute.Float64Value(float64(v.(float32)))
		case reflect.Float64:
			attr.Value = attribute.Float64Value(v.(float64))
		case reflect.Bool:
			attr.Value = attribute.BoolValue(v.(bool))
		default:
			log.From(context.Background()).
				Warn().
				Str("kind", t.Kind().String()).
				Interface("value", v).
				Msg("unsupported type of value used for metrics attribute")
			continue
		}

		result = append(result, attr)
	}

	return result
}
