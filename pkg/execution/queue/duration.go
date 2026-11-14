package queue

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

// duration is a helper function to record durations of queue operations.
func Duration[T any](ctx context.Context, queueShardName string, op string, start time.Time, f func(ctx context.Context) (T, error)) (T, error) {
	return DurationWithTags(ctx, queueShardName, op, start, f, nil)
}

// durationWithTags is a helper function to record durations of queue operations.
func DurationWithTags[T any](ctx context.Context, queueShardName string, op string, start time.Time, f func(ctx context.Context) (T, error), tags map[string]any, slowLogFields ...func() []any) (T, error) {
	if start.IsZero() {
		start = time.Now()
	}

	finalTags := map[string]any{
		"operation":   op,
		"queue_shard": queueShardName,
	}
	for k, v := range tags {
		finalTags[k] = v
	}

	res, err := f(ctx)

	d := time.Since(start)
	if d > time.Second {
		fields := []any{"op", op, "duration", d}
		for _, getFields := range slowLogFields {
			fields = append(fields, getFields()...)
		}
		logger.StdlibLogger(ctx).Warn("queue operation took >1s", fields...)
	}
	metrics.HistogramQueueOperationDuration(
		ctx,
		d.Milliseconds(),
		metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    finalTags,
		},
	)
	return res, err
}
