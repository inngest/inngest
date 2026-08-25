package metrics

import (
	"context"
	"time"
)

var (
	batchResidencyDurationBoundaries = []float64{
		100,
		500,
		1_000,
		5_000,
		10_000,
		30_000,
		60_000,
		5 * 60_000,
		10 * 60_000,
		20 * 60_000,
		30 * 60_000,
		60 * 60_000,
	}
)

// IncrBatchCommittedBytesCounter records serialized bytes committed to batch
// storage. The counter can be summed by account_tier and backend for throughput
// and capacity planning.
func IncrBatchCommittedBytesCounter(ctx context.Context, bytes int64, opts CounterOpt) {
	RecordCounterMetric(ctx, bytes, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_committed_bytes_total",
		Description: "Total serialized BatchItem bytes committed to batch storage",
		Tags:        opts.Tags,
	})
}

// GaugeBatchBufferBytesPending records the event payload bytes currently held
// in the in-process batching buffer. It does not attempt to include Go object
// overhead.
func GaugeBatchBufferBytesPending(ctx context.Context, bytes int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, bytes, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_bytes_pending",
		Description: "Event payload bytes currently held in the in-process batching buffer",
		Tags:        opts.Tags,
	})
}

// IncrBatchBufferBytesAddedCounter records event payload bytes accepted into
// the in-process batching buffer.
func IncrBatchBufferBytesAddedCounter(ctx context.Context, bytes int64, opts CounterOpt) {
	RecordCounterMetric(ctx, bytes, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_bytes_added_total",
		Description: "Total event payload bytes accepted into the in-process batching buffer",
		Tags:        opts.Tags,
	})
}

// IncrBatchBufferBytesRemovedCounter records event payload bytes released from
// the in-process batching buffer when a flush takes ownership of them.
func IncrBatchBufferBytesRemovedCounter(ctx context.Context, bytes int64, opts CounterOpt) {
	RecordCounterMetric(ctx, bytes, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_buffer_bytes_removed_total",
		Description: "Total event payload bytes released from the in-process batching buffer",
		Tags:        opts.Tags,
	})
}

// HistogramBatchResidencyDuration records how long a batch list existed before
// it was actually deleted. Its sum is total batch residency time and its count
// is the number of deleted batches.
func HistogramBatchResidencyDuration(ctx context.Context, dur time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_residency_duration",
		Description: "Distribution of time batch lists remain in batch storage before deletion",
		Unit:        "ms",
		Tags:        opts.Tags,
		Boundaries:  batchResidencyDurationBoundaries,
	})
}
