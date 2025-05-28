package redis_state

import (
	"context"
)

type cancellationChanMsg struct {
}

func (q *queue) cancellationScan(ctx context.Context) error {
	cc := make(chan cancellationChanMsg)

	for i := int32(0); i < q.numCancellationWorkers; i++ {
		go q.cancellationWorker(ctx, cc)
	}

	tick := q.clock.NewTicker(q.cancelPollTick)
	q.log.Debug("starting cancellation scanner", "poll", q.cancelPollTick.String())

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return nil

		case <-tick.Chan():
			// TODO: scan cancellation partitions
		}
	}
}

func (q *queue) cancellationWorker(ctx context.Context, cc chan cancellationChanMsg) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-cc:
			// TODO: process whatever that needs to be cancelled
		}
	}
}
