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
		accountID := rs.accountID

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

	constraintapi.ScavengeResult{
		TotalExpiredLeasesCount: expired,
		ReclaimedLeases:         reclaimed,
	}.Report(ctx, m.shardName)

	return reclaimed, firstErr
}

// housekeep sweeps the idempotency maps, frees slab pages and drops counters
// that read zero or expired.
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
		if _, err := m.Scavenge(ctx); err != nil {
			m.log(ctx).Error("scavenging expired leases failed", "err", err)
		}
		ticks++
		if ticks%housekeepEvery == 0 {
			m.housekeep(m.nowMS())
		}
	}
}
