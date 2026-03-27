package lifecycles

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/connect"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
)

type semaphoreLifecycles struct {
	sm constraintapi.SemaphoreManager
}

func NewSemaphoreLifecycleListener(sm constraintapi.SemaphoreManager) connect.ConnectGatewayLifecycleListener {
	return &semaphoreLifecycles{sm: sm}
}

func (s *semaphoreLifecycles) OnConnected(ctx context.Context, conn *state.Connection)          {}
func (s *semaphoreLifecycles) OnReady(ctx context.Context, conn *state.Connection)              {}
func (s *semaphoreLifecycles) OnHeartbeat(ctx context.Context, conn *state.Connection)          {}
func (s *semaphoreLifecycles) OnStartDraining(ctx context.Context, conn *state.Connection)      {}
func (s *semaphoreLifecycles) OnStartDisconnecting(ctx context.Context, conn *state.Connection) {}

// OnSynced is called after a worker group has been synced. At this point, AppID is available.
// We adjust the app semaphore capacity by the worker's max concurrency.
func (s *semaphoreLifecycles) OnSynced(ctx context.Context, conn *state.Connection) {
	if conn.Data == nil {
		return
	}

	maxConcurrency := consts.DefaultWorkerConcurrency
	if conn.Data.MaxWorkerConcurrency != nil && *conn.Data.MaxWorkerConcurrency > 0 {
		maxConcurrency = *conn.Data.MaxWorkerConcurrency
	}

	l := logger.StdlibLogger(ctx)

	for _, group := range conn.Groups {
		if group.AppID == nil {
			continue
		}

		semID := constraintapi.SemaphoreIDApp(*group.AppID)
		idempotencyKey := fmt.Sprintf("connect-%s", conn.ConnectionId)

		_, err := util.WithRetry(ctx, "adjust-semaphore-capacity-connect", func(ctx context.Context) (struct{}, error) {
			return struct{}{}, s.sm.AdjustCapacity(ctx, conn.AccountID, semID, idempotencyKey, maxConcurrency)
		}, util.NewRetryConf())
		if err != nil {
			l.Error("failed to adjust semaphore capacity on worker sync after retries",
				"error", err,
				"app_id", group.AppID,
				"semaphore", semID,
				"delta", maxConcurrency,
				"connection_id", conn.ConnectionId,
			)
		}
	}
}

// OnDisconnected is called when a connection is lost. Decrement the app semaphore capacity.
func (s *semaphoreLifecycles) OnDisconnected(ctx context.Context, conn *state.Connection, closeReason string) {
	if conn.Data == nil {
		return
	}

	maxConcurrency := consts.DefaultWorkerConcurrency
	if conn.Data.MaxWorkerConcurrency != nil && *conn.Data.MaxWorkerConcurrency > 0 {
		maxConcurrency = *conn.Data.MaxWorkerConcurrency
	}

	l := logger.StdlibLogger(ctx)

	for _, group := range conn.Groups {
		if group.AppID == nil {
			continue
		}

		semID := constraintapi.SemaphoreIDApp(*group.AppID)
		// Different idempotency key from connect so both operations can execute
		idempotencyKey := fmt.Sprintf("disconnect-%s", conn.ConnectionId)

		_, err := util.WithRetry(ctx, "adjust-semaphore-capacity-disconnect", func(ctx context.Context) (struct{}, error) {
			return struct{}{}, s.sm.AdjustCapacity(ctx, conn.AccountID, semID, idempotencyKey, -maxConcurrency)
		}, util.NewRetryConf())
		if err != nil {
			l.Error("failed to adjust semaphore capacity on worker disconnect after retries",
				"error", err,
				"app_id", group.AppID,
				"semaphore", semID,
				"delta", -maxConcurrency,
				"connection_id", conn.ConnectionId,
				"close_reason", closeReason,
			)
		}
	}
}
