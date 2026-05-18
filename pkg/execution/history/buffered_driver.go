package history

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
)

const (
	DefaultBatchSize     = 500
	DefaultFlushInterval = 50 * time.Millisecond
)

// BufferedDriver wraps a Driver, collecting individual Write calls into
// batches that are flushed via WriteBatch. Terminal history types
// (cancelled, completed, failed) bypass the buffer: the pending buffer
// is flushed first, then the terminal row is written synchronously via
// the underlying Driver.Write so that InsertFunctionFinish is sequenced
// after all preceding history INSERTs.
type BufferedDriver struct {
	driver Driver
	log    logger.Logger

	mu  sync.Mutex
	buf []History

	batchSize     int
	flushInterval time.Duration

	flushMu   sync.Mutex
	wg        sync.WaitGroup
	closing   atomic.Bool
	closeOnce sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// NewBufferedDriver creates a BufferedDriver that wraps the given driver.
// Call Close to stop the background flush goroutine and drain remaining items.
func NewBufferedDriver(driver Driver, log logger.Logger) *BufferedDriver {
	b := &BufferedDriver{
		driver:        driver,
		log:           log,
		buf:           make([]History, 0, DefaultBatchSize),
		batchSize:     DefaultBatchSize,
		flushInterval: DefaultFlushInterval,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
	go b.flushLoop()
	return b
}

func (b *BufferedDriver) Write(ctx context.Context, h History) error {
	if isTerminalType(h.Type) {
		return b.writeTerminal(ctx, h)
	}

	b.mu.Lock()
	b.buf = append(b.buf, h)
	shouldFlush := len(b.buf) >= b.batchSize
	b.mu.Unlock()

	if shouldFlush {
		b.flushAsync()
	}
	return nil
}

func (b *BufferedDriver) WriteBatch(ctx context.Context, items []History) error {
	return b.driver.WriteBatch(ctx, items)
}

func (b *BufferedDriver) Close(ctx context.Context) error {
	b.closing.Store(true)
	b.closeOnce.Do(func() { close(b.stopCh) })
	<-b.doneCh

	b.wg.Wait()

	b.mu.Lock()
	remaining := b.buf
	b.buf = nil
	b.mu.Unlock()

	if len(remaining) > 0 {
		if err := b.driver.WriteBatch(context.Background(), remaining); err != nil {
			b.log.Error("error flushing remaining history on close", "error", err)
		}
	}

	return b.driver.Close(ctx)
}

// writeTerminal flushes the buffer, then writes the terminal row
// synchronously via the underlying driver so InsertFunctionFinish
// is sequenced after all buffered history rows.
func (b *BufferedDriver) writeTerminal(ctx context.Context, h History) error {
	b.flushMu.Lock()
	defer b.flushMu.Unlock()

	b.mu.Lock()
	pending := b.buf
	b.buf = make([]History, 0, b.batchSize)
	b.mu.Unlock()

	if len(pending) > 0 {
		if err := b.driver.WriteBatch(ctx, pending); err != nil {
			return fmt.Errorf("flushing history before terminal write: %w", err)
		}
	}

	return b.driver.Write(ctx, h)
}

func (b *BufferedDriver) flushLoop() {
	defer close(b.doneCh)
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopCh:
			return
		case <-ticker.C:
			b.flush()
		}
	}
}

func (b *BufferedDriver) flush() {
	b.flushMu.Lock()
	defer b.flushMu.Unlock()

	b.mu.Lock()
	if len(b.buf) == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.buf
	b.buf = make([]History, 0, b.batchSize)
	b.mu.Unlock()

	if err := b.driver.WriteBatch(context.Background(), batch); err != nil {
		b.log.Error("error flushing buffered history", "error", err, "count", len(batch))
	}
}

func (b *BufferedDriver) flushAsync() {
	if b.closing.Load() {
		return
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.flush()
	}()
}

func isTerminalType(t string) bool {
	return t == enums.HistoryTypeFunctionCancelled.String() ||
		t == enums.HistoryTypeFunctionCompleted.String() ||
		t == enums.HistoryTypeFunctionFailed.String()
}
