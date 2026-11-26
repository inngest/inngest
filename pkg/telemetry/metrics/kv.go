package metrics

import "context"

// kvOperationDurationBoundaries covers one redis or sql round trip up to a
// slow call that holds a loop key for seconds.
var kvOperationDurationBoundaries = []float64{
	1, 2, 5, 10, 25, 50, 100,
	250, 500, 1_000, 2_500, 5_000, 10_000,
}

// HistogramKVOperationDuration records one kv call.  the histogram count is
// the operation counter, so tags carry the op, namespace, and result.
func HistogramKVOperationDuration(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "kv_operation_duration",
		Description: "Distribution of kv store operation durations",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  kvOperationDurationBoundaries,
	})
}

// HistogramKVValueSize records the value length of a successful get or set.
func HistogramKVValueSize(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "kv_value_size_bytes",
		Description: "Distribution of kv value sizes read and written",
		Tags:        opts.Tags,
		Unit:        "bytes",
		Boundaries:  stateStoreBytesWrittenBoundaries,
	})
}

// IncrKVBytesCounter adds the bytes a successful get or set moved.
func IncrKVBytesCounter(ctx context.Context, size int64, opts CounterOpt) {
	RecordCounterMetric(ctx, size, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "kv_bytes_total",
		Description: "The total number of bytes read from and written to the kv store",
		Tags:        opts.Tags,
		Unit:        "bytes",
	})
}
