package batch

import (
	"context"
	"errors"
	"fmt"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
	"sync"
	"time"
)

const pkgName = "execution.batch"

type batchInstrumenter struct {
	queue        queue.Queue
	batchClient  *redis_state.BatchClient
	configLeaser redis_state.ConfigLeaser
	logger       *zerolog.Logger
	batchManager BatchManager

	leaseLock       *sync.RWMutex
	existingLeaseId *ulid.ULID
}

func (b *batchInstrumenter) Name() string {
	return "batch-instrumenter"
}

func (b *batchInstrumenter) Pre(ctx context.Context) error {
	configLeaser, ok := b.queue.(redis_state.ConfigLeaser)
	if !ok {
		return fmt.Errorf("expected config leaser to be passed in")
	}
	b.configLeaser = configLeaser
	return nil
}

func (b *batchInstrumenter) currentLease() *ulid.ULID {
	b.leaseLock.RLock()
	defer b.leaseLock.RUnlock()
	if b.existingLeaseId == nil {
		return nil
	}
	copied := *b.existingLeaseId
	return &copied
}

func (b *batchInstrumenter) setLease(leaseId *ulid.ULID) {
	b.leaseLock.Lock()
	defer b.leaseLock.Unlock()
	copied := *leaseId
	b.existingLeaseId = &copied
}

func (b *batchInstrumenter) Run(ctx context.Context) error {
	{
		leaseID, err := b.configLeaser.ConfigLease(ctx, b.batchClient.KeyGenerator().BatchInstrument(), redis_state.ConfigLeaseDuration, b.currentLease())
		if err != redis_state.ErrConfigAlreadyLeased && err != nil {
			return fmt.Errorf("could not lease instrument: %w", err)
		}

		b.setLease(leaseID)
	}

	batchInstrumentTicker := time.NewTicker(10 * time.Second)
	leaseTick := time.NewTicker(redis_state.ConfigLeaseMax / 3)

	go func() {
		for {
			select {
			case <-ctx.Done():
				batchInstrumentTicker.Stop()
				leaseTick.Stop()
				return
			case <-leaseTick.C:
				leaseID, err := b.configLeaser.ConfigLease(ctx, b.batchClient.KeyGenerator().BatchInstrument(), redis_state.ConfigLeaseDuration, b.currentLease())
				if errors.Is(err, redis_state.ErrConfigAlreadyLeased) {
					b.setLease(nil)
					continue
				}

				if err != nil {
					b.logger.Error().Err(err).Msg("error claiming instrumentation lease")
					b.setLease(nil)
					continue
				}

				b.setLease(leaseID)
				continue
			case <-batchInstrumentTicker.C:
			}

			pendingBatchCount, err := b.batchManager.PendingBatchCount(ctx)
			if err != nil {
				b.logger.Error().Err(err).Msg("error retrieving pending batches")
				continue
			}

			for accountId, count := range pendingBatchCount {
				b.logger.Trace().Str("account_id", accountId.String()).Int64("count", count).Msg("pending batch count")
				metrics.HistogramPendingEventBatches(ctx, count, metrics.HistogramOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"account_id": accountId.String(),
					},
				})
			}
		}
	}()

	return nil
}

func (b batchInstrumenter) Stop(ctx context.Context) error {
	return nil
}

func NewBatchInstrumenter(ctx context.Context, b *redis_state.BatchClient, q queue.Queue) service.Service {
	return &batchInstrumenter{
		batchClient:     b,
		logger:          logger.From(ctx),
		batchManager:    NewRedisBatchManager(b, q),
		leaseLock:       &sync.RWMutex{},
		existingLeaseId: nil,
	}
}
