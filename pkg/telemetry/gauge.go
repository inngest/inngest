package telemetry

import "context"

func GaugeQueueItemLatencyEWMA(ctx context.Context, value int64, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_item_latency_ewma",
		Description: "The moving average of the queue item latency",
		Tags:        opts.Tags,
		Callback: func(ctx context.Context) (int64, error) {
			return value, nil
		},
	})
}

func GaugeWorkerQueueCapacity(ctx context.Context, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_capacity_total",
		Description: "Capacity of current worker",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}

func GaugeGlobalQueuePartitionCount(ctx context.Context, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_global_partition_count",
		Description: "Number of total partitions in the global queue",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}

func GaugeGlobalQueuePartitionAvailable(ctx context.Context, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_global_partition_available_count",
		Description: "Number of available partitions in the global queue",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}

func GaugeQueueShardCount(ctx context.Context, value int64, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shards_count",
		Description: "Number of shards in the queue",
		Tags:        opts.Tags,
		Callback: func(ctx context.Context) (int64, error) {
			return value, nil
		},
	})
}

func GaugeQueueShardGuaranteedCapacityCount(ctx context.Context, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shards_guaranteed_capacity_count",
		Description: "Shard guaranteed capacity",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}

func GaugeQueueShardLeaseCount(ctx context.Context, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shards_lease_count",
		Description: "Shard current lease count",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}

func GaugeQueueShardPartitionAvailableCount(ctx context.Context, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shard_partition_available_count",
		Description: "The number of shard partitions available",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}
