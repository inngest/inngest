package queue

import (
	"context"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
)

func NewInstrumentationRole(q *queueProcessor, opts ...QueueRoleOpt) QueueRole {
	return newQueueRole(QueueRoleInstrumentation, RoleLeaseMax, DefaultInstrumentInterval, func(ctx context.Context, shard QueueShard) error {
		ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Instrument"), redis_telemetry.ScopeQueue)
		return shard.Instrument(ctx)
	}, func(ctx context.Context, shard QueueShard) {
		metrics.GaugeWorkerQueueCapacity(ctx, int64(q.numWorkers), metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name()}})
		metrics.GaugePartitionProcessorCapacity(ctx, q.partitionCapacity(), metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name()}})
		metrics.GaugePartitionProcessorInFlight(ctx, q.partitionSem.Count(), metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": shard.Name()}})

		shardAssignmentConfig := shard.ShardAssignmentConfig()
		metrics.GaugeShardLeaseCapacity(ctx, int64(shardAssignmentConfig.NumExecutors), metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": shardAssignmentConfig.ShardGroup, "queue_shard": shard.Name(), "segment": q.ShardLeaseKeySuffix}})
	}, opts...)
}
