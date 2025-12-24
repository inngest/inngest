package deletion

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
)

type ItemHandler func(ctx context.Context, shard queue.QueueShard, qi *queue.QueueItem) error

type DeleteManager interface {
	DeleteQueueItem(ctx context.Context, shard queue.QueueShard, qi *queue.QueueItem) error
}

type deleteManager struct {
	pm    pauses.Manager
	deb   debounce.Debouncer
	batch batch.BatchManager

	handleUnknown ItemHandler
}

// DeleteQueueItem implements DeleteManager.
func (d *deleteManager) DeleteQueueItem(ctx context.Context, shard queue.QueueShard, item *queue.QueueItem) error {
	switch item.Data.Kind {
	// For pause timeouts, delete the associated pause. The pause might otherwise sit in the system for up to a year.
	case queue.KindPause:
		if d.pm == nil {
			break
		}

		payload, ok := item.Data.Payload.(queue.PayloadPauseTimeout)
		if !ok {
			break
		}

		pause, err := d.pm.PauseByID(ctx, pauses.PauseIndex(payload.Pause), payload.PauseID)
		if err != nil {
			break
		}

		if pause == nil {
			break
		}

		err = d.pm.Delete(ctx, pauses.PauseIndex(payload.Pause), *pause)
		if err != nil {
			return fmt.Errorf("could not delete pause for timeout item %q: %w", item.ID, err)
		}
	// Delete associated debounce state
	case queue.KindDebounce:
		if d.deb == nil {
			break
		}

		payload, ok := item.Data.Payload.(debounce.DebouncePayload)
		if !ok {
			break
		}

		di, err := d.deb.GetDebounceItem(ctx, payload.DebounceID, payload.AccountID)
		if err != nil {
			return fmt.Errorf("could not get debounce item: %w", err)
		}

		if di == nil {
			break
		}

		err = d.deb.DeleteDebounceItem(ctx, payload.DebounceID, *di, di.AccountID)
		if err != nil {
			return fmt.Errorf("could not delete debounce item: %w", err)
		}
	// Delete associated batch data
	case queue.KindScheduleBatch:
		if d.batch == nil {
			break
		}

		payload, ok := item.Data.Payload.(batch.ScheduleBatchPayload)
		if !ok {
			break
		}

		err := d.batch.DeleteKeys(ctx, payload.FunctionID, payload.BatchID)
		if err != nil {
			return fmt.Errorf("could not delete batch: %w", err)
		}
	// Some items do not have any other associated data
	// TODO: Should we drop state for function runs?
	case queue.KindEdge, queue.KindEdgeError, queue.KindStart, queue.KindSleep:
		break
	// The following system queues do not have associated state we need to clean up
	case queue.KindQueueMigrate, queue.KindCancel, queue.KindJobPromote, queue.KindPauseBlockFlush:
		break
	default:
		// If the queue item kind is unknown and we have a handler func, execute this to perform external cleanup operations.
		if d.handleUnknown != nil {
			err := d.handleUnknown(ctx, shard, item)
			if err != nil {
				return fmt.Errorf("could not handle item: %w", err)
			}
		}
	}

	partition := item.FunctionID.String()
	if item.QueueName != nil {
		partition = *item.QueueName
	}

	err := shard.RemoveQueueItem(ctx, partition, item.ID)
	if err != nil {
		return fmt.Errorf("could not remove queue item: %w", err)
	}
	return nil
}

type deleteManagerOpt func(o *deleteManager)

func WithPauseManager(pm pauses.Manager) deleteManagerOpt {
	return func(o *deleteManager) {
		o.pm = pm
	}
}

func WithDebouncer(deb debounce.Debouncer) deleteManagerOpt {
	return func(o *deleteManager) {
		o.deb = deb
	}
}

func WithBatchManager(batch batch.BatchManager) deleteManagerOpt {
	return func(o *deleteManager) {
		o.batch = batch
	}
}

func WithUnknownHandler(h ItemHandler) deleteManagerOpt {
	return func(o *deleteManager) {
		o.handleUnknown = h
	}
}

func NewDeleteManager(options ...deleteManagerOpt) (DeleteManager, error) {
	dm := &deleteManager{}
	for _, opt := range options {
		opt(dm)
	}

	return dm, nil
}
