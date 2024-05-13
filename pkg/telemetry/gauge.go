package telemetry

import "context"

type GaugeOpt struct {
	PkgName  string
	Tags     map[string]any
	Observer GaugeCallback
}

func GaugeQueueItemLatencyEWMA(ctx context.Context, value int64, opts GaugeOpt) {
	registerAsyncGauge(ctx, gaugeOpt{
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
	registerAsyncGauge(ctx, gaugeOpt{
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
	registerAsyncGauge(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_global_partition_count",
		Description: "Number of total partitions in the global queue",
		Attributes:  opts.Tags,
		Callback:    opts.Observer,
	})
}

func GaugeGlobalQueuePartitionAvailable(ctx context.Context, opts GaugeOpt) {
	registerAsyncGauge(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_global_partition_available_count",
		Description: "Number of available partitions in the global queue",
		Attributes:  opts.Tags,
		Callback:    opts.Observer,
	})
}

func GaugeQueueShardCount(ctx context.Context, value int64, opts GaugeOpt) {
	registerAsyncGauge(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_shards_count",
		Description: "Number of shards in the queue",
		Attributes:  opts.Tags,
		Callback: func(ctx context.Context) (int64, error) {
			return value, nil
		},
	})
}

func GaugeQueueShardGuaranteedCapacityCount(ctx context.Context, value int64, opts GaugeOpt) {
	registerAsyncGauge(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_shards_guaranteed_capacity_count",
		Description: "Shard guaranteed capacity",
		Attributes:  opts.Tags,
		Callback: func(ctx context.Context) (int64, error) {
			return value, nil
		},
	})
}

func GaugeQueueShardLeaseCount(ctx context.Context, value int64, opts GaugeOpt) {
	registerAsyncGauge(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_shards_lease_count",
		Description: "Shard current lease count",
		Attributes:  opts.Tags,
		Callback: func(ctx context.Context) (int64, error) {
			return value, nil
		},
	})
}

func GaugeQueueShardPartitionAvailableCount(ctx context.Context, opts GaugeOpt) {
	registerAsyncGauge(ctx, gaugeOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_shard_partition_available_count",
		Description: "The number of shard partitions available",
		Attributes:  opts.Tags,
		Callback:    opts.Observer,
	})
}
