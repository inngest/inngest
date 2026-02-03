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

func GaugeSpanBatchProcessorNatsAsyncPending(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_batch_processor_async_pending",
		Description: "The number of messages pending to publish to NATS stream",
		Tags:        opts.Tags,
	})
}

func GaugeSpanExporterBuffer(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_exporter_buffer_size",
		Description: "The number of messages in the NATS exporter buffer",
		Tags:        opts.Tags,
	})
}

func GaugeConnectGatewayActiveConnections(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.connections.active",
		Description: "The number of active connections on a connect gateway",
		Tags:        opts.Tags,
	})
}

func GaugeConnectActiveGateway(ctx context.Context, value int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, value, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.gateways.active",
		Description: "Total number of active connect gateways",
		Tags:        opts.Tags,
	})
}

func GaugeConnectDrainingGateway(ctx context.Context, value int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, value, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.gateways.draining",
		Description: "Total number of draining connect gateways",
		Tags:        opts.Tags,
	})
}

func GaugeShadowPartitionSize(ctx context.Context, value int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, value, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "shadow_partition_size",
		Description: "The number of backlogs in a shadow partition",
		Tags:        opts.Tags,
	})
}

func GaugeAggregatorPendingDeletes(ctx context.Context, value int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, value, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "aggr_evaluables_pending_deletes",
		Description: "Number of pending deletes in the aggregator",
		Tags:        opts.Tags,
	})
}
