package metrics

import (
	"context"
)

var (
	batchBufferWaitDurationBoundaries = []float64{5, 10, 25, 50, 100, 200, 500, 1000, 2000, 3000, 5000}
	batchBufferFlushSizeBoundaries    = []float64{1, 2, 5, 10, 20, 50, 100, 200, 500}
)

// #1 - Gauge: items currently in-memory waiting to flush
func GaugeBatchBufferItemsPending(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_items_pending",
		Description: "Items currently in-memory waiting to flush",
		Tags:        opts.Tags,
	})
}

// #2 - Gauge: active buffer keys (fn + batch pointer combos)
func GaugeBatchBufferKeysActive(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_keys_active",
		Description: "Active buffer keys (fn + batch pointer combos)",
		Tags:        opts.Tags,
	})
}

// #3 - Counter: flush count by trigger type
func IncrBatchBufferFlushCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_flush_total",
		Description: "Total number of batch buffer flushes",
		Tags:        opts.Tags,
	})
}

// #4 - Counter: items flushed by trigger type
func IncrBatchBufferItemsFlushedCounter(ctx context.Context, count int64, opts CounterOpt) {
	RecordCounterMetric(ctx, count, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_items_flushed_total",
		Description: "Total number of items flushed from batch buffer",
		Tags:        opts.Tags,
	})
}

// #5 - Counter: in-memory duplicate catches
func IncrBatchBufferDedupCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_dedup_total",
		Description: "Total number of in-memory duplicate event catches",
		Tags:        opts.Tags,
	})
}

// #6 - Histogram: time items wait in buffer before flush
func HistogramBatchBufferWaitDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_wait_duration",
		Description: "Distribution of time items wait in buffer before flush",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  batchBufferWaitDurationBoundaries,
	})
}

// #7 - Histogram: items per flush
func HistogramBatchBufferFlushSize(ctx context.Context, val int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, val, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_flush_size",
		Description: "Distribution of items per flush",
		Tags:        opts.Tags,
		Boundaries:  batchBufferFlushSizeBoundaries,
	})
}

// #8 - Histogram: BulkAppend Redis call latency
func HistogramBatchBufferRedisFlushDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_redis_flush_duration",
		Description: "Distribution of BulkAppend Redis call latency",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

// #9 - Counter: BulkAppend results by status
func IncrBatchBufferBulkAppendCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_bulk_append_total",
		Description: "Total number of BulkAppend results by status",
		Tags:        opts.Tags,
	})
}

// #10 - Counter: items committed to Redis
func IncrBatchBufferItemsCommittedCounter(ctx context.Context, count int64, opts CounterOpt) {
	RecordCounterMetric(ctx, count, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_items_committed_total",
		Description: "Total number of items committed to Redis",
		Tags:        opts.Tags,
	})
}

// #11 - Counter: Redis-level duplicates
func IncrBatchBufferItemsDuplicatedCounter(ctx context.Context, count int64, opts CounterOpt) {
	RecordCounterMetric(ctx, count, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_items_duplicated_total",
		Description: "Total number of Redis-level duplicate items",
		Tags:        opts.Tags,
	})
}

// #12 - Counter: errors by error_type
func IncrBatchBufferErrorsCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_errors_total",
		Description: "Total number of batch buffer errors",
		Tags:        opts.Tags,
	})
}

// #13 - Counter: execution schedule operations
func IncrBatchBufferScheduleCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_schedule_total",
		Description: "Total number of batch buffer execution schedule operations",
		Tags:        opts.Tags,
	})
}
