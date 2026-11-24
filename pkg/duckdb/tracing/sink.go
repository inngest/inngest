package tracing

import (
	"context"
	"sync"
	"sync/atomic"
)

// Sink receives every entity the listener produces, one call per entity.
//
// Calls happen synchronously on the calling hook's goroutine, which is the
// executor's and runner's critical path, so implementations must never block
// on I/O: buffer and drop, or hand off asynchronously (e.g. an async Kafka
// producer). A Sink may also implement interface{ Close(context.Context) error },
// which the listener's Close calls.
//
// BatchingSink is the in-process implementation: per-entity channels drained
// by batchers into an Inserter.
type Sink interface {
	SendSpan(ctx context.Context, s Span)
	SendEvent(ctx context.Context, e Event)
	SendRunMetadata(ctx context.Context, m RunMetadata)
}

// BatchingSink buffers each entity type on its own bounded channel, drained
// by a background batcher (batch.go) that flushes through an Inserter. Sends
// never block: a full channel drops the entity and counts it.
type BatchingSink struct {
	spans    chan Span
	events   chan Event
	metadata chan RunMetadata

	droppedSpans    atomic.Int64
	droppedEvents   atomic.Int64
	droppedMetadata atomic.Int64

	ins       Inserter
	batchers  []interface{ stop() }
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    chan struct{}
	closeErr  error
}

// NewBatchingSink returns a BatchingSink flushing through ins, with its
// batcher goroutines already running. They run until Close.
func NewBatchingSink(ins Inserter, opts ...Option) *BatchingSink {
	o := defaultSetupOpts()
	for _, apply := range opts {
		apply(&o)
	}

	s := newBatchingSinkChannels(o.spansCap, o.eventsCap, o.metadataCap)
	s.ins = ins

	// One shared disabledState across every batcher, so temporary
	// duckdb.ErrDisabled cooldown warnings share a single rate limit rather
	// than logging on every flush across every entity.
	disabled := &disabledState{}
	startBatcher(s, newBatcher("run_trace_spans", s.spans, ins.InsertSpans, batcherOpts{maxSize: o.batchMaxSize, flushInterval: o.batchInterval, disabled: disabled, dropped: &s.droppedSpans}))
	startBatcher(s, newBatcher("events", s.events, ins.InsertEvents, batcherOpts{maxSize: o.batchMaxSize, flushInterval: o.batchInterval, disabled: disabled, dropped: &s.droppedEvents}))
	startBatcher(s, newBatcher("run_metadata", s.metadata, ins.InsertRunMetadata, batcherOpts{maxSize: o.batchMaxSize, flushInterval: o.batchInterval, disabled: disabled, dropped: &s.droppedMetadata}))
	return s
}

// newBatchingSinkChannels returns a BatchingSink with its channels but no
// batchers, so tests can observe buffering and drops directly.
func newBatchingSinkChannels(spansCap, eventsCap, metadataCap int) *BatchingSink {
	return &BatchingSink{
		spans:    make(chan Span, spansCap),
		events:   make(chan Event, eventsCap),
		metadata: make(chan RunMetadata, metadataCap),
		closed:   make(chan struct{}),
	}
}

// startBatcher runs b on its own goroutine until Close, registering it for
// Close.
func startBatcher[T any](s *BatchingSink, b *batcher[T]) {
	s.batchers = append(s.batchers, b)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		b.run(context.Background())
	}()
}

func (s *BatchingSink) SendSpan(_ context.Context, sp Span) {
	select {
	case s.spans <- sp:
	default:
		s.droppedSpans.Add(1)
	}
}

func (s *BatchingSink) SendEvent(_ context.Context, e Event) {
	select {
	case s.events <- e:
	default:
		s.droppedEvents.Add(1)
	}
}

func (s *BatchingSink) SendRunMetadata(_ context.Context, m RunMetadata) {
	select {
	case s.metadata <- m:
	default:
		s.droppedMetadata.Add(1)
	}
}

// Dropped returns how many entities of each type were dropped because their
// channel was full.
func (s *BatchingSink) Dropped() (spans, events, metadata int64) {
	return s.droppedSpans.Load(), s.droppedEvents.Load(), s.droppedMetadata.Load()
}

// Close stops every batcher, waits (bounded by ctx) for them to drain and
// flush, then closes the Inserter if it implements interface{ Close() error }.
//
// Shutdown initiation runs once in a background goroutine so concurrent Close
// callers each select between shutdown completion and their own ctx deadline.
func (s *BatchingSink) Close(ctx context.Context) error {
	s.closeOnce.Do(func() {
		if s.closed == nil {
			s.closed = make(chan struct{})
		}
		go func() {
			defer close(s.closed)
			for _, b := range s.batchers {
				b.stop()
			}

			// Wait for every batcher to finish any in-flight flush (bounded by
			// flushExecTimeout) and its final drain flush (bounded by
			// finalFlushBudget) before closing the Inserter underneath it.
			// Callers with a tighter deadline return via ctx.Done() below.
			s.wg.Wait()

			if c, ok := s.ins.(interface{ Close() error }); ok {
				s.closeErr = c.Close()
			}
		}()
	})

	select {
	case <-s.closed:
		return s.closeErr
	default:
	}

	select {
	case <-s.closed:
		return s.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}
