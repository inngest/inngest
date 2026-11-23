package memory

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
)

// Scavenge releases every expired lease through Release with the scavenger
// lease source, so hooks, metrics and forced semaphore release behave as they
// do for the Redis scavenger.  it returns how many leases it reclaimed.
func (m *Manager) Scavenge(ctx context.Context) (int, error) {
	m.hkMu.Lock()
	defer m.hkMu.Unlock()
	return m.scavenge(ctx)
}

// scavenge is the body of Scavenge.  the caller has the housekeeping mutex,
// so Freeze never runs in the middle of a sweep.
func (m *Manager) scavenge(ctx context.Context) (int, error) {
	now := m.clock.Now()
	nowMS := now.UnixMilli()

	var expired, reclaimed int
	var firstErr error
	reclaim := func(seq uint64) {
		sl := m.slab.get(seq)
		if sl == nil {
			return
		}
		rs := sl.req.Load()
		if rs == nil {
			// taken by a release between get and here
			return
		}
		expired++
		leaseID := encodeLeaseID(sl.expiresAtMS, seq, m.nonce)
		accountID := rs.set.accountID

		m.stats.record(metricEvent{kind: histLeaseAge, value: now.Sub(time.UnixMilli(sl.expiresAtMS))})

		_, released, err := m.doRelease(ctx, &constraintapi.CapacityReleaseRequest{
			IdempotencyKey: leaseID.String(),
			AccountID:      accountID,
			LeaseID:        leaseID,
			Source: constraintapi.LeaseSource{
				Service:  constraintapi.ServiceConstraintScavenger,
				Location: constraintapi.CallerLocationLeaseScavenge,
			},
		})
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return
		}
		if released {
			reclaimed++
		}
	}

	// whole seconds behind now hold only expired leases.  the current second
	// is checked lease by lease and left in place.
	m.expiry.drain(nowMS, reclaim)
	m.expiry.scan(nowMS, func(seq uint64) {
		if sl := m.slab.get(seq); sl != nil && sl.expiresAtMS < nowMS {
			reclaim(seq)
		}
	})
	if m.sweepInterval <= 0 {
		m.housekeep(nowMS)
	}

	constraintapi.ScavengeResult{
		TotalExpiredLeasesCount: expired,
		ReclaimedLeases:         reclaimed,
	}.Report(ctx, m.shardName)

	return reclaimed, firstErr
}

// setDropDelay is how long after a constraint set left the sets map its
// RecSetDrop is logged.  a request that loaded the set before it left can
// still log a grant for it, and a follower must have the definition then.
const setDropDelay = 2 * time.Second

// setDrop is a constraint set that left the sets map and whose RecSetDrop is
// not logged yet.
type setDrop struct {
	set         *constraintSet
	deletedAtMS int64
}

// housekeep sweeps the idempotency maps, frees slab pages, drops counters
// that read zero or expired, and drops constraint sets nobody used.  the
// caller has the housekeeping mutex.
func (m *Manager) housekeep(nowMS int64) {
	m.acqIdem.sweep(nowMS)
	m.extIdem.sweep(nowMS)
	m.relIdem.sweep(nowMS)
	m.checkIdem.sweep(nowMS)
	m.semIdem.sweep(nowMS)

	m.slab.freePages()

	m.sems.Range(func(k, v any) bool {
		if v.(*semaphoreCell).kill() {
			m.sems.CompareAndDelete(k, v)
		}
		return true
	})
	// a set nobody used since the last round is dropped.  a request that
	// loaded it meanwhile keeps working on it, and the next request rebuilds.
	l := m.log.Load()
	if l != nil {
		// a drop is logged in the first round that is setDropDelay after the
		// set left the map
		keep := m.setDrops[:0]
		for _, d := range m.setDrops {
			if nowMS-d.deletedAtMS < setDropDelay.Milliseconds() {
				keep = append(keep, d)
				continue
			}
			l.append(uint8(d.set.id%logBuffers), Record{
				Kind:  RecSetDrop,
				SetID: d.set.id,
			})
		}
		m.setDrops = keep
	}
	m.sets.Range(func(k, v any) bool {
		s := v.(*constraintSet)
		if !s.used.Swap(false) && m.sets.CompareAndDelete(k, v) && l != nil {
			m.setDrops = append(m.setDrops, setDrop{
				set:         s,
				deletedAtMS: nowMS,
			})
		}
		return true
	})
	m.gcra.Range(func(k, v any) bool {
		if v.(*gcraCell).kill(nowMS) {
			m.gcra.CompareAndDelete(k, v)
		}
		return true
	})
}

// run is the sweeper goroutine.  every tick reclaims expired leases and
// every housekeepEvery ticks it also runs housekeep.
func (m *Manager) run() {
	defer m.wg.Done()
	ticker := m.clock.NewTicker(m.sweepInterval)
	defer ticker.Stop()

	ctx := context.Background()
	ticks := 0
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.Chan():
		}
		m.hkMu.Lock()
		if _, err := m.scavenge(ctx); err != nil {
			m.logger(ctx).Error("scavenging expired leases failed", "err", err)
		}
		ticks++
		if ticks%housekeepEvery == 0 {
			m.housekeep(m.nowMS())
		}
		m.hkMu.Unlock()
	}
}
