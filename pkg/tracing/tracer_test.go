package tracing

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

// TestNoTracerProviderLeakAcrossSpans is the regression test for issue #3892.
// Previously otelTracerProvider.getTracer built a fresh sdktrace.NewTracerProvider
// plus SimpleSpanProcessor plus executionProcessor on every CreateSpan / UpdateSpan,
// and never called Shutdown on any of them. Under sustained traffic the Go runtime
// could not reclaim them (RSS 500MB -> 10GB over 48h in the reporter cluster).
//
// This test proves two invariants of the fix:
//
//  1. NewOtelTracerProvider now builds the underlying sdktrace.TracerProvider ONCE
//     and reuses the same trace.Tracer forever: the pointer returned by getTracer
//     must be identical across many CreateSpan calls.
//  2. Creating many spans does not leak goroutines. We take a goroutine snapshot
//     before the loop, create 2000 spans, let the GC settle, and assert the delta
//     is bounded (the pre-fix behaviour spawned one SimpleSpanProcessor goroutine
//     per call = 2000+ goroutines).
func TestNoTracerProviderLeakAcrossSpans(t *testing.T) {
	ctx := t.Context()
	tp := NewOtelTracerProvider(nil, time.Millisecond)

	// Invariant 1: shared Tracer identity.
	otp, ok := tp.(*otelTracerProvider)
	require.True(t, ok, "NewOtelTracerProvider should return *otelTracerProvider")
	first := otp.getTracer(nil)
	second := otp.getTracer(nil)
	require.Same(t, first, second, "getTracer must return the same Tracer instance across calls (issue #3892)")

	// Invariant 2: no goroutine leak across many spans.
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	const spans = 2000
	for i := 0; i < spans; i++ {
		s, err := tp.CreateDroppableSpan(ctx, "leak-check", &CreateSpanOptions{})
		require.NoError(t, err)
		s.Drop()
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	after := runtime.NumGoroutine()

	// Pre-fix: expect ~2000 extra goroutines (one SimpleSpanProcessor per call).
	// Post-fix: no new goroutines should be created per span. Allow a tiny slack
	// for test-infra noise (<= 20).
	require.LessOrEqual(t, after-baseline, 20,
		"goroutine count grew by %d after %d spans; TracerProvider is still being rebuilt per span (issue #3892)",
		after-baseline, spans)
}

// recordingExporter captures every ExportSpans batch for inspection.
// Deliberately simple so the test can assert on span attributes.
type recordingExporter struct {
	mu   sync.Mutex
	seen []sdktrace.ReadOnlySpan
}

func (e *recordingExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.seen = append(e.seen, spans...)
	return nil
}

func (e *recordingExporter) Shutdown(_ context.Context) error { return nil }

// TestMetadataFlowsViaContextAcrossRuns verifies the per-span metadata
// pipeline rewritten for #3892. Before the fix each span had its own
// executionProcessor that captured md on construction; after the fix a
// single shared executionProcessor reads md from context on OnStart.
// This test proves the shared processor still delivers the correct
// per-run tenant attributes even when two different runs interleave.
func TestMetadataFlowsViaContextAcrossRuns(t *testing.T) {
	ctx := t.Context()
	rec := &recordingExporter{}
	// Short batch timeout so the batcher flushes within the test window.
	tp := NewOtelTracerProvider(rec, 5*time.Millisecond)

	mdA := &statev2.Metadata{}
	mdA.ID.Tenant.AccountID = uuid.New()
	mdA.ID.Tenant.EnvID = uuid.New()
	mdA.ID.Tenant.AppID = uuid.New()

	mdB := &statev2.Metadata{}
	mdB.ID.Tenant.AccountID = uuid.New()
	mdB.ID.Tenant.EnvID = uuid.New()
	mdB.ID.Tenant.AppID = uuid.New()

	// Interleave spans from two runs so the shared processor must distinguish
	// them purely by context, not by any captured per-run state.
	for i := 0; i < 10; i++ {
		for _, md := range []*statev2.Metadata{mdA, mdB} {
			_, err := tp.CreateSpan(ctx, "step", &CreateSpanOptions{Metadata: md})
			require.NoError(t, err)
		}
	}

	// Flush any batched spans before we read.
	require.NoError(t, tp.Shutdown(ctx))

	rec.mu.Lock()
	defer rec.mu.Unlock()
	require.Equal(t, 20, len(rec.seen), "expected 20 recorded spans (10 per run)")

	countA, countB := 0, 0
	accountA := mdA.ID.Tenant.AccountID.String()
	accountB := mdB.ID.Tenant.AccountID.String()
	for _, span := range rec.seen {
		for _, attr := range span.Attributes() {
			if string(attr.Key) == meta.Attrs.AccountID.Key() {
				switch attr.Value.AsString() {
				case accountA:
					countA++
				case accountB:
					countB++
				}
			}
		}
	}
	require.Equal(t, 10, countA, "shared processor must enrich mdA spans with mdA attributes")
	require.Equal(t, 10, countB, "shared processor must enrich mdB spans with mdB attributes")
}
