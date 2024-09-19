package batch

import (
	"context"
	"errors"
	"fmt"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"sync"
	"time"
)

const pkgName = "execution.batch"

func InstrumentBatching(ctx context.Context, q redis_state.QueueManager, b *redis_state.BatchClient) error {
	log := logger.From(context.Background())
	configLeaser, ok := q.(redis_state.ConfigLeaser)
	if !ok {
		return fmt.Errorf("expected config leaser to be passed in")
	}

	batchManager := NewRedisBatchManager(b, q)

	var currentLease func() *ulid.ULID
	var setLease func(leaseId *ulid.ULID)

	{
		leaseLock := sync.RWMutex{}
		var existingLeaseId *ulid.ULID

		currentLease = func() *ulid.ULID {
			leaseLock.RLock()
			defer leaseLock.RUnlock()
			if existingLeaseId == nil {
				return nil
			}
			copied := *existingLeaseId
			return &copied
		}

		setLease = func(leaseId *ulid.ULID) {
			leaseLock.Lock()
			defer leaseLock.Unlock()
			copied := *leaseId
			existingLeaseId = &copied
		}
	}

	{
		leaseID, err := configLeaser.ConfigLease(ctx, b.KeyGenerator().BatchInstrument(), redis_state.ConfigLeaseDuration, currentLease())
		if err != redis_state.ErrConfigAlreadyLeased && err != nil {
			return fmt.Errorf("could not lease instrument")
		}

		setLease(leaseID)
	}

	batchInstrumentTicker := time.NewTicker(10 * time.Second)
	leaseTick := time.NewTicker(redis_state.ConfigLeaseMax / 3)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-leaseTick.C:
			leaseID, err := configLeaser.ConfigLease(ctx, b.KeyGenerator().BatchInstrument(), redis_state.ConfigLeaseDuration, currentLease())
			if errors.Is(err, redis_state.ErrConfigAlreadyLeased) {
				setLease(nil)
				continue
			}

			if err != nil {
				log.Error().Err(err).Msg("error claiming instrumentation lease")
				setLease(nil)
				continue
			}

			setLease(leaseID)
			continue
		case <-batchInstrumentTicker.C:
		}

		pendingBatchCount, err := batchManager.PendingBatchCount(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error retrieving pending batches")
			continue
		}

		for accountId, count := range pendingBatchCount {
			metrics.HistogramPendingEventBatches(ctx, count, metrics.HistogramOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"account_id": accountId.String(),
				},
			})
		}
	}
}
