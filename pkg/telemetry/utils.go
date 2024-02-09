package telemetry

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/inngest/inngest/pkg/inngest/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func env() string {
	val := os.Getenv("ENV")
	if val == "" {
		val = "development"
	}
	return val
}

type CounterOpt struct {
	Name        string
	Description string
	Meter       metric.Meter
	MetricName  string
	Attributes  map[string]any
	Unit        string
}

// RecordCounterMetric increments the counter by the provided value.
// The meter used can either be passed in or is the global meter
func RecordCounterMetric(ctx context.Context, incr int64, opts CounterOpt) {
	attrs := []attribute.KeyValue{}
	if opts.Attributes != nil {
		attrs = append(attrs, parseAttributes(opts.Attributes)...)
	}

	// use the global one by default
	meter := otel.Meter(opts.Name)
	if opts.Meter != nil {
		meter = opts.Meter
	}

	c, err := meter.
		Int64Counter(
			fmt.Sprintf("%s_%s", prefix, opts.MetricName),
			metric.WithDescription(opts.Description),
			metric.WithUnit(opts.Unit),
		)
	if err != nil {
		log.From(ctx).Error().Err(err).Msg(fmt.Sprintf("error for meter: %s", opts.MetricName))
		return
	}

	c.Add(ctx, incr, metric.WithAttributes(attrs...))
}

type GaugeOpt struct {
	Name        string
	Description string
	MetricName  string
	Meter       metric.Meter
	Attributes  map[string]any
	Unit        string
	Callback    GaugeCallback
}

type GaugeCallback func(ctx context.Context) (int64, error)

// RecordGaugeMetric records the gauge value via a callback.
// The callback needs to be passed in so it doesn't get captured as a closure when instrumenting the value
func RecordGaugeMetric(ctx context.Context, opts GaugeOpt) {
	// use the global one by default
	meter := otel.Meter(opts.Name)
	if opts.Meter != nil {
		meter = opts.Meter
	}

	attrs := []attribute.KeyValue{}
	if opts.Attributes != nil {
		attrs = append(attrs, parseAttributes(opts.Attributes)...)
	}

	observe := func(ctx context.Context, o metric.Int64Observer) error {
		value, err := opts.Callback(ctx)
		if err != nil {
			return err
		}
		o.Observe(value, metric.WithAttributes(attrs...))

		return nil
	}

	if _, err := meter.
		Int64ObservableGauge(
			fmt.Sprintf("%s_%s", prefix, opts.MetricName),
			metric.WithDescription(opts.Name),
			metric.WithUnit(opts.Unit),
			metric.WithInt64Callback(observe),
		); err != nil {
		log.From(ctx).Error().Err(err).Msg(fmt.Sprintf("error for meter: %s", opts.MetricName))
		return
	}
}

type HistogramOpt struct {
	Name        string
	Description string
	Meter       metric.Meter
	MetricName  string
	Attributes  map[string]any
	Unit        string
	Boundaries  []float64
}

// RecordIntHistogramMetric records the observed value for distributions.
// Bucket can be provided
func RecordIntHistogramMetric(ctx context.Context, value int64, opts HistogramOpt) {
	// use the global one by default
	meter := otel.Meter(opts.Name)
	if opts.Meter != nil {
		meter = opts.Meter
	}

	h, err := meter.
		Int64Histogram(
			fmt.Sprintf("%s_%s", prefix, opts.MetricName),
			metric.WithDescription(opts.Description),
			metric.WithUnit(opts.Unit),
			metric.WithExplicitBucketBoundaries(opts.Boundaries...),
		)

	if err != nil {
		log.From(ctx).Err(err).Msg(fmt.Sprintf("error for meter: %s", opts.MetricName))
		return
	}

	attrs := []attribute.KeyValue{}
	if opts.Attributes != nil {
		attrs = append(attrs, parseAttributes(opts.Attributes)...)
	}
	h.Record(ctx, value, metric.WithAttributes(attrs...))
}

// parseAttributes parses the attribute map into otel compatible attributes
func parseAttributes(attrs map[string]any) []attribute.KeyValue {
	result := make([]attribute.KeyValue, 0)

	for k, v := range attrs {
		attr := attribute.KeyValue{Key: attribute.Key(k)}

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
