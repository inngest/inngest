package tracing

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/stretchr/testify/require"
)

// memInserter is an in-memory Inserter: it records every flushed batch by
// entity, so listener tests can assert what reached the sink without a
// duckdb subprocess.
type memInserter struct {
	mu       sync.Mutex
	spans    []Span
	events   []Event
	metadata []RunMetadata
	batches  int
	err      error
	closed   bool
}

func (m *memInserter) InsertSpans(_ context.Context, spans []Span) error {
	return m.record(func() { m.spans = append(m.spans, spans...) })
}

func (m *memInserter) InsertEvents(_ context.Context, events []Event) error {
	return m.record(func() { m.events = append(m.events, events...) })
}

func (m *memInserter) InsertRunMetadata(_ context.Context, entries []RunMetadata) error {
	return m.record(func() { m.metadata = append(m.metadata, entries...) })
}

func (m *memInserter) record(apply func()) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batches++
	if m.err != nil {
		return m.err
	}
	apply()
	return nil
}

func (m *memInserter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *memInserter) snapshot() (spans []Span, events []Event, metadata []RunMetadata) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Span(nil), m.spans...), append([]Event(nil), m.events...), append([]RunMetadata(nil), m.metadata...)
}

// TestListenerWithInserterRoutesEntities proves NewListenerWithInserter
// builds the same entities as the DuckDB path but hands them to any
// Inserter, one method per entity: events straight from their hook, spans
// via the listener's own SpanExporter.
func TestListenerWithInserterRoutesEntities(t *testing.T) {
	ins := &memInserter{}
	l := NewListenerWithInserter(ins, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })
	md := testMetadata(t)

	l.OnEventReceived(context.Background(), event.NewBaseTrackedEvent(event.Event{Name: "app/test"}, nil))
	l.OnFunctionScheduled(context.Background(), md, queue.Item{}, nil)

	require.Eventually(t, func() bool {
		spans, events, _ := ins.snapshot()
		return len(events) == 1 && len(spans) >= 1
	}, 2*time.Second, 20*time.Millisecond)

	spans, events, _ := ins.snapshot()
	require.Equal(t, "app/test", events[0].EventName)
	require.JSONEq(t, `{}`, string(events[0].EventData), "nil event data is normalized to {}")

	var queued *Span
	for i := range spans {
		if spans[i].Name == "executor.run.queued" {
			queued = &spans[i]
		}
	}
	require.NotNil(t, queued, "the run.queued span should reach InsertSpans")
	require.Equal(t, md.ID.RunID.String(), queued.RunID)
	require.Equal(t, md.ID.FunctionID.String(), queued.FunctionID)

	require.NoError(t, l.(Closer).Close(context.Background()))
	require.True(t, ins.closed, "Close should close an Inserter that implements Close")
}

// TestListenerWithInserterDropsFailedBatches proves an Inserter error is
// contained: the batch is dropped and later batches still flush.
func TestListenerWithInserterDropsFailedBatches(t *testing.T) {
	ins := &memInserter{err: errors.New("sink down")}
	l := NewListenerWithInserter(ins, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })
	t.Cleanup(func() { _ = l.(Closer).Close(context.Background()) })

	l.OnEventReceived(context.Background(), event.NewBaseTrackedEvent(event.Event{Name: "app/dropped"}, nil))
	require.Eventually(t, func() bool {
		ins.mu.Lock()
		defer ins.mu.Unlock()
		return ins.batches >= 1
	}, 2*time.Second, 20*time.Millisecond)

	ins.mu.Lock()
	ins.err = nil
	ins.mu.Unlock()

	l.OnEventReceived(context.Background(), event.NewBaseTrackedEvent(event.Event{Name: "app/kept"}, nil))
	require.Eventually(t, func() bool {
		_, events, _ := ins.snapshot()
		return len(events) == 1 && events[0].EventName == "app/kept"
	}, 2*time.Second, 20*time.Millisecond)
}

// memSink is a direct Sink (no buffering), like a Kafka producer would be:
// it records each entity the moment the listener hands it over.
type memSink struct {
	mu       sync.Mutex
	spans    []Span
	events   []Event
	metadata []RunMetadata
	closed   bool
}

func (m *memSink) SendSpan(_ context.Context, s Span) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spans = append(m.spans, s)
}

func (m *memSink) SendEvent(_ context.Context, e Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
}

func (m *memSink) SendRunMetadata(_ context.Context, r RunMetadata) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata = append(m.metadata, r)
}

func (m *memSink) Close(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// TestListenerWithSinkSendsEntitiesSynchronously proves NewListenerWithSink
// bypasses in-process batching entirely: each entity reaches the Sink during
// the hook call itself, including spans (via the listener's own
// TracerProvider and SpanExporter) and metadata.
func TestListenerWithSinkSendsEntitiesSynchronously(t *testing.T) {
	sink := &memSink{}
	l := NewListenerWithSink(sink)
	md := testMetadata(t)

	l.OnEventReceived(context.Background(), event.NewBaseTrackedEvent(event.Event{Name: "app/direct"}, nil))
	l.OnFunctionScheduled(context.Background(), md, queue.Item{}, nil)
	l.OnMetadataEntry(context.Background(), execution.MetadataEntry{
		AccountID:  md.ID.Tenant.AccountID,
		EnvID:      md.ID.Tenant.EnvID,
		AppID:      md.ID.Tenant.AppID,
		FunctionID: md.ID.FunctionID,
		RunID:      md.ID.RunID,
		Parent:     tracing.RunSpanRefFromMetadata(&md),
		Kind:       metadata.Kind("test.kind"),
		Scope:      enums.MetadataScopeRun,
		Values:     metadata.Values{"foo": json.RawMessage(`"bar"`)},
		CreatedAt:  time.Now(),
	})

	// No Eventually: a direct Sink sees everything before the hooks return.
	sink.mu.Lock()
	defer sink.mu.Unlock()
	require.Len(t, sink.events, 1)
	require.Equal(t, "app/direct", sink.events[0].EventName)

	require.NotEmpty(t, sink.spans)
	var names []string
	for _, s := range sink.spans {
		names = append(names, s.Name)
		require.Equal(t, md.ID.RunID.String(), s.RunID)
	}
	require.Contains(t, names, "executor.run.queued")

	require.Len(t, sink.metadata, 1)
	require.Equal(t, md.ID.RunID.String(), sink.metadata[0].RunID)
	require.Equal(t, metadata.Kind("test.kind").Suffix(), sink.metadata[0].Kind)
	require.Nil(t, sink.metadata[0].StepID, "run-scoped metadata has no step")
	sink.mu.Unlock()

	require.NoError(t, l.(Closer).Close(context.Background()))
	sink.mu.Lock()
	require.True(t, sink.closed, "Close should close a Sink that implements Close(ctx)")
}

func TestRootSpanParentSpanIDIsEmpty(t *testing.T) {
	sink := &memSink{}
	tp := newListenerTracerProvider(&SpanExporter{sink: sink}, 0)
	md := testMetadata(t)

	rootAttrs := meta.NewAttrSet()
	addTenantAndDebugAttrs(rootAttrs, &md)
	rootRef, err := tp.CreateSpan(context.Background(), "executor.run", &tracing.CreateSpanOptions{
		Metadata:   safeMetadata(md),
		Attributes: rootAttrs,
		StartTime:  time.Now(),
		EndTime:    time.Now(),
	})
	require.NoError(t, err)

	childAttrs := meta.NewAttrSet()
	addTenantAndDebugAttrs(childAttrs, &md)
	_, err = tp.CreateSpan(context.Background(), "executor.step", &tracing.CreateSpanOptions{
		Metadata:   safeMetadata(md),
		Parent:     rootRef,
		Attributes: childAttrs,
		StartTime:  time.Now(),
		EndTime:    time.Now(),
	})
	require.NoError(t, err)

	sink.mu.Lock()
	defer sink.mu.Unlock()
	require.Len(t, sink.spans, 2)
	require.Equal(t, "executor.run", sink.spans[0].Name)
	require.Empty(t, sink.spans[0].ParentSpanID, "root span parent_span_id should be empty (stored as NULL), not 0000000000000000")
	require.Equal(t, "executor.step", sink.spans[1].Name)
	require.Equal(t, sink.spans[0].SpanID, sink.spans[1].ParentSpanID, "child span should reference its parent span ID")
}

type blockingInserter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	closed  bool
	mu      sync.Mutex
}

func (b *blockingInserter) InsertSpans(context.Context, []Span) error { return nil }
func (b *blockingInserter) InsertEvents(ctx context.Context, _ []Event) error {
	b.once.Do(func() { close(b.started) })
	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (b *blockingInserter) InsertRunMetadata(context.Context, []RunMetadata) error { return nil }
func (b *blockingInserter) Close() error {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
	return nil
}

func TestBatchingSinkCloseIdempotentAndConcurrentDeadlines(t *testing.T) {
	ins := &memInserter{}
	s := NewBatchingSink(ins, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	// Calling Close multiple times must not panic on closing batcher stop channels.
	require.NotPanics(t, func() {
		require.NoError(t, s.Close(context.Background()))
		require.NoError(t, s.Close(context.Background()))
	})
	require.True(t, ins.closed)

	// Concurrent Close callers must each honor their own context deadline even
	// while an earlier Close is still waiting for a slow flush to drain.
	bins := &blockingInserter{started: make(chan struct{}), release: make(chan struct{})}
	s2 := NewBatchingSink(bins, func(o *setupOpts) {
		o.batchMaxSize = 1
		o.batchInterval = time.Hour
	})
	s2.SendEvent(context.Background(), Event{EventID: "e1"})
	<-bins.started

	close1Done := make(chan error, 1)
	go func() {
		close1Done <- s2.Close(context.Background())
	}()

	// Give caller 1 a moment to initiate closeOnce.
	time.Sleep(15 * time.Millisecond)

	shortCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	err2 := s2.Close(shortCtx)
	require.ErrorIs(t, err2, context.DeadlineExceeded)
	require.Less(t, time.Since(start), time.Second, "concurrent Close must honor its own deadline")

	// Unblock flush and ensure caller 1 finishes cleanly.
	close(bins.release)
	select {
	case err1 := <-close1Done:
		require.NoError(t, err1)
	case <-time.After(2 * time.Second):
		t.Fatal("first Close did not complete after flush unblocked")
	}
	bins.mu.Lock()
	require.True(t, bins.closed)
	bins.mu.Unlock()
}

// Compile-time checks.
var (
	_ Inserter                        = (*DuckDBInserter)(nil)
	_ Inserter                        = (*memInserter)(nil)
	_ Sink                            = (*BatchingSink)(nil)
	_ Sink                            = (*memSink)(nil)
	_ execution.SyncLifecycleListener = (*listener)(nil)
)
