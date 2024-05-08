package telemetry

import "context"

type CounterOpt struct {
	PkgName string
	Tags    map[string]any
}

func IncrQueuePeekedCounter(ctx context.Context, incr int64, opts CounterOpt) {
	recordCounterMetric(ctx, incr, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_peeked_total",
		Description: "The total number of queues peeked",
		Attributes:  opts.Tags,
	})
}

func IncrQueuePartitionLeasedCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_partition_lease_total",
		Description: "The total number of queue partitions leased",
		Attributes:  opts.Tags,
	})
}

func IncrQueueProcessNoCapacityCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_process_no_capacity_total",
		Description: "Total number of times the queue no longer has capacity to process items",
		Attributes:  opts.Tags,
	})
}

func IncrQueuePartitionLeaseContentionCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_partition_lease_contention_total",
		Description: "The total number of times contention occurred for partition leasing",
		Attributes:  opts.Tags,
	})
}

func IncrQueueItemLeaseContentionCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_item_lease_contention_total",
		Description: "The total number of times contention occurred for item leasing",
		Attributes:  opts.Tags,
	})
}

func IncrQueuePartitionProcessNoCapacityCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_partition_process_no_capacity_total",
		Description: "The number of times the queue no longer has capacity to process partitions",
		Attributes:  opts.Tags,
	})
}

func IncrQueueThrottledCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_throttled_total",
		Description: "The number of times the queue has been throttled",
		Attributes:  opts.Tags,
	})
}

func IncrQueuePartitionConcurrencyLimitCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_partition_concurrency_limit_total",
		Description: "The total number of times the queue partition hits concurrency limits",
		Attributes:  opts.Tags,
	})
}

func IncrQueueScanNoCapacityCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_scan_no_capacity_total",
		Description: "The total number of times the queue no longer have workers to scan",
		Attributes:  opts.Tags,
	})
}

func IncrQueuePartitionProcessedCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_partition_processed_total",
		Description: "The total number of queue partitions processed",
		Attributes:  opts.Tags,
	})
}

func IncrPartitionGoneCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "partition_gone_total",
		Description: "The total number of times a worker didn't find a partition",
		Attributes:  opts.Tags,
	})
}

func IncrQueueItemEnqueuedCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_items_enqueued_total",
		Description: "Total number of queue items enqueued",
		Attributes:  opts.Tags,
	})
}

func IncrQueueItemStartedCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_items_started_total",
		Description: "Total number of queue items started",
		Attributes:  opts.Tags,
	})
}

func IncrQueueItemErroredCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_items_errored_total",
		Description: "Total number of queue items errored",
		Attributes:  opts.Tags,
	})
}

func IncrQueueItemCompletedCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_items_completed_total",
		Description: "Total number of queue items completed",
		Attributes:  opts.Tags,
	})
}

func IncrQueueSequentialLeaseClaimsCounter(ctx context.Context, opts CounterOpt) {
	recordCounterMetric(ctx, 1, counterOpt{
		Name:        opts.PkgName,
		MetricName:  "queue_sequential_lease_claims_total",
		Description: "Total number of sequential lease claimed by worker",
		Attributes:  opts.Tags,
	})
}
