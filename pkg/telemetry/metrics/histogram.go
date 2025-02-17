package metrics

import "context"

var (
	// in milliseconds
	DefaultBoundaries          = []float64{10, 50, 100, 200, 500, 1000, 2000, 5000, 10000}
	QueueItemLatencyBoundaries = []float64{
		5, 10, 50, 100, 200, 500, // < 1s
		1000, 2000, 5000, 30_000, // < 1m
		60_000, 300_000, // < 10m
		600_000, 1_800_000, // < 1h
	}

	processPartitionBoundaries = []float64{
		5, 10, 25, 50, 100, 200, // < 1s
		400, 600, 800, 1_000,
		1_500, 2_000, 4_000,
		8_000, 15_000,
	}

	peekSizeBoundaries = []float64{10, 30, 50, 100, 250, 500, 1000, 3000, 5000}

	PausesBoundaries = []float64{
		5, 10, 50, 100, 200, 500, // < 1s
		1000, 2000, 5000, 30_000, // < 1m
		60_000, 300_000, // < 10m
		600_000, 1_800_000, // < 1h
		3_600_000, // 1h
	}
)

func HistogramQueueItemLatency(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_item_latency_duration",
		Description: "Distribution of queue item latency",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  QueueItemLatencyBoundaries,
	})
}

func HistogramProcessPartitionDuration(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_process_partition_duration",
		Description: "Distribution of how long it takes to process a partition",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  processPartitionBoundaries,
	})
}

func HistogramQueueOperationDuration(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_operation_duration",
		Description: "Distribution of atomic queueing operation durations",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  processPartitionBoundaries,
	})
}

func HistogramQueuePeekSize(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_peek_size",
		Description: "Distribution of the number of items being peeked on each call",
		Tags:        opts.Tags,
		Boundaries:  peekSizeBoundaries,
	})
}

func HistogramQueuePeekEWMA(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_peek_ewma",
		Description: "Distribution of the EWMA values for peeks",
		Tags:        opts.Tags,
		Boundaries:  peekSizeBoundaries,
	})
}

func HistogramRedisCommandDuration(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "redis_command_duration",
		Description: "Redis command duration",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramAggregatePausesLoadDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "aggr_pauses_load_duration",
		Description: "Duration for loading aggregate pauses processing",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}

func HistogramAggregatePausesEvalDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "aggr_pauses_eval_duration",
		Description: "Duration for evaluating aggregate pauses processing",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}

func HistogramConnectProxyAckTime(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_proxy.ack_time",
		Description: "Duration for a request to be acknowledged",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}

func HistogramConnectExecutorEndToEndDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_proxy.end_to_end_duration",
		Description: "Duration for a request to be proxied and for the response to be received by the executor.",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}

func HistogramConnectSetupDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.connection.setup_duration",
		Description: "Duration from starting to set up the connection to being fully ready.",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}

func HistogramConnectSyncDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.connection.sync_duration",
		Description: "End to end duration for the out-of-band sync request initiated by the connect gateway.",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}

func HistogramConnectAppLoaderDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "connect_gateway.connection.app_loader_duration",
		Description: "Duration for loading the app during connection setup in the connect gateway.",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}
