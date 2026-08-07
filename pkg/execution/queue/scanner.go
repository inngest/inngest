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
	// IsRoleActive reports whether this processor currently owns a scanner role.
	// Custom scanners should evaluate it for each scan pass because ownership can
	// change while Run is active.
	IsRoleActive func(name string) bool
}

// QueueScanner discovers and leases queue work. It should hand leased items to
// the dispatch function and leave item execution to the common queue processor layer.
type QueueScanner interface {
	Run(ctx context.Context, rt QueueScannerRuntime) error
}

// QueueScannerRoleProvider allows a custom scanner to declare leased roles it
// needs. The queue processor owns acquisition and renewal of these roles and
// exposes their current state through QueueScannerRuntime.IsRoleActive.
type QueueScannerRoleProvider interface {
	QueueScannerRoles() []QueueRole
}

type partitionQueueScanner struct {
	q *queueProcessor
}

func (partitionQueueScanner) QueueScannerRoles() []QueueRole {
	return []QueueRole{NewSequentialRole()}
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
