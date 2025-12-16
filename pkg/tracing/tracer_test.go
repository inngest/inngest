package tracing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeterministicTraceID(t *testing.T) {
	ctx := t.Context()

	for range 10 {
		tp := NewOtelTracerProvider(nil, time.Millisecond)
		s, err := tp.CreateDroppableSpan(ctx, "whatever", &CreateSpanOptions{
			Seed: []byte("whatever"),
		})
		require.NoError(t, err)

		require.Equal(t, "3ef37e678cd5f60a", s.span.SpanContext().SpanID().String())
		require.Equal(t, "b9f5a2dd4a1235e85b5d67e2b5f7394b", s.span.SpanContext().TraceID().String())

		ref := s.Ref

		require.Equal(t, "3ef37e678cd5f60a", ref.DynamicSpanID)
		require.Equal(t, "00-b9f5a2dd4a1235e85b5d67e2b5f7394b-3ef37e678cd5f60a-01", ref.TraceParent)
		require.Equal(t, "00-b9f5a2dd4a1235e85b5d67e2b5f7394b-0000000000000000-01", ref.DynamicSpanTraceParent)

		// Another span with a different seed should act differently.
		s2, err := tp.CreateDroppableSpan(ctx, "another", &CreateSpanOptions{
			Seed: []byte("another"),
		})
		require.NoError(t, err)
		require.NotEqual(t, s2.span.SpanContext().SpanID().String(), s.span.SpanContext().SpanID().String())
		require.NotEqual(t, s2.span.SpanContext().TraceID().String(), s.span.SpanContext().TraceID().String())
		ref2 := s2.Ref
		require.NotEqual(t, ref.DynamicSpanID, ref2.DynamicSpanID)
		require.NotEqual(t, ref.TraceParent, ref2.TraceParent)
		require.NotEqual(t, ref.DynamicSpanTraceParent, ref2.DynamicSpanTraceParent)
	}
}

func TestRandomSpanIDGeneration(t *testing.T) {
	ctx := t.Context()
	tp := NewOtelTracerProvider(nil, time.Millisecond)

	// Store span and trace IDs to check for uniqueness
	spanIDs := make(map[string]bool)
	traceIDs := make(map[string]bool)
	dynamicSpanIDs := make(map[string]bool)
	traceParents := make(map[string]bool)

	for i := 0; i < 10; i++ {
		s, err := tp.CreateDroppableSpan(ctx, "random-span", &CreateSpanOptions{
			// No seed - should generate random IDs
		})
		require.NoError(t, err)

		spanID := s.span.SpanContext().SpanID().String()
		traceID := s.span.SpanContext().TraceID().String()
		dynamicSpanID := s.Ref.DynamicSpanID
		traceParent := s.Ref.TraceParent

		// Ensure no duplicates
		require.False(t, spanIDs[spanID], "Span ID %s was generated twice", spanID)
		require.False(t, traceIDs[traceID], "Trace ID %s was generated twice", traceID)
		require.False(t, dynamicSpanIDs[dynamicSpanID], "Dynamic span ID %s was generated twice", dynamicSpanID)
		require.False(t, traceParents[traceParent], "TraceParent %s was generated twice", traceParent)

		// Store for future uniqueness checks
		spanIDs[spanID] = true
		traceIDs[traceID] = true
		dynamicSpanIDs[dynamicSpanID] = true
		traceParents[traceParent] = true

		// Ensure IDs are not empty
		require.NotEmpty(t, spanID)
		require.NotEmpty(t, traceID)
		require.NotEmpty(t, dynamicSpanID)
		require.NotEmpty(t, traceParent)
	}

	// Verify we collected 10 unique IDs of each type
	require.Len(t, spanIDs, 10)
	require.Len(t, traceIDs, 10)
	require.Len(t, dynamicSpanIDs, 10)
	require.Len(t, traceParents, 10)
}

func TestSeededSpanThenReuseContext(t *testing.T) {
	ctx := t.Context()
	tp := NewOtelTracerProvider(nil, time.Millisecond)

	// Create a seeded span first
	seededSpan, err := tp.CreateDroppableSpan(ctx, "seeded-span", &CreateSpanOptions{
		Seed: []byte("test-seed"),
	})
	require.NoError(t, err)

	// Get the IDs from the seeded span
	seededSpanID := seededSpan.span.SpanContext().SpanID().String()
	seededTraceID := seededSpan.span.SpanContext().TraceID().String()
	seededDynamicSpanID := seededSpan.Ref.DynamicSpanID
	seededTraceParent := seededSpan.Ref.TraceParent

	// Now reuse the same context to create a new span without seed
	// This should create a new span with different IDs, not reuse the seeded ones
	newSpan, err := tp.CreateDroppableSpan(ctx, "new-span", &CreateSpanOptions{
		// No seed - should generate different IDs
	})
	require.NoError(t, err)

	// Get the IDs from the new span
	newSpanID := newSpan.span.SpanContext().SpanID().String()
	newTraceID := newSpan.span.SpanContext().TraceID().String()
	newDynamicSpanID := newSpan.Ref.DynamicSpanID
	newTraceParent := newSpan.Ref.TraceParent

	// Verify that the new span has different IDs than the seeded span
	require.NotEqual(t, seededSpanID, newSpanID, "New span should have different span ID than seeded span")
	require.NotEqual(t, seededTraceID, newTraceID, "New span should have different trace ID than seeded span")
	require.NotEqual(t, seededDynamicSpanID, newDynamicSpanID, "New span should have different dynamic span ID than seeded span")
	require.NotEqual(t, seededTraceParent, newTraceParent, "New span should have different trace parent than seeded span")

	// Verify that IDs are not empty
	require.NotEmpty(t, newSpanID)
	require.NotEmpty(t, newTraceID)
	require.NotEmpty(t, newDynamicSpanID)
	require.NotEmpty(t, newTraceParent)
}
