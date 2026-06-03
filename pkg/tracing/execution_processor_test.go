package tracing

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddQueueTimestampAttrs(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	later := now.Add(5 * time.Second)
	earlier := now.Add(-5 * time.Second)

	t.Run("both zero: no attributes set", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{}
		AddQueueTimestampAttrs(attrs, item)

		_, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		_, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		assert.False(t, hasQueuedAt)
		assert.False(t, hasScheduledAt)
	})

	t.Run("only EnqueuedAt set: QueuedAt set, ScheduledAt absent", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{EnqueuedAt: now}
		AddQueueTimestampAttrs(attrs, item)

		queuedAt, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		_, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		require.True(t, hasQueuedAt)
		assert.Equal(t, now, *queuedAt)
		assert.False(t, hasScheduledAt)
	})

	t.Run("only At set: ScheduledAt equals At, QueuedAt absent", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{At: now}
		AddQueueTimestampAttrs(attrs, item)

		_, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		scheduledAt, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		assert.False(t, hasQueuedAt)
		require.True(t, hasScheduledAt)
		assert.Equal(t, now, *scheduledAt)
	})

	t.Run("At after EnqueuedAt: ScheduledAt equals At", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{EnqueuedAt: now, At: later}
		AddQueueTimestampAttrs(attrs, item)

		queuedAt, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		scheduledAt, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		require.True(t, hasQueuedAt)
		require.True(t, hasScheduledAt)
		assert.Equal(t, now, *queuedAt)
		assert.Equal(t, later, *scheduledAt)
	})

	t.Run("At before EnqueuedAt: EnqueuedAt fudged to ScheduledAt", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{EnqueuedAt: now, At: earlier}
		AddQueueTimestampAttrs(attrs, item)

		queuedAt, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		scheduledAt, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		require.True(t, hasQueuedAt)
		require.True(t, hasScheduledAt)
		assert.Equal(t, now, *queuedAt)
		// ScheduledAt must never be before QueuedAt
		assert.Equal(t, now, *scheduledAt)
		assert.False(t, scheduledAt.Before(*queuedAt))
	})

	t.Run("At equals EnqueuedAt: ScheduledAt equals both", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{EnqueuedAt: now, At: now}
		AddQueueTimestampAttrs(attrs, item)

		queuedAt, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		scheduledAt, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		require.True(t, hasQueuedAt)
		require.True(t, hasScheduledAt)
		assert.Equal(t, now, *queuedAt)
		assert.Equal(t, now, *scheduledAt)
	})
}
