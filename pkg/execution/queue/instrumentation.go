package queue

import (
	"context"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
)

func NewInstrumentationRole(opts ...QueueRoleOpt) QueueRole {
	return newQueueRole(QueueRoleInstrumentation, RoleLeaseMax, DefaultInstrumentInterval, func(ctx context.Context, shard QueueShard) error {
		shardAssignmentConfig := shard.ShardAssignmentConfig()
		metrics.GaugeShardLeaseCapacity(ctx, int64(shardAssignmentConfig.NumExecutors), metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": shardAssignmentConfig.ShardGroup, "queue_shard": shard.Name()}})

		ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Instrument"), redis_telemetry.ScopeQueue)
		return shard.Instrument(ctx)
	}, opts...)
}
