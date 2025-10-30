package metrics

import (
	"context"
)

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

func IncrSpanExportDataLoss(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_export_data_loss_total",
		Description: "Total number of data loss detected",
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

func IncrSpanBatchProcessorDroppedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_batch_processor_dropped_total",
		Description: "Total number of spans dropped for batch processing",
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

func IncrLogExportDataLoss(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "log_export_data_loss_total",
		Description: "Total number of data loss detected in logs",
		Tags:        opts.Tags,
	})
}

func IncrLogRecordExportedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "log_records_exported_total",
		Description: "Total number of log records exported",
		Tags:        opts.Tags,
	})
}

func IncrAggregatePausesEvaluatedCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "aggr_pauses_evaluated_total",
		Description: "Total number of pauses evaluated",
		Tags:        opts.Tags,
	})
}

func IncrAggregatePausesFoundCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "aggr_pauses_found_total",
		Description: "Total number of pauses founded via evaluation",
		Tags:        opts.Tags,
	})
}

func IncrConnectGatewayReceivedRouterGRPCMessageCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.router.received_pubsub_messages",
		Description: "Total number of router PubSub messages received by a connect gateway",
		Tags:        opts.Tags,
	})
}

func IncrConnectGatewayGRPCClientCreateCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.grpc.client_created",
		Description: "The total number of GRPC clients created",
		Tags:        opts.Tags,
	})
}

func IncrConnectGatewayGRPCClientGCCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.grpc.client_garbage_collected",
		Description: "The total number of GRPC client garbage collected",
		Tags:        opts.Tags,
	})
}

func IncrConnectGatewayGRPCForwardCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.grpc.forward_total",
		Description: "Total number of messages forwarded via gRPC to connect gateways",
		Tags:        opts.Tags,
	})
}

func IncrConnectGatewayGRPCReplyCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.grpc.reply_total",
		Description: "Total number of replies coming from connect gateways to executors",
		Tags:        opts.Tags,
	})
}

func IncrConnectGatewayGRPCClientFailureCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.grpc.client_failure_total",
		Description: "Total number of gRPC client creation failures for connect gateways",
		Tags:        opts.Tags,
	})
}

func IncrConnectRouterGRPCMessageSentCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.router.sent_pubsub_messages",
		Description: "Total number of router PubSub messages sent by the connect router",
		Tags:        opts.Tags,
	})
}

func IncrConnectGatewayReceivedWorkerMessageCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.worker.received_messages",
		Description: "Total number of worker messages received by a connect gateway",
		Tags:        opts.Tags,
	})
}

func IncrConnectGatewayReceiveConnectionAttemptCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.connection_attempts",
		Description: "Total number of worker connection attempts received by a connect gateway",
		Tags:        opts.Tags,
	})
}

func IncrConnectRouterNoHealthyConnectionCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_router.no_healthy_connections",
		Description: "Total number of attempts to forward a message without finding healthy connections",
		Tags:        opts.Tags,
	})
}

func IncrConnectRouterAllWorkersAtCapacityCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_router.all_workers_at_capacity",
		Description: "Total number of attempts to forward a message without finding any worker capacity",
		Tags:        opts.Tags,
	})
}

func IncrQueueContinuationAddedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_continuation_added_total",
		Description: "The total number of queue continuations added",
		Tags:        opts.Tags,
	})
}

func IncrQueueContinuationCooldownCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_continuation_cooldown_total",
		Description: "The total number of queue continuations added",
		Tags:        opts.Tags,
	})
}

func IncrQueueContinuationMaxCapcityCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_continuation_max_capacity_total",
		Description: "The total number of queue continuations added",
		Tags:        opts.Tags,
	})
}

func IncrQueueContinuationRemovedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_continuation_added_total",
		Description: "The total number of queue continuations added",
		Tags:        opts.Tags,
	})
}

func IncrQueueDebounceOperationCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_debounce_operation",
		Description: "The total number of debounce operations",
		Tags:        opts.Tags,
	})
}

func IncrBacklogProcessedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "backlog_processed_total",
		Description: "The total number of backlogs processed",
		Tags:        opts.Tags,
	})
}

func IncrBacklogNormalizationScannedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "backlog_normalization_scanned_total",
		Description: "The total number of backlogs that were scanned for normalization",
		Tags:        opts.Tags,
	})
}

func IncrBacklogNormalizedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "backlog_normalized_total",
		Description: "The total number of backlogs normalized",
		Tags:        opts.Tags,
	})
}

func IncrBacklogNormalizedItemCounter(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "backlog_normalized_item_total",
		Description: "The total number of items that were normalized",
		Tags:        opts.Tags,
	})
}

// NOTE: this is a metric that's mainly used for observing migrations to key queues
// it's not needed once the migration completes
func IncrBacklogRequeuedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "backlog_requeued_total",
		Description: "The total number of queue items that were requeued into backlogs from fn partition",
		Tags:        opts.Tags,
	})
}

func IncrRequeueExistingToBacklogCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "backlog_requeue_existing_total",
		Description: "The total number of existing queue items that were requeued into backlogs from fn partition after hitting constraints",
		Tags:        opts.Tags,
	})
}

func ActiveShadowScannerCount(ctx context.Context, incr int64, opts CounterOpt) {
	RecordUpDownCounterMetric(ctx, incr, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "active_shadow_scanner_count",
		Description: "The number of active shadow scaners",
		Tags:        opts.Tags,
	})
}

func IncrQueueShadowContinuationOpCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shadow_continuation_op",
		Description: "The total number of queue continuation ops",
		Tags:        opts.Tags,
	})
}

func IncrQueueBacklogRefilledCounter(ctx context.Context, incr int64, opts CounterOpt) {
	RecordCounterMetric(ctx, incr, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_backlog_refilled_total",
		Description: "The total number of items refilled from backlog",
		Tags:        opts.Tags,
	})
}

func IncrQueueBacklogRefillConstraintCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_backlog_refill_contrainted_total",
		Description: "The total number of times backlog was constrainted when attempt to refill",
		Tags:        opts.Tags,
	})
}

func IncrQueueShadowPartitionLeaseContentionCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shadow_partition_lease_contention_total",
		Description: "The total number of times shadow partition lease has contention",
		Tags:        opts.Tags,
	})
}

func IncrQueueShadowPartitionGoneCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shadow_partition_gone_total",
		Description: "The total number of times shadow worker didn't find a partition",
		Tags:        opts.Tags,
	})
}

func IncrQueueShadowPartitionProcessedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shadow_partition_processed_total",
		Description: "The total number of shadow partition processed",
		Tags:        opts.Tags,
	})
}

func IncrQueuePeekLeaseContentionCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_peek_lease_contention",
		Description: "Total number of leased queue items peeked for a partition",
		Tags:        opts.Tags,
	})
}

func IncrQueueOutdatedBacklogCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_shadow_outdated_backlog_total",
		Description: "The total number of times outdated backlogs were detected",
		Tags:        opts.Tags,
	})
}

func IncrRunFinalizedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "run_finalized_total",
		Description: "The total number of calls to finalize a run.",
		Tags:        opts.Tags,
	})
}

func IncrStateWrittenCounter(ctx context.Context, size int, opts CounterOpt) {
	RecordCounterMetric(ctx, int64(size), CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "state_store_bytes_written",
		Description: "The total number of bytes written to the state store",
		Tags:        opts.Tags,
	})
}

func IncrHTTPAPIRequestsCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "http_api_requests_total",
		Description: "Total number of HTTP API requests",
		Tags:        opts.Tags,
	})
}

func IncrQueueActiveCheckInvalidItemsFoundCounter(ctx context.Context, val int64, opts CounterOpt) {
	RecordCounterMetric(ctx, val, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_active_check_invalid_items_found_total",
		Description: "The total number of invalid items found during an active check",
		Tags:        opts.Tags,
	})
}

func IncrQueueActiveCheckInvalidItemsRemovedCounter(ctx context.Context, val int64, opts CounterOpt) {
	RecordCounterMetric(ctx, val, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_active_check_invalid_items_removed_total",
		Description: "The total number of invalid items removed during an active check",
		Tags:        opts.Tags,
	})
}

func IncrQueueActiveCheckAccountScannedCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_active_check_account_scanned_total",
		Description: "The total number of times an account was scanned during an active check",
		Tags:        opts.Tags,
	})
}

func ActiveBacklogNormalizeCount(ctx context.Context, incr int64, opts CounterOpt) {
	RecordUpDownCounterMetric(ctx, incr, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "active_backlog_normalize_count",
		Description: "The number of active backlog normalizations",
		Tags:        opts.Tags,
	})
}

func IncrAPICacheHit(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "http_api_cache_hit",
		Description: "The number of times a HTTP API request is served from cache",
		Tags:        opts.Tags,
	})
}

func IncrAPICacheMiss(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "http_api_cache_miss",
		Description: "The number of times a HTTP API request is not served from cache",
		Tags:        opts.Tags,
	})
}

func IncrQueueThrottleKeyExpressionMismatchCounter(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_throttle_key_expr_mismatch",
		Description: "The total number of times a throttle key expression mismatch was detected",
		Tags:        opts.Tags,
	})
}

func IncrPausesFlushedToBlocks(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "pauses_flushed_to_blocks_total",
		Description: "Total number of pauses flushed to blocks",
		Tags:        opts.Tags,
	})
}

func IncrPausesBlocksCreated(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "pauses_blocks_created_total",
		Description: "Total number of pause blocks created",
		Tags:        opts.Tags,
	})
}

func IncrPausesDeletedAfterBlockFlush(ctx context.Context, value int64, opts CounterOpt) {
	RecordCounterMetric(ctx, value, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "pauses_deleted_after_flush_total",
		Description: "Total number of pauses deleted after flushing them to blocks",
		Tags:        opts.Tags,
	})
}
