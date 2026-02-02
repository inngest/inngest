package metrics

import "context"

func IncrRealtimeHTTPRequestsTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_http_requests_total",
		Description: "Total number of realtime HTTP requests",
		Tags:        opts.Tags,
	})
}

func HistogramRealtimeHTTPRequestDuration(ctx context.Context, dur int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, dur, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_http_request_duration_ms",
		Description: "Duration of realtime HTTP requests in ms",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  DefaultBoundaries,
	})
}

func IncrRealtimeAuthFailuresTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_auth_failures_total",
		Description: "Total number of realtime authentication failures",
		Tags:        opts.Tags,
	})
}

func GaugeRealtimeConnectionsActive(ctx context.Context, val int64, opts GaugeOpt) {
	RecordGaugeMetric(ctx, val, GaugeOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_connections_active",
		Description: "Number of active realtime connections",
		Tags:        opts.Tags,
	})
}

func IncrRealtimeKeepaliveFailuresTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_keepalive_failures_total",
		Description: "Total number of realtime keepalive failures",
		Tags:        opts.Tags,
	})
}

func IncrRealtimeMessagesPublishedTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_messages_published_total",
		Description: "Total number of realtime messages published",
		Tags:        opts.Tags,
	})
}

func HistogramRealtimePublishFanoutSize(ctx context.Context, val int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, val, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_publish_fanout_size",
		Description: "Number of subscribers a message is published to",
		Tags:        opts.Tags,
		Boundaries:  peekSizeBoundaries,
	})
}

func HistogramRealtimePayloadSizeBytes(ctx context.Context, val int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, val, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_payload_size_bytes",
		Description: "Size of realtime payloads in bytes",
		Tags:        opts.Tags,
		Unit:        "bytes",
		Boundaries:  DefaultBoundaries,
	})
}

func IncrRealtimeRedisOpsTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_redis_ops_total",
		Description: "Total number of realtime Redis operations",
		Tags:        opts.Tags,
	})
}

func IncrRealtimeRedisErrorsTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_redis_errors_total",
		Description: "Total number of realtime Redis errors",
		Tags:        opts.Tags,
	})
}

func HistogramRealtimeRawDataSizeBytes(ctx context.Context, val int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, val, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_raw_data_size_bytes",
		Description: "Size of raw data payload in bytes",
		Tags:        opts.Tags,
		Unit:        "bytes",
		Boundaries:  DefaultBoundaries,
	})
}

func IncrRealtimeDisconnectionsTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_disconnections_total",
		Description: "Total number of realtime disconnections",
		Tags:        opts.Tags,
	})
}

func IncrRealtimeRedisMessageTypesTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_redis_message_types_total",
		Description: "Total number of Redis messages by type",
		Tags:        opts.Tags,
	})
}

func IncrRealtimeMessageDeliveryFailuresTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_message_delivery_failures_total",
		Description: "Total number of message delivery failures",
		Tags:        opts.Tags,
	})
}

func HistogramRealtimeConnectionDuration(ctx context.Context, val int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, val, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_connection_duration_ms",
		Description: "Duration of realtime connections in ms",
		Tags:        opts.Tags,
		Unit:        "ms",
		Boundaries:  QueueItemLatencyBoundaries,
	})
}

func IncrRealtimeJWTTokensCreatedTotal(ctx context.Context, opts CounterOpt) {
	RecordCounterMetric(ctx, 1, CounterOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_jwt_tokens_created_total",
		Description: "Total number of JWT tokens created",
		Tags:        opts.Tags,
	})
}

func HistogramRealtimeSubscriptionTopicsCount(ctx context.Context, val int64, opts HistogramOpt) {
	RecordIntHistogramMetric(ctx, val, HistogramOpt{
		PkgName:     opts.PkgName,
		MetricName:  "realtime_subscription_topics_count",
		Description: "Number of topics per subscription",
		Tags:        opts.Tags,
		Boundaries:  peekSizeBoundaries,
	})
}
