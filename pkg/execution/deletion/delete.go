package deletion

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
)

type ItemHandler func(ctx context.Context, shard redis_state.QueueShard, qi *queue.QueueItem) error

type DeleteManager interface {
	DeleteQueueItem(ctx context.Context, shard redis_state.QueueShard, qi *queue.QueueItem) error
}

type deleteManager struct {
	pm    pauses.Manager
	qm    redis_state.QueueManager
	deb   debounce.Debouncer
	batch batch.BatchManager

	handleUnknown ItemHandler
}

// DeleteQueueItem implements DeleteManager.
func (d *deleteManager) DeleteQueueItem(ctx context.Context, shard redis_state.QueueShard, item *queue.QueueItem) error {
	switch item.Data.Kind {
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
	case queue.KindEdge, queue.KindEdgeError, queue.KindStart, queue.KindSleep:
		break
	case queue.KindQueueMigrate, queue.KindCancel, queue.KindJobPromote, queue.KindPauseBlockFlush:
		break
	default:
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

	partitionKey := shard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, partition, "")

	err := d.qm.RemoveQueueItem(ctx, shard.Name, partitionKey, item.ID)
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

func WithQueueManager(qm redis_state.QueueManager) deleteManagerOpt {
	return func(o *deleteManager) {
		o.qm = qm
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

	if dm.qm == nil {
		return nil, fmt.Errorf("missing queue manager")
	}

	return dm, nil
}
