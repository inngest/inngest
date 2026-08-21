package metrics

import (
	"context"
	"time"
)

var (
	batchItemCountBoundaries = []float64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1_000, 2_500, 5_000, 10_000}
	batchByteBoundaries      = []float64{
		1 * 1024,
		4 * 1024,
		16 * 1024,
		64 * 1024,
		256 * 1024,
		1 * 1024 * 1024,
		2 * 1024 * 1024,
		4 * 1024 * 1024,
		8 * 1024 * 1024,
		16 * 1024 * 1024,
	}
)

// IncrBatchMetricTierRefreshCounter records background batch metric tier
// refreshes by whether the lookup resolved a tier. An unknown result is not
// necessarily an error; it can also mean missing data or an unsupported plan.
func IncrBatchMetricTierRefreshCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_metric_tier_refresh_total",
		Description: "Total number of background batch metric tier refreshes",
		Tags:        opts.Tags,
	})
}

// HistogramBatchPayloadSizeBytes records serialized bytes committed by one
// append or bulk-append Redis operation. Its exported sum is the total number
// of serialized bytes committed.
func HistogramBatchPayloadSizeBytes(ctx context.Context, bytes int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, bytes, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_payload_size_bytes",
		Description: "Distribution of serialized BatchItem bytes committed per Redis operation",
		Unit:        "By",
		Tags:        opts.Tags,
		Boundaries:  batchByteBoundaries,
	})
}

// HistogramBatchListResidentSizeBytes records a write-weighted, append-time
// Redis MEMORY USAGE sample for the current batch list. It is not an inventory
// or a final/peak batch-size measurement, and it excludes pointers, metadata,
// and idempotency keys.
func HistogramBatchListResidentSizeBytes(ctx context.Context, bytes int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, bytes, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_list_resident_size_bytes",
		Description: "Write-weighted distribution of append-time Redis resident bytes for the current batch list only",
		Unit:        "By",
		Tags:        opts.Tags,
		Boundaries:  batchByteBoundaries,
	})
}

// HistogramBatchListItemCount records the write-weighted current batch list
// length paired with each resident-size observation.
func HistogramBatchListItemCount(ctx context.Context, count int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, count, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_list_item_count",
		Description: "Write-weighted distribution of current batch list lengths after Redis writes",
		Tags:        opts.Tags,
		Boundaries:  batchItemCountBoundaries,
	})
}

func HistogramBatchRetrieveItemsDuration(ctx context.Context, dur time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_retrieve_items_duration",
		Description: "Distribution of batch item retrieval latency",
		Unit:        "ms",
		Tags:        opts.Tags,
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramBatchRetrievedItemCount(ctx context.Context, count int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, count, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_retrieved_item_count",
		Description: "Distribution of items returned by batch retrievals",
		Tags:        opts.Tags,
		Boundaries:  batchItemCountBoundaries,
	})
}

func HistogramBatchDeleteKeysDuration(ctx context.Context, dur time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_delete_keys_duration",
		Description: "Distribution of batch key deletion latency",
		Unit:        "ms",
		Tags:        opts.Tags,
		Boundaries:  DefaultBoundaries,
	})
}
