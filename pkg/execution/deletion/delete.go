package deletion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
)

// itemPayload returns the typed payload for a queue item.
//
// Payloads reach us in two shapes. An item enqueued in-process still holds the
// concrete struct, while an item read back from the queue holds a
// json.RawMessage for every kind that queue.decodePayloadForKind does not
// decode. KindDebounce and KindScheduleBatch are both in the latter group --
// queue cannot import the debounce or batch packages without an import cycle --
// so a plain type assertion silently fails for every persisted item and skips
// its cleanup.
func itemPayload[T any](payload any) (T, bool) {
	var out T
	if typed, ok := payload.(T); ok {
		return typed, true
	}

	var raw []byte
	switch v := payload.(type) {
	case json.RawMessage:
		raw = v
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return out, false
	}

	if len(raw) == 0 {
		return out, false
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, false
	}
	return out, true
}

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

		payload, ok := itemPayload[debounce.DebouncePayload](item.Data.Payload)
		if !ok {
			break
		}

		scope := queue.Scope{
			AccountID:  payload.AccountID,
			EnvID:      payload.WorkspaceID,
			FunctionID: payload.FunctionID,
		}
		di, err := d.deb.GetDebounceItem(ctx, scope, payload.DebounceID)
		if err != nil {
			return fmt.Errorf("could not get debounce item: %w", err)
		}

		if di == nil {
			break
		}

		err = d.deb.DeleteDebounceItem(ctx, scope, payload.DebounceID, *di)
		if err != nil {
			return fmt.Errorf("could not delete debounce item: %w", err)
		}
	// Delete associated batch data
	case queue.KindScheduleBatch:
		if d.batch == nil {
			break
		}

		payload, ok := itemPayload[batch.ScheduleBatchPayload](item.Data.Payload)
		if !ok {
			break
		}

		// Pin cleanup to the backend and key namespace that own this batch.
		// Empty values select the legacy pre-routing namespace on the default
		// backend, which is correct for jobs enqueued before batch routing.
		batchCtx := batch.WithBatchCluster(ctx, payload.BatchCluster)
		batchCtx = redis_state.WithBatchGeneration(batchCtx, payload.BatchGeneration)

		err := d.batch.DeleteKeys(batchCtx, payload.FunctionID, payload.BatchID)
		if err != nil {
			return fmt.Errorf("could not delete batch: %w", err)
		}
	// Some items do not have any other associated data
	// TODO: Should we drop state for function runs?
	case queue.KindEdge, queue.KindEdgeError, queue.KindStart, queue.KindSleep:
		break
	// The following system queues do not have associated state we need to clean up
	case queue.KindCancel, queue.KindJobPromote, queue.KindPauseBlockFlush:
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

	err := shard.RemoveQueueItem(ctx, queue.ScopeFromQueueItem(*item), partition, item.ID)
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
