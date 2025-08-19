package cron

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
)

func NewRedisCronManager() CronManager {
	manager := &redisCronManager{}

	return manager
}

type redisCronManager struct {
	q redis_state.QueueManager

	log logger.Logger
}

func (c *redisCronManager) ScheduleNext(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (c *redisCronManager) UpdateSchedule(ctx context.Context, ci CronItem) error {
	switch ci.Op {
	case CronOpNew, CronOpUnpause:
		// TODO
		// - enqueue new schedule
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
