package base_cqrs

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

func TestSorter(t *testing.T) {
	now := time.Now()

	t.Run("sorts by start time when different", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			Children: []*cqrs.OtelSpan{
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span2", StartTime: now.Add(2 * time.Second)}},
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span1", StartTime: now.Add(1 * time.Second)}},
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span3", StartTime: now.Add(3 * time.Second)}},
			},
		}

		sorter(span)

		require.Equal(t, "span1", span.Children[0].SpanID)
		require.Equal(t, "span2", span.Children[1].SpanID)
		require.Equal(t, "span3", span.Children[2].SpanID)
	})

	t.Run("sorts by span ID when start times are equal", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			Children: []*cqrs.OtelSpan{
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span_c", StartTime: now}},
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span_a", StartTime: now}},
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span_b", StartTime: now}},
			},
		}

		sorter(span)

		// This test should fail on main branch but pass on chore/EXE-192
		// because main branch doesn't handle equal timestamps
		require.Equal(t, "span_a", span.Children[0].SpanID)
		require.Equal(t, "span_b", span.Children[1].SpanID)
		require.Equal(t, "span_c", span.Children[2].SpanID)
	})

	t.Run("mixed case - sort by time first, then by span ID", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			Children: []*cqrs.OtelSpan{
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span_z", StartTime: now.Add(1 * time.Second)}},
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span_b", StartTime: now}},
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span_a", StartTime: now}},
				{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "span_y", StartTime: now.Add(1 * time.Second)}},
			},
		}

		sorter(span)

		// Should sort by time first (now < now+1s), then by SpanID for equal times
		require.Equal(t, "span_a", span.Children[0].SpanID)
		require.Equal(t, "span_b", span.Children[1].SpanID)
		require.Equal(t, "span_y", span.Children[2].SpanID)
		require.Equal(t, "span_z", span.Children[3].SpanID)
	})

	t.Run("recursively sorts nested children", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			Children: []*cqrs.OtelSpan{
				{
					RawOtelSpan: cqrs.RawOtelSpan{SpanID: "parent2", StartTime: now.Add(2 * time.Second)},
					Children: []*cqrs.OtelSpan{
						{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "child2_b", StartTime: now.Add(3 * time.Second)}},
						{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "child2_a", StartTime: now.Add(3 * time.Second)}},
					},
				},
				{
					RawOtelSpan: cqrs.RawOtelSpan{SpanID: "parent1", StartTime: now.Add(1 * time.Second)},
					Children: []*cqrs.OtelSpan{
						{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "child1_z", StartTime: now.Add(2 * time.Second)}},
						{RawOtelSpan: cqrs.RawOtelSpan{SpanID: "child1_a", StartTime: now.Add(1 * time.Second)}},
					},
				},
			},
		}

		sorter(span)

		// Parent level should be sorted by time
		require.Equal(t, "parent1", span.Children[0].SpanID)
		require.Equal(t, "parent2", span.Children[1].SpanID)

		// Children should be sorted within each parent
		require.Equal(t, "child1_a", span.Children[0].Children[0].SpanID)
		require.Equal(t, "child1_z", span.Children[0].Children[1].SpanID)

		// Children with same timestamp should be sorted by SpanID
		require.Equal(t, "child2_a", span.Children[1].Children[0].SpanID)
		require.Equal(t, "child2_b", span.Children[1].Children[1].SpanID)
	})
}
