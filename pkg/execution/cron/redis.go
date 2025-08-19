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
	return fmt.Errorf("not implemented")
}
