package metrics

import "context"

func GaugeQueueItemLatencyEWMA(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_item_latency_ewma",
		Description: "The moving average of the queue item latency",
		Tags:        opts.Tags,
	})
}

func GaugeWorkerQueueCapacity(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_capacity_total",
		Description: "Capacity of current worker",
		Tags:        opts.Tags,
	})
}

func GaugeGlobalPartitionSize(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "global_partition_size",
		Description: "Number of total partitions in the global queue",
		Tags:        opts.Tags,
		Callback:    opts.Callback,
	})
}

func GaugePartitionSize(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "partition_size",
		Description: "Number of items in a particular partition",
		Tags:        opts.Tags,
	})
}

func GaugeQueueShardCount(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shards_count",
		Description: "Number of shards in the queue",
		Tags:        opts.Tags,
	})
}

func GaugeQueueShardGuaranteedCapacityCount(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shards_guaranteed_capacity_count",
		Description: "Shard guaranteed capacity",
		Tags:        opts.Tags,
	})
}

func GaugeQueueShardLeaseCount(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shards_lease_count",
		Description: "Shard current lease count",
		Tags:        opts.Tags,
	})
}

func GaugeQueueShardPartitionAvailableCount(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shard_partition_available_count",
		Description: "The number of shard partitions available",
		Tags:        opts.Tags,
	})
}

func GaugeSpanBatchProcessorBufferSize(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_batch_processor_buffer_size",
		Description: "The number of items in buffer point in time",
		Tags:        opts.Tags,
	})
}

func GaugeSpanBatchProcessorBufferKeys(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_batch_processor_buffer_keys",
		Description: "The number of keys used in buffer point in time",
		Tags:        opts.Tags,
	})
}
