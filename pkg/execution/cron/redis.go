package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
)

func NewRedisCronManager(
	c *redis_state.CronClient,
	q redis_state.QueueManager,
	log logger.Logger,
) CronManager {
	manager := &redisCronManager{
		c:   c,
		q:   q,
		log: log,
	}

	return manager
}

type redisCronManager struct {
	c *redis_state.CronClient
	q redis_state.QueueManager

	log logger.Logger
}

func (c *redisCronManager) ScheduleNext(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (c *redisCronManager) UpdateSchedule(ctx context.Context, ci CronItem) error {
	l := c.log.With("action", "redisCronManager.UpdateSchedule", "cron_item", ci)

	switch ci.Op {
	case CronOpNew, CronOpUnpause:
		jobID := ci.ID.String()
		queueName := queue.KindCron
		maxAttempts := consts.MaxRetries + 1

		// enqueue new schedule
		err := c.q.Enqueue(ctx, queue.Item{
			JobID:       &jobID,
			GroupID:     uuid.New().String(),
			WorkspaceID: ci.WorkspaceID,
			Identifier: state.Identifier{
				AccountID:       ci.AccountID,
				WorkspaceID:     ci.WorkspaceID,
				AppID:           ci.AppID,
				WorkflowID:      ci.FunctionID,
				WorkflowVersion: ci.FunctionVersion,
			},
			QueueName:   &queueName,
			MaxAttempts: &maxAttempts,
		}, time.Now(), queue.EnqueueOpts{})
		switch err {
		case nil, redis_state.ErrQueueItemExists, redis_state.ErrQueueItemSingletonExists:
			// no-op
		default:
			l.ReportError(err, "error enqueueing cron for next schedule")
			return fmt.Errorf("error enqueueing cron for next schedule: %w", err)
		}

		// TODO
		// - update mapping

	case CronOpUpdate:
		// TODO
		// - delete and dequeue existing queue item
		// - enqueue new schedule
		// - update mapping

	case CronOpPause:
		// TODO
		// - delete and dequeue existing queue item

	default:
		return fmt.Errorf("unknow cron operation: %d", ci.Op)
	}

	return fmt.Errorf("not implemented")
}
