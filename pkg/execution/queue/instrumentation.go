package queue

import (
	"context"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
)

func (q *queueProcessor) runInstrumentation(ctx context.Context) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Instrument"), redis_telemetry.ScopeQueue)

	leaseID, err := q.PrimaryQueueShard.ConfigLease(ctx, "instrument", ConfigLeaseMax, q.instrumentationLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	setLease := func(lease *ulid.ULID) {
		q.instrumentationLeaseLock.Lock()
		defer q.instrumentationLeaseLock.Unlock()
		q.instrumentationLeaseID = lease

		if lease != nil && q.instrumentationLeaseID == nil {
			metrics.IncrInstrumentationLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.PrimaryQueueShard.Name()}})
		}
	}

	setLease(leaseID)

	tick := q.Clock.NewTicker(ConfigLeaseMax / 3)
	instr := q.Clock.NewTicker(q.instrumentInterval)

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			instr.Stop()
			return
		case <-instr.Chan():
			if q.isInstrumentator() {
				if err := q.PrimaryQueueShard.Instrument(ctx); err != nil {
					q.log.Error("error running instrumentation", "error", err)
				}
			}
		case <-tick.Chan():
			metrics.GaugeWorkerQueueCapacity(ctx, int64(q.numWorkers), metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.PrimaryQueueShard.Name()}})

			leaseID, err := q.PrimaryQueueShard.ConfigLease(ctx, "instrument", ConfigLeaseMax, q.instrumentationLease())
			if err == ErrConfigAlreadyLeased {
				setLease(nil)
				continue
			}

			if err != nil {
				q.log.Error("error claiming instrumentation lease", "error", err)
				setLease(nil)
				continue
			}

			setLease(leaseID)
		}
	}
}
