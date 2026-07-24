package queue

import (
	"context"

	"golang.org/x/sync/errgroup"
)

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
