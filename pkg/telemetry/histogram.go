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
