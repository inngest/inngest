package executor

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/interval"
	"github.com/stretchr/testify/assert"
)

func TestOpcodeTimingQueuedAt(t *testing.T) {
	enqueuedAt := time.Now().Add(-time.Minute).Truncate(time.Millisecond)

	wakeAt := enqueuedAt.Add(30 * time.Second)

	gen := &state.GeneratorOpcode{
		Op:     enums.OpcodeStepFailed,
		ID:     "step-id",
		Timing: interval.New(wakeAt.Add(100*time.Millisecond), wakeAt.Add(200*time.Millisecond)),
	}

	e := &executor{}

	t.Run("step run while resuming a sleep is queued when the sleep ends", func(t *testing.T) {
		rc := &mockRunContext{
			lifecycleItem: queue.Item{
				Kind:       queue.KindSleep,
				EnqueuedAt: enqueuedAt,
				At:         wakeAt,
			},
		}

		queuedAt, scheduledAt, startedAt, endedAt := e.opcodeTiming(context.Background(), rc, gen)

		assert.Equal(t, wakeAt, queuedAt)
		assert.Equal(t, wakeAt, scheduledAt)
		assert.Equal(t, gen.Timing.Start(), startedAt)
		assert.Equal(t, gen.Timing.End(), endedAt)
	})

	t.Run("step run from an edge is queued when its item was enqueued", func(t *testing.T) {
		rc := &mockRunContext{
			lifecycleItem: queue.Item{
				Kind:       queue.KindEdge,
				EnqueuedAt: enqueuedAt,
				At:         wakeAt,
			},
		}

		queuedAt, scheduledAt, startedAt, _ := e.opcodeTiming(context.Background(), rc, gen)

		assert.Equal(t, enqueuedAt, queuedAt)
		assert.Equal(t, wakeAt, scheduledAt)
		assert.Equal(t, gen.Timing.Start(), startedAt)
	})
}
