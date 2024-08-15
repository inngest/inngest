package metrics

import "context"

var (
	// in milliseconds
	DefaultBoundaries          = []float64{10, 50, 100, 200, 500, 1000, 2000, 5000, 10000}
	QueueItemLatencyBoundaries = []float64{
		5, 10, 50, 100, 200, 500, // < 1s
		1000, 2000, 5000, 30_000, // < 1m
		60_000, 300_000, // < 10m
		600_000, 1_800_000, // < 1h
	}

	processPartitionBoundaries = []float64{
		5, 10, 25, 50, 100, 200, // < 1s
		400, 600, 800, 1_000,
		1_500, 2_000, 4_000,
		8_000, 15_000,
	}

	peekSizeBoundaries = []float64{10, 30, 50, 100, 250, 500, 1000, 3000, 5000}
)

func HistogramQueueItemLatency(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_item_latency_duration",
		Description: "Distribution of queue item latency",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  QueueItemLatencyBoundaries,
	})
}

func HistogramProcessPartitionDuration(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_process_partition_duration",
		Description: "Distribution of how long it takes to process a partition",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  processPartitionBoundaries,
	})
}

func HistogramQueueOperationDuration(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_operation_duration",
		Description: "Distribution of atomic queueing operation durations",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  processPartitionBoundaries,
	})
}

func HistogramQueuePeekSize(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_peek_size",
		Description: "Distribution of the number of items being peeked on each call",
		Tags:        opts.Tags,
		Boundaries:  peekSizeBoundaries,
	})
}

func HistogramQueuePeekEWMA(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_peek_ewma",
		Description: "Distribution of the EWMA values for peeks",
		Tags:        opts.Tags,
		Boundaries:  peekSizeBoundaries,
	})
}

func HistogramRedisCommandDuration(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "redis_command_duration",
		Description: "Redis command duration",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}
