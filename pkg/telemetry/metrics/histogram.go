package metrics

import (
	"context"
	"time"
)

var (
	// in milliseconds
	DefaultBoundaries          = []float64{10, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 30000, 60000}
	QueueItemLatencyBoundaries = []float64{
		5, 10, 50, 100, 200, 500, // < 1s
		1000, 2000, 5000, 30_000, // < 1m
		60_000, 300_000, // < 10m
		600_000, 1_800_000, // < 1h
	}

	cancellationReadDurationBoundaries = []float64{
		5, 10, 50, 100, 200, 500, // < 1s
		1000, 2000, 5000, 30_000, // < 1m
		60_000, 120_000, 300_000, // < 10m
		600_000, 1_800_000, // < 1h
	}

	processPartitionBoundaries = []float64{
		5, 10, 25, 50, 100, 200, // < 1s
		400, 600, 800, 1_000,
		1_500, 2_000, 4_000,
		8_000, 15_000,
	}

	peekSizeBoundaries = []float64{10, 30, 50, 100, 250, 500, 1000, 3000, 5000}

	cancellationReadSizeBoundaries = []float64{10, 50, 100, 250, 500, 1000, 2500, 5000, 10_000, 25_000, 50_000, 75_000, 100_000}

	PausesBoundaries = []float64{
		5, 10, 50, 100, 200, 500, // < 1s
		1000, 2000, 5000, 30_000, // < 1m
		60_000, 300_000, // < 10m
		600_000, 1_800_000, // < 1h
		3_600_000, // 1h
	}

	constraintAPIRequestStateSizeBoundaries = []float64{
		512,             // 512 bytes
		1024,            // 1 KiB
		4 * 1024,        // 4 KiB
		8 * 1024,        // 8 KiB
		16 * 1024,       // 16 KiB
		32 * 1024,       // 32 KiB
		64 * 1024,       // 64 KiB
		128 * 1024,      // 128 KiB
		256 * 1024,      // 256 KiB
		512 * 1024,      // 512 KiB
		1024 * 1024,     // 1 MiB
		2 * 1024 * 1024, // 2 MiB
		4 * 1024 * 1024, // 4 MiB
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

func HistogramQueueOperationDelay(ctx context.Context, delay time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, delay.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_operation_delay",
		Description: "Distribution of queue item operation delays",
		Tags:        opts.Tags,
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramQueueActiveCheckDuration(ctx context.Context, delay time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, delay.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_active_check_duration",
		Description: "Distribution of active check durations",
		Tags:        opts.Tags,
		Boundaries:  DefaultBoundaries,
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
		Boundaries: []float64{
			5, 10, 25, 50, 100, 250, 500, 1000, // < 1s
			5000, 10000, 30_000, 60_000, 120_000,
			300_000, 900_000, 1_800_000, // 5m, 15m, 30m
			3_600_000, 7_200_000, // 1h, 2h
		},
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

func HistogramHTTPDNSLookupDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "httpdriver_dns_lookup_duration",
		Description: "Distribution of DNS lookup durations",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries: []float64{
			5, 10, 25, 50, 100, 250, 500, 1000, // < 1s
			2500, 5000, 10000,
		},
	})
}

func HistogramHTTPTCPConnDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "httpdriver_tcp_conn_duration",
		Description: "Distribution of TCP Handshake durations",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries: []float64{
			5, 10, 25, 50, 100, 250, 500, 1000, // < 1s
			2500, 5000, 10000, 30_000,
		},
	})
}

func HistogramHTTPTLSHandshakeDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "httpdriver_tls_handshake_duration",
		Description: "Distribution of TLS Handshake durations",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries: []float64{
			5, 10, 25, 50, 100, 250, 500, 1000, // < 1s
			2500, 5000, 10000, 30_000, 60_000, 120_000,
			300_000,
		},
	})
}

func HistogramHTTPServerProcessingDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "httpdriver_request_sent_duration",
		Description: "Distribution of the outbound request durations",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries: []float64{
			5, 10, 25, 50, 100, 250, 500, 1000, // < 1s
			5000, 10000, 30_000, 60_000, 120_000,
			300_000, 900_000, 1_800_000, // 5m, 15m, 30m
		},
	})
}

func HistogramHTTPAPIDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "http_api_duration",
		Description: "API request duration in ms",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramHTTPAPIBytesWritten(ctx context.Context, bytes int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, bytes, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "http_api_bytes_written",
		Description: "API response size in bytes",
		Tags:        opts.Tags,
		Unit:        "bytes",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramPauseBlockFlushLatency(ctx context.Context, delay time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, delay.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "pauses_block_flush_duration",
		Description: "Distribution of pauses block flush latency",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramPauseBlockFetchLatency(ctx context.Context, delay time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, delay.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "pauses_block_fetch_duration",
		Description: "Distribution of pauses block fetching latency",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramPauseDeleteLatencyAfterBlockFlush(ctx context.Context, delay time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, delay.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "pauses_delete_after_flush_duration",
		Description: "Distribution of pauses deletion duration after flushing a block",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}

func HistogramSpanFlush(ctx context.Context, delay time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, delay.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "span_flush_duration",
		Description: "Distribution of span flushes from tracing",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  QueueItemLatencyBoundaries,
	})
}

func HistogramPauseBlockCompactionDuration(ctx context.Context, delay time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, delay.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "pauses_block_compaction_duration",
		Description: "Distribution of pause block compaction duration per block",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}

func HistogramConstraintAPIScavengerShardProcessDuration(ctx context.Context, dur time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "constraintapi_scavenger_shard_process_duration",
		Description: "Distribution of scavenger shard processing time duration",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramConstraintAPIScavengerLeaseAge(ctx context.Context, age time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, age.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "constraintapi_scavenger_shard_lease_age",
		Description: "Distribution of scavenger expired lease age",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramCheckpointStartLatency(ctx context.Context, age time.Duration, typ string, opts HistogramOpt) {
	if opts.Tags == nil {
		opts.Tags = map[string]any{}
	}
	opts.Tags["type"] = typ

	RecordIntHistogramMetric(ctx, age.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "checkpoint_start_latency",
		Description: "Distribution of time it took to receive the API request and start processing",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramQueueScavengerPartitionScavengeDuration(ctx context.Context, dur time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "queue_partition_scavenge_duration",
		Description: "Distribution of queue scavenger duration",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramCancellationReadSize(ctx context.Context, value int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, value, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "cancellation_read_size",
		Description: "Distribution of the number of items being read in single operations by the cancellation read writer",
		Tags:        opts.Tags,
		Boundaries:  cancellationReadSizeBoundaries,
	})
}

func HistogramCancellationReadDuration(ctx context.Context, dur time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "cancellation_read_duration",
		Description: "Distribution of cancellation read duration",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  cancellationReadDurationBoundaries,
	})
}

func HistogramExecutorLatency(ctx context.Context, dur time.Duration, typ string, opts HistogramOpt) {
	if opts.Tags == nil {
		opts.Tags = map[string]any{}
	}
	opts.Tags["type"] = typ

	RecordIntHistogramMetric(ctx, dur.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "execution_latency",
		Description: "Distribution of latency within the executor",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  PausesBoundaries,
	})
}

func HistogramCancellationCheckDuration(ctx context.Context, dur time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "cancellation_check_duration",
		Description: "Distribution of cancellation check duration",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  cancellationReadDurationBoundaries,
	})
}

func HistogramScheduleDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "schedule_duration",
		Description: "Distribution of schedule duration with constraint API tracking",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramConstraintAPIRequestStateSize(ctx context.Context, size int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, size, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "constraintapi_request_state_size",
		Description: "Distribution of request state size",
		Tags:        opts.Tags,
		Unit:        "bytes",
		Boundaries:  constraintAPIRequestStateSizeBoundaries,
	})
}

func HistogramConstraintAPILuaScriptDuration(ctx context.Context, duration time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, duration.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "constraintapi_lua_duration",
		Description: "Distribution of Lua script duration",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func HistogramConstraintAPIAcquireRequestLatency(ctx context.Context, latency time.Duration, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, latency.Milliseconds(), HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "constraintapi_acquire_request_latency",
		Description: "Distribution of request latency for Acquire calls",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}
