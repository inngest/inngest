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

func GaugeQueueGuaranteedCapacityCount(ctx context.Context, value int64, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_guaranteed_capacity_count",
		Description: "Number of accounts with guaranteed capacity in the queue",
		Tags:        opts.Tags,
		Callback: func(ctx context.Context) (int64, error) {
			return value, nil
		},
	})
}

func GaugeQueueAccountGuaranteedCapacityCount(ctx context.Context, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_account_guaranteed_capacity_count",
		Description: "Account guaranteed capacity",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}

func GaugeQueueGuaranteedCapacityLeaseCount(ctx context.Context, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_guaranteed_capacity_lease_count",
		Description: "Guaranteed capacity current lease count",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}

func GaugeQueueGuaranteedCapacityAccountPartitionAvailableCount(ctx context.Context, opts GaugeOpt) {
	RegisterAsyncGauge(ctx, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_guaranteed_capacity_account_partition_available_count",
		Description: "The number of partitions available for account with guaranteed capacity",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}
