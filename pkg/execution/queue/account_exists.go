package queue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

const (
	deletedAccountPartitionActionFound           = "found"
	deletedAccountPartitionActionQuarantined     = "quarantined"
	deletedAccountPartitionActionCheckError      = "check_error"
	deletedAccountPartitionActionQuarantineError = "quarantine_error"
)

func (q *queueProcessor) accountExists(ctx context.Context, accountID uuid.UUID) (bool, error) {
	if accountID == uuid.Nil {
		return true, nil
	}

	if q.AccountExists == nil {
		return true, nil
	}

	return q.AccountExists(ctx, accountID)
}

func (q *queueProcessor) requeueDeletedAccountPartition(ctx context.Context, shard QueueShard, p *QueuePartition) error {
	requeueAt := q.Clock().Now().Add(PartitionDeletedAccountRequeueExtension)
	if err := shard.PartitionRequeue(ctx, p, requeueAt, true); err != nil {
		metrics.IncrQueueDeletedAccountPartitionCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard": shard.Name(),
				"action":      deletedAccountPartitionActionQuarantineError,
			},
		})
		return fmt.Errorf("error requeueing deleted account partition: %w", err)
	}

	metrics.IncrQueueDeletedAccountPartitionCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": shard.Name(),
			"action":      deletedAccountPartitionActionQuarantined,
		},
	})

	logger.StdlibLogger(ctx).Warn(
		"requeued partition for deleted account",
		"account_id", p.AccountID.String(),
		"partition_id", p.ID,
		"queue_shard", shard.Name(),
		"requeue_at", requeueAt,
	)

	return nil
}
