package metrics

import "context"

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

func IncrQueueItemProcessedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_process_item_total",
		Description: "Total number of queue items processed",
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

func IncrQueuePartitionProcessNoCapacityCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_partition_process_no_capacity_total",
		Description: "The number of times the queue no longer has capacity to process partitions",
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

func IncrQueueScanCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_scan_total",
		Description: "The total number of times we scanned the queue",
		Tags:        opts.Tags,
	})
}

func IncrQueuePartitionScannedCounter(ctx context.Context, parts int64, opts CounterOpt) {
	RecordCounterMetric(ctx, parts, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_partitions_scanned_total",
		Description: "The total number of partitions we peeked in a single scan loop",
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

func IncrQueueItemStatusCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_item_status_total",
		Description: "Total number of queue items in each status",
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

func WorkerQueueCapacityCounter(ctx context.Context, incr int64, opts CounterOpt) {
	RecordUpDownCounterMetric(ctx, incr, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_worker_capacity_in_use",
		Description: "Capacity of current worker",
		Tags:        opts.Tags,
	})
}

func IncrBatchScheduledCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "new_batch_scheduled_total",
		Description: "Total number of new batch scheduled",
		Tags:        opts.Tags,
	})
}

func IncrBatchProcessStartCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "batch_processing_started_total",
		Description: "Total number of completed batches for event batching, either through timeout or full batch.",
		Tags:        opts.Tags,
	})
}

func IncrInstrumentationLeaseClaimsCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_instrumentation_lease_claims_total",
		Description: "Total number of instrumentation lease claimed by executors",
		Tags:        opts.Tags,
	})
}

func IncrSpanExportedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_exported_total",
		Description: "Total number of run spans exported",
		Tags:        opts.Tags,
	})
}

func IncrSpanBatchProcessorEnqueuedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_batch_processor_enqueued_total",
		Description: "Total number of spans enqueued for batch processing",
		Tags:        opts.Tags,
	})
}

func IncrSpanBatchProcessorAttemptCounter(ctx context.Context, incr int64, opts CounterOpt) {
	RecordCounterMetric(ctx, incr, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_batch_processor_attempt_total",
		Description: "Total number of spans attempted to export",
		Tags:        opts.Tags,
	})
}

func IncrSpanBatchProcessorDeadLetterCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_batch_processor_deadletter_total",
		Description: "Total number of spans that got passed into the deadletter stream",
		Tags:        opts.Tags,
	})
}

func IncrSpanBatchProcessorDeadLetterPublishStatusCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_batch_processor_deadletter_publish_status_total",
		Description: "Total number of spans that got published to the deadletter stream and their status",
		Tags:        opts.Tags,
	})
}
