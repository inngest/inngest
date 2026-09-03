package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// testStart is the fake clock start shared by every backend so lease expiries
// and retry times compare equal across them.
var testStart = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// newMemoryManager builds a manager with the background sweeper disabled so
// tests own the moment expired leases are reclaimed.
func newMemoryManager(t testing.TB, clock clockwork.Clock, opts ...memory.Option) *memory.Manager {
	t.Helper()
	all := append([]memory.Option{
		memory.WithShardName("memory"),
		memory.WithClock(clock),
		memory.WithSweepInterval(0),
	}, opts...)
	m, err := memory.NewManager(all...)
	require.NoError(t, err)
	require.NotNil(t, m)
	t.Cleanup(func() { _ = m.Close() })
	return m
}

func newMiniredisClient(t testing.TB) (*miniredis.Miniredis, rueidis.Client) {
	t.Helper()
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	return r, rc
}

type ids struct {
	account, env, fn, app uuid.UUID
}

func newIDs() ids {
	return ids{account: uuid.New(), env: uuid.New(), fn: uuid.New(), app: uuid.New()}
}

func semaphoreItem(sem *constraintapi.SemaphoreConstraint) constraintapi.ConstraintItem {
	return constraintapi.ConstraintItem{Kind: constraintapi.ConstraintKindSemaphore, Semaphore: sem}
}

func semaphoreConfig(sems ...constraintapi.SemaphoreConstraint) constraintapi.ConstraintConfig {
	cfg := constraintapi.ConstraintConfig{FunctionVersion: 1}
	for _, s := range sems {
		cfg.Semaphores = append(cfg.Semaphores, constraintapi.Semaphore(s))
	}
	return cfg
}

// acquireRequest builds a one lease request in the shape the executor sends.
func acquireRequest(id ids, clock clockwork.Clock, config constraintapi.ConstraintConfig, constraints []constraintapi.ConstraintItem, idempotencyKey string) *constraintapi.CapacityAcquireRequest {
	return &constraintapi.CapacityAcquireRequest{
		AccountID:            id.account,
		EnvID:                id.env,
		FunctionID:           id.fn,
		AppID:                id.app,
		IdempotencyKey:       idempotencyKey,
		Constraints:          constraints,
		Amount:               1,
		Configuration:        config,
		LeaseIdempotencyKeys: []string{idempotencyKey + "-lease"},
		LeaseRunIDs:          map[string]ulid.ULID{},
		CurrentTime:          clock.Now(),
		Duration:             5 * time.Second,
		MaximumLifetime:      time.Hour,
		Source: constraintapi.LeaseSource{
			Service:           constraintapi.ServiceExecutor,
			Location:          constraintapi.CallerLocationItemLease,
			RunProcessingMode: constraintapi.RunProcessingModeBackground,
		},
	}
}

// withAmount turns a one lease request into an n lease request.
func withAmount(req *constraintapi.CapacityAcquireRequest, n int) *constraintapi.CapacityAcquireRequest {
	req.Amount = n
	req.LeaseIdempotencyKeys = make([]string, n)
	for i := range req.LeaseIdempotencyKeys {
		req.LeaseIdempotencyKeys[i] = req.IdempotencyKey + "-lease-" + string(rune('a'+i))
	}
	return req
}

func acquire(t *testing.T, cm constraintapi.CapacityManager, req *constraintapi.CapacityAcquireRequest) *constraintapi.CapacityAcquireResponse {
	t.Helper()
	resp, err := cm.Acquire(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	return resp
}

func release(t *testing.T, cm constraintapi.CapacityManager, accountID uuid.UUID, leaseID ulid.ULID, key string) *constraintapi.CapacityReleaseResponse {
	t.Helper()
	resp, err := cm.Release(context.Background(), &constraintapi.CapacityReleaseRequest{
		IdempotencyKey: key,
		AccountID:      accountID,
		LeaseID:        leaseID,
		Source: constraintapi.LeaseSource{
			Service:  constraintapi.ServiceExecutor,
			Location: constraintapi.CallerLocationItemLease,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	return resp
}

func extend(t *testing.T, cm constraintapi.CapacityManager, accountID uuid.UUID, leaseID ulid.ULID, key string, d time.Duration) *constraintapi.CapacityExtendLeaseResponse {
	t.Helper()
	resp, err := cm.ExtendLease(context.Background(), &constraintapi.CapacityExtendLeaseRequest{
		IdempotencyKey: key,
		AccountID:      accountID,
		LeaseID:        leaseID,
		Duration:       d,
		Source: constraintapi.LeaseSource{
			Service:  constraintapi.ServiceExecutor,
			Location: constraintapi.CallerLocationItemLease,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	return resp
}

func usageOf(t *testing.T, sm constraintapi.SemaphoreManager, accountID uuid.UUID, sem constraintapi.SemaphoreConstraint) int64 {
	t.Helper()
	_, usage, err := sm.GetCapacity(context.Background(), accountID, sem.ID, sem.EvaluatedKeyHash)
	require.NoError(t, err)
	return usage
}

func setCapacity(t *testing.T, sm constraintapi.SemaphoreManager, accountID uuid.UUID, name string, capacity int64) {
	t.Helper()
	_, err := sm.SetCapacity(context.Background(), accountID, name, "setcap-"+name+"-"+uuid.NewString(), capacity)
	require.NoError(t, err)
}
