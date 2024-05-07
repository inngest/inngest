package telemetry

import "context"

type GaugeOpt struct {
	PkgName  string
	Tags     map[string]any
	Observer GaugeCallback
}

func GaugeQueueItemLatencyEWMA(ctx context.Context, value int64, opts GaugeOpt) {
	recordGaugeMetric(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_item_latency_ewma",
		Description: "The moving average of the queue item latency",
		Attributes:  opts.Tags,
		Callback: func(ctx context.Context) (int64, error) {
			return value, nil
		},
	})
}

func GaugeWorkerQueueCapacity(ctx context.Context, value int64, opts GaugeOpt) {
	recordGaugeMetric(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_capacity_total",
		Description: "Capacity of current worker",
		Attributes:  opts.Tags,
		Callback: func(ctx context.Context) (int64, error) {
			return value, nil
		},
	})
}

func GaugeGlobalQueuePartitionCount(ctx context.Context, opts GaugeOpt) {
	recordGaugeMetric(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_global_partition_total_count",
		Description: "Number of total partitions in the global queue",
		Attributes:  opts.Tags,
		Callback:    opts.Observer,
	})
}

func GaugeGlobalQueuePartitionAvailable(ctx context.Context, opts GaugeOpt) {
	recordGaugeMetric(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_global_partition_available_count",
		Description: "Number of available partitions in the global queue",
		Attributes:  opts.Tags,
		Callback:    opts.Observer,
	})
}
