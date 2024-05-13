package telemetry

import "context"

func IncrQueuePeekedCounter(ctx context.Context, incr int64, opts CounterOpt) {
	RecordCounterMetric(ctx, incr, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_peeked_total",
		Description: "The total number of queues peeked",
		Tags:        opts.Tags,
	})
}

func IncrQueuePartitionLeasedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_partition_lease_total",
		Description: "The total number of queue partitions leased",
		Tags:        opts.Tags,
	})
}

func IncrQueueProcessNoCapacityCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_process_no_capacity_total",
		Description: "Total number of times the queue no longer has capacity to process items",
		Tags:        opts.Tags,
	})
}

func IncrQueuePartitionLeaseContentionCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_partition_lease_contention_total",
		Description: "The total number of times contention occurred for partition leasing",
		Tags:        opts.Tags,
	})
}

func IncrQueueItemLeaseContentionCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_item_lease_contention_total",
		Description: "The total number of times contention occurred for item leasing",
		Tags:        opts.Tags,
	})
}

func IncrQueuePartitionProcessNoCapacityCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_partition_process_no_capacity_total",
		Description: "The number of times the queue no longer has capacity to process partitions",
		Tags:        opts.Tags,
	})
}

func IncrQueueThrottledCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_throttled_total",
		Description: "The number of times the queue has been throttled",
		Tags:        opts.Tags,
	})
}

func IncrQueuePartitionConcurrencyLimitCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_partition_concurrency_limit_total",
		Description: "The total number of times the queue partition hits concurrency limits",
		Tags:        opts.Tags,
	})
}

func IncrQueueScanNoCapacityCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_scan_no_capacity_total",
		Description: "The total number of times the queue no longer have workers to scan",
		Tags:        opts.Tags,
	})
}

func IncrQueuePartitionProcessedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_partition_processed_total",
		Description: "The total number of queue partitions processed",
		Tags:        opts.Tags,
	})
}

func IncrPartitionGoneCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "partition_gone_total",
		Description: "The total number of times a worker didn't find a partition",
		Tags:        opts.Tags,
	})
}

func IncrQueueItemEnqueuedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_items_enqueued_total",
		Description: "Total number of queue items enqueued",
		Tags:        opts.Tags,
	})
}

func IncrQueueItemStartedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_items_started_total",
		Description: "Total number of queue items started",
		Tags:        opts.Tags,
	})
}

func IncrQueueItemErroredCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_items_errored_total",
		Description: "Total number of queue items errored",
		Tags:        opts.Tags,
	})
}

func IncrQueueItemCompletedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_items_completed_total",
		Description: "Total number of queue items completed",
		Tags:        opts.Tags,
	})
}

func IncrQueueSequentialLeaseClaimsCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_sequential_lease_claims_total",
		Description: "Total number of sequential lease claimed by worker",
		Tags:        opts.Tags,
	})
}
