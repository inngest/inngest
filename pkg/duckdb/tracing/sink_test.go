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

// Compile-time checks.
var (
	_ Inserter                        = (*DuckDBInserter)(nil)
	_ Inserter                        = (*memInserter)(nil)
	_ Sink                            = (*BatchingSink)(nil)
	_ Sink                            = (*memSink)(nil)
	_ execution.SyncLifecycleListener = (*listener)(nil)
)
