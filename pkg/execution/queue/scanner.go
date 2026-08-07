package queue

import (
	"context"

	"github.com/inngest/inngest/pkg/util"
	"golang.org/x/sync/errgroup"
)

type QueueScannerRuntime struct {
	Leaser          QueueItemLeaser
	Dispatch        DispatchFunc
	WorkerSemaphore util.TrackingSemaphore
	// IsSequential reports whether this processor currently owns the sequential
	// scanner role. Custom scanners should evaluate it for each scan pass because
	// role ownership can change while Run is active.
	IsSequential func() bool
}

// QueueScanner discovers and leases queue work. It should hand leased items to
// the dispatch function and leave item execution to the common queue processor layer.
type QueueScanner interface {
	Run(ctx context.Context, rt QueueScannerRuntime) error
}

type partitionQueueScanner struct {
	q *queueProcessor
}

func (s partitionQueueScanner) Run(ctx context.Context, rt QueueScannerRuntime) error {
	q := s.q

	// start execution and shadow scan concurrently
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return q.executionScan(ctx, rt.Dispatch)
	})

	if q.runMode.ShadowPartition {
		eg.Go(func() error {
			return q.shadowScan(ctx)
		})
	}

	if q.runMode.NormalizePartition {
		eg.Go(func() error {
			return q.backlogNormalizationScan(ctx)
		})
	}

	return eg.Wait()
}
