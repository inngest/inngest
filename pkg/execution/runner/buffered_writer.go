package runner

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

const (
	defaultFlushInterval = 100 * time.Millisecond
	defaultFlushSize     = 50
)

// BufferedEventWriter collects events and flushes them in bulk to reduce
// database round-trips. Events are flushed when the buffer reaches
// flushSize or after flushInterval, whichever comes first.
//
// Because flushes happen asynchronously, persistence errors are logged
// rather than returned to callers. This is an intentional trade-off:
// the previous synchronous InsertEvent path treated errors as fatal, but
// at incident scale (78K events) the per-event round-trip was the
// bottleneck. Callers that need synchronous guarantees should use
// InsertEvent directly.
type BufferedEventWriter struct {
	writer cqrs.EventWriter
	log    logger.Logger

	flushInterval time.Duration
	flushSize     int

	mu      sync.Mutex
	buf     []cqrs.Event
	cancel  context.CancelFunc
	done    chan struct{}
	stopped atomic.Bool
}

type BufferedEventWriterOpt func(*BufferedEventWriter)

func WithFlushInterval(d time.Duration) BufferedEventWriterOpt {
	return func(w *BufferedEventWriter) {
		w.flushInterval = d
	}
}

func WithFlushSize(n int) BufferedEventWriterOpt {
	return func(w *BufferedEventWriter) {
		w.flushSize = n
	}
}

func NewBufferedEventWriter(writer cqrs.EventWriter, log logger.Logger, opts ...BufferedEventWriterOpt) *BufferedEventWriter {
	w := &BufferedEventWriter{
		writer:        writer,
		log:           log,
		flushInterval: defaultFlushInterval,
		flushSize:     defaultFlushSize,
		done:          make(chan struct{}),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Start begins the background flush loop. It should be called once.
func (w *BufferedEventWriter) Start(ctx context.Context) {
	cctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	go w.flushLoop(cctx)
}

// Stop flushes remaining events and stops the background loop.
// After Stop returns, any subsequent Write calls fall through to
// synchronous single-event inserts so no events are orphaned.
func (w *BufferedEventWriter) Stop(ctx context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}
	<-w.done
	// Final flush with the provided context so remaining events are persisted.
	w.flushNow(ctx)
	w.stopped.Store(true)
	return nil
}

// Write buffers an event for bulk insertion. If the buffer reaches
// flushSize, an immediate flush is triggered.
// After Stop has been called, Write falls through to a synchronous
// single-event insert so that in-flight handleMessage goroutines
// do not silently lose events.
func (w *BufferedEventWriter) Write(ctx context.Context, e cqrs.Event) {
	if w.stopped.Load() {
		if err := w.writer.InsertEvent(ctx, e); err != nil {
			w.log.Error("post-shutdown event insert failed", "error", err)
		}
		return
	}

	w.mu.Lock()
	w.buf = append(w.buf, e)
	needsFlush := len(w.buf) >= w.flushSize
	w.mu.Unlock()

	if needsFlush {
		w.flushNow(context.Background())
	}
}

func (w *BufferedEventWriter) flushLoop(ctx context.Context) {
	defer close(w.done)
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Use a detached context for DB writes so that loop
			// cancellation does not abort in-flight inserts. When
			// Stop() cancels ctx while a flush is executing, the
			// ticker can fire before ctx.Done() is selected; passing
			// the cancelled ctx would cause ExecContext to fail
			// immediately, losing the buffered events.
			w.flushNow(context.Background())
		}
	}
}

func (w *BufferedEventWriter) flushNow(ctx context.Context) {
	w.mu.Lock()
	if len(w.buf) == 0 {
		w.mu.Unlock()
		return
	}
	events := w.buf
	w.buf = nil
	w.mu.Unlock()

	if err := w.writer.InsertEvents(ctx, events); err != nil {
		var partial *cqrs.PartialInsertError
		if errors.As(err, &partial) {
			w.log.Warn("buffered event writer partial flush",
				"error", err,
				"inserted", partial.Inserted,
				"skipped", partial.Skipped,
			)
			metrics.IncrEventFlushDroppedCounter(ctx, int64(partial.Skipped), metrics.CounterOpt{
				PkgName: pkgName,
			})
		} else {
			w.log.Error("buffered event writer flush failed",
				"error", err,
				"event_count", len(events),
			)
			metrics.IncrEventFlushErrorCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
			})
			metrics.IncrEventFlushDroppedCounter(ctx, int64(len(events)), metrics.CounterOpt{
				PkgName: pkgName,
			})
		}
	}
}
