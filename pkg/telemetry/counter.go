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
