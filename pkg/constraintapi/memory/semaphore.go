package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

// semResult is what one semaphore write left in its flight.  applied is
// false on a replay.
type semResult struct {
	value   int64
	applied bool
}

func semaphoreCapacityKey(accountID uuid.UUID, name string) string {
	return (&constraintapi.SemaphoreConstraint{ID: name}).CapacityKey(accountID)
}

func semaphoreUsageKey(accountID uuid.UUID, name, evaluatedKeyHash string) string {
	return (&constraintapi.SemaphoreConstraint{ID: name, EvaluatedKeyHash: evaluatedKeyHash}).UsageKey(accountID)
}

// semWrite runs fn once per idempotency key within the semaphore idempotency
// window and returns the value fn produced the first time.
func (m *Manager) semWrite(accountID uuid.UUID, op, idempotencyKey string, fn func() int64) semResult {
	nowMS := m.nowMS()
	key := hashKey(accountID, op, idempotencyKey)
	executed := false
	v, _, _ := m.flight.Do(flightKey('s', key), func() (any, error) {
		executed = true
		if cached, ok := m.semIdem.get(nowMS, key); ok {
			return semResult{value: cached, applied: false}, nil
		}
		value := fn()
		m.semIdem.set(key, value, idemExpiry(nowMS, constraintapi.SemaphoreIdempotencyTTL))
		return semResult{value: value, applied: true}, nil
	})
	res := v.(semResult)
	if !executed {
		res.applied = false
	}
	return res
}

// SetCapacity implements constraintapi.SemaphoreManager.
func (m *Manager) SetCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, capacity int64) (constraintapi.SetResult, error) {
	start := time.Now()
	capKey := semaphoreCapacityKey(accountID, name)
	res := m.semWrite(accountID, "setcap", idempotencyKey, func() int64 {
		for cell := m.sem(capKey); !cell.set(capacity); cell = m.sem(capKey) {
		}
		return capacity
	})
	m.recordMetrics(ctx, "set_capacity", start)
	return constraintapi.SetResult{Applied: res.applied, Capacity: res.value}, nil
}

// AdjustCapacity implements constraintapi.SemaphoreManager.  capacity never
// goes below zero.
func (m *Manager) AdjustCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, delta int64) (constraintapi.AdjustResult, error) {
	start := time.Now()
	capKey := semaphoreCapacityKey(accountID, name)
	res := m.semWrite(accountID, "adjcap", idempotencyKey, func() int64 {
		for {
			if v, ok := m.sem(capKey).adjust(delta); ok {
				return v
			}
		}
	})
	m.recordMetrics(ctx, "adjust_capacity", start)
	return constraintapi.AdjustResult{Applied: res.applied, Capacity: res.value}, nil
}

// GetCapacity implements constraintapi.SemaphoreManager.  an unknown
// semaphore reads as capacity 0 and usage 0.
func (m *Manager) GetCapacity(ctx context.Context, accountID uuid.UUID, name, evaluatedKeyHash string) (capacity int64, usage int64, err error) {
	start := time.Now()
	capacity = m.peekSem(semaphoreCapacityKey(accountID, name))
	usage = m.peekSem(semaphoreUsageKey(accountID, name, evaluatedKeyHash))
	m.recordMetrics(ctx, "get_capacity", start)
	return capacity, usage, nil
}

// ReleaseSemaphore implements constraintapi.SemaphoreManager.  usage never
// goes below zero.
func (m *Manager) ReleaseSemaphore(ctx context.Context, accountID uuid.UUID, name, evaluatedKeyHash, idempotencyKey string, weight int64) error {
	start := time.Now()
	usageKey := semaphoreUsageKey(accountID, name, evaluatedKeyHash)
	m.semWrite(accountID, "rel", idempotencyKey, func() int64 {
		for {
			if v, ok := m.sem(usageKey).give(weight); ok {
				return v
			}
		}
	})
	m.recordMetrics(ctx, "release", start)
	return nil
}

func (m *Manager) recordMetrics(ctx context.Context, operation string, start time.Time) {
	tags := map[string]any{
		"operation": operation,
		"status":    "success",
	}
	metrics.IncrConstraintAPISemaphoreCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: tags})
	metrics.HistogramConstraintAPISemaphoreDuration(ctx, time.Since(start), metrics.HistogramOpt{PkgName: pkgName, Tags: tags})
}
