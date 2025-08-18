package cron

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

func (c *redisCronManager) UpsertSchedule(ctx context.Context, fnID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
