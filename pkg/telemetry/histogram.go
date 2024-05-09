package telemetry

import "context"

var (
	// in milliseconds
	// defaultBoundaries          = []float64{10, 50, 100, 200, 500, 1000, 2000, 5000, 10000}
	queueItemLatencyBoundaries = []float64{
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
)

type HistogramOpt struct {
	PkgName string
	Tags    map[string]any
}

func HistogramQueueItemLatency(ctx context.Context, value int64, opts HistogramOpt) {
	recordIntHistogramMetric(ctx, value, histogramOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_item_latency_duration",
		Description: "Distribution of queue item latency",
		Attributes:  opts.Tags,
		Unit:        "ms",
		Boundaries:  queueItemLatencyBoundaries,
	})
}

func HistogramProcessPartitionDration(ctx context.Context, value int64, opts HistogramOpt) {
	recordIntHistogramMetric(ctx, value, histogramOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_process_partition_duration",
		Description: "Distribution of how long it takes to process a partition",
		Attributes:  opts.Tags,
		Unit:        "ms",
		Boundaries:  processPartitionBoundaries,
	})
}
