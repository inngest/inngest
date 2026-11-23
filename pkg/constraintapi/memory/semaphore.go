package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
)

// semResult is what one semaphore write produced.  applied is false on a
// replay.
type semResult struct {
	value   int64
	applied bool
}

// semWrite runs fn once per idempotency key within the semaphore idempotency
// window and returns the value fn produced the first time.  fn returns the
// value and the record of what it changed.  cell is the counter fn writes;
// its records go to the buffer picked by the cell so two writes to one
// counter reach a follower in the order they were made.  capacity makes the
// write and its log append one step under the capacity mutex, because
// SetCapacity and AdjustCapacity log the resulting value and two of them can
// run for one counter at once.  the caller is answered when every in-sync
// replica has the write.
func (m *Manager) semWrite(ctx context.Context, accountID uuid.UUID, op, idempotencyKey string, cell cellKey, capacity bool, fn func() (int64, Record)) (semResult, error) {
	nowMS := m.nowMS()
	key := opKey(accountID, op, idempotencyKey)

	var batch *Batch
	mu := m.lock(key)
	value, hit := m.semIdem.get(nowMS, key)
	if !hit {
		expiresAtMS := idemExpiry(nowMS, constraintapi.SemaphoreIdempotencyTTL)
		if capacity {
			m.capMu.Lock()
		}
		var rec Record
		value, rec = fn()
		m.semIdem.set(key, value, expiresAtMS)
		if l := m.log.Load(); l != nil {
			batch = l.append(uint8(cell.a%logBuffers), rec, Record{
				Kind:        RecIdem,
				KeyA:        key,
				Value:       value,
				ExpiresAtMS: expiresAtMS,
			})
		}
		if capacity {
			m.capMu.Unlock()
		}
	}
	mu.Unlock()

	if hit {
		batch = m.replayBatch()
	}
	if ierr := m.awaitCommit(ctx, batch, constraintapi.CommitLevelAll, constraintapi.CommitLevelAll); ierr != nil {
		return semResult{}, ierr
	}
	return semResult{
		value:   value,
		applied: !hit,
	}, nil
}

// SetCapacity implements constraintapi.SemaphoreManager.
func (m *Manager) SetCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, capacity int64) (constraintapi.SetResult, error) {
	start := time.Now()
	_, capKey := semaphoreCells(accountID, name, "")
	res, err := m.semWrite(ctx, accountID, "setcap", idempotencyKey, capKey, true, func() (int64, Record) {
		for cell := m.sem(capKey); !cell.set(capacity); cell = m.sem(capKey) {
		}
		return capacity, Record{
			Kind:  RecCell,
			KeyA:  capKey.a,
			KeyB:  capKey.b,
			Value: capacity,
		}
	})
	m.stats.semaphoreOp("set_capacity", start)
	if err != nil {
		return constraintapi.SetResult{}, err
	}
	return constraintapi.SetResult{Applied: res.applied, Capacity: res.value}, nil
}

// AdjustCapacity implements constraintapi.SemaphoreManager.  capacity never
// goes below zero.
func (m *Manager) AdjustCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, delta int64) (constraintapi.AdjustResult, error) {
	start := time.Now()
	_, capKey := semaphoreCells(accountID, name, "")
	res, err := m.semWrite(ctx, accountID, "adjcap", idempotencyKey, capKey, true, func() (int64, Record) {
		for {
			if v, ok := m.sem(capKey).adjust(delta); ok {
				return v, Record{
					Kind:  RecCell,
					KeyA:  capKey.a,
					KeyB:  capKey.b,
					Value: v,
				}
			}
		}
	})
	m.stats.semaphoreOp("adjust_capacity", start)
	if err != nil {
		return constraintapi.AdjustResult{}, err
	}
	return constraintapi.AdjustResult{Applied: res.applied, Capacity: res.value}, nil
}

// GetCapacity implements constraintapi.SemaphoreManager.  an unknown
// semaphore reads as capacity 0 and usage 0.
func (m *Manager) GetCapacity(ctx context.Context, accountID uuid.UUID, name, evaluatedKeyHash string) (capacity int64, usage int64, err error) {
	start := time.Now()
	usageKey, capKey := semaphoreCells(accountID, name, evaluatedKeyHash)
	capacity = m.peekSem(capKey)
	usage = m.peekSem(usageKey)
	m.stats.semaphoreOp("get_capacity", start)
	return capacity, usage, nil
}

// ReleaseSemaphore implements constraintapi.SemaphoreManager.  usage never
// goes below zero.
func (m *Manager) ReleaseSemaphore(ctx context.Context, accountID uuid.UUID, name, evaluatedKeyHash, idempotencyKey string, weight int64) error {
	start := time.Now()
	usageKey, _ := semaphoreCells(accountID, name, evaluatedKeyHash)
	_, err := m.semWrite(ctx, accountID, "rel", idempotencyKey, usageKey, false, func() (int64, Record) {
		for {
			if v, applied, ok := m.sem(usageKey).giveApplied(weight); ok {
				return v, Record{
					Kind:  RecDelta,
					KeyA:  usageKey.a,
					KeyB:  usageKey.b,
					Value: -applied,
				}
			}
		}
	})
	m.stats.semaphoreOp("release", start)
	return err
}
