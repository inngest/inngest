package lifecycles

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/constraintapi"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type semaphoreAdjustCall struct {
	accountID      uuid.UUID
	name           string
	idempotencyKey string
	delta          int64
}

type recordingSemaphoreManager struct {
	mu    sync.Mutex
	calls []semaphoreAdjustCall
}

func (m *recordingSemaphoreManager) SetCapacity(context.Context, uuid.UUID, string, string, int64) (constraintapi.SetResult, error) {
	panic("not implemented")
}

func (m *recordingSemaphoreManager) AdjustCapacity(_ context.Context, accountID uuid.UUID, name, idempotencyKey string, delta int64) (constraintapi.AdjustResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, semaphoreAdjustCall{
		accountID:      accountID,
		name:           name,
		idempotencyKey: idempotencyKey,
		delta:          delta,
	})

	return constraintapi.AdjustResult{Applied: true, Capacity: delta}, nil
}

func (m *recordingSemaphoreManager) GetCapacity(context.Context, uuid.UUID, string, string) (int64, int64, error) {
	panic("not implemented")
}

func (m *recordingSemaphoreManager) ReleaseSemaphore(context.Context, uuid.UUID, string, string, string, int64) error {
	panic("not implemented")
}

func (m *recordingSemaphoreManager) recordedCalls() []semaphoreAdjustCall {
	m.mu.Lock()
	defer m.mu.Unlock()

	calls := make([]semaphoreAdjustCall, len(m.calls))
	copy(calls, m.calls)
	return calls
}

func (m *recordingSemaphoreManager) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = nil
}

func TestSemaphoreLifecycleUsesUniqueIdempotencyKeysPerApp(t *testing.T) {
	ctx := context.Background()
	accountID := uuid.New()
	appID1 := uuid.New()
	appID2 := uuid.New()
	connID := ulid.Make()
	maxConcurrency := int64(7)

	conn := &state.Connection{
		AccountID:    accountID,
		ConnectionId: connID,
		Data: &connectpb.WorkerConnectRequestData{
			MaxWorkerConcurrency: &maxConcurrency,
		},
		Groups: map[string]*state.WorkerGroup{
			"group-1": {AppID: &appID1},
			"group-2": {AppID: &appID2},
		},
	}

	sm := &recordingSemaphoreManager{}
	lifecycle := NewSemaphoreLifecycleListener(sm, time.Time{})

	lifecycle.OnSynced(ctx, conn)

	connectCalls := sm.recordedCalls()
	require.Len(t, connectCalls, 2)
	require.ElementsMatch(t, []string{
		constraintapi.SemaphoreIDApp(appID1),
		constraintapi.SemaphoreIDApp(appID2),
	}, []string{connectCalls[0].name, connectCalls[1].name})

	connectKeys := map[string]struct{}{}
	for _, call := range connectCalls {
		require.Equal(t, accountID, call.accountID)
		require.Equal(t, maxConcurrency, call.delta)
		require.Contains(t, call.idempotencyKey, "connect-"+connID.String()+"-")
		require.Contains(t, call.idempotencyKey, call.name)
		connectKeys[call.idempotencyKey] = struct{}{}
	}
	require.Len(t, connectKeys, 2)

	sm.reset()
	lifecycle.OnDisconnected(ctx, conn, "test close")

	disconnectCalls := sm.recordedCalls()
	require.Len(t, disconnectCalls, 2)
	require.ElementsMatch(t, []string{
		constraintapi.SemaphoreIDApp(appID1),
		constraintapi.SemaphoreIDApp(appID2),
	}, []string{disconnectCalls[0].name, disconnectCalls[1].name})

	disconnectKeys := map[string]struct{}{}
	for _, call := range disconnectCalls {
		require.Equal(t, accountID, call.accountID)
		require.Equal(t, -maxConcurrency, call.delta)
		require.Contains(t, call.idempotencyKey, "disconnect-"+connID.String()+"-")
		require.Contains(t, call.idempotencyKey, call.name)
		require.NotContains(t, connectKeys, call.idempotencyKey)
		disconnectKeys[call.idempotencyKey] = struct{}{}
	}
	require.Len(t, disconnectKeys, 2)
}
