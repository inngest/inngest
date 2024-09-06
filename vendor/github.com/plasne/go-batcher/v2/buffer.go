package batcher

import (
	"errors"
	"sync"
)

type ibuffer interface {
	size() uint32
	max() uint32
	top() Operation
	skip() Operation
	remove() Operation
	enqueue(Operation, bool) error
	shutdown()
}

type buffer struct {
	// WARNING: internal properties; only use the methods
	lock       *sync.Mutex
	notFull    *sync.Cond
	len        uint32
	cap        uint32
	head       *links
	tail       *links
	cursor     *links
	isShutdown bool
}

type links struct {
	prv *links
	op  Operation
	nxt *links
}

// This method creates a new buffer. The Buffer is a double-linked list holding the Operations that are
// enqueued. All methods in the buffer are threadsafe since they can be called from Batcher.Enqueue and the Batcher main processing
// loop which are commonly in different goroutines.
func newBuffer(max uint32) ibuffer {
	lock := &sync.Mutex{}
	return &buffer{
		lock:    lock,
		notFull: sync.NewCond(lock),
		cap:     max,
	}
}

// This returns the number of Operations in the buffer.
func (b *buffer) size() uint32 {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.len
}

// This returns the maximum number of Operations that can be held in the buffer.
func (b *buffer) max() uint32 {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.cap
}

// This sets the cursor position to the top of the Buffer and returns the head Operation. This method will return nil if there
// is no head Operation. Batcher's main processing loop runs on a single thread so having a single cursor is appropriate.
func (b *buffer) top() Operation {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cursor = b.head
	if b.cursor == nil {
		return nil
	}
	return b.cursor.op
}

// This advances the cursor position leaving the current record in the buffer. It returns the Operation at the new cursor
// position. This method will return nil if there are no more Operations in the Buffer.
func (b *buffer) skip() Operation {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.cursor == nil {
		return nil
	}
	b.cursor = b.cursor.nxt
	if b.cursor == nil {
		return nil
	}
	return b.cursor.op
}

// This advances the cursor position removing the current record from the buffer. It returns the Operation at the new cursor
// position. This method will return nil if there are no more Operations in the Buffer.
func (b *buffer) remove() Operation {
	b.lock.Lock()
	defer b.lock.Unlock()

	switch {
	case b.cursor == nil:
		return nil
	case b.cursor.prv != nil && b.cursor.nxt != nil:
		// cursor is at neither a head nor a tail
		b.cursor.prv.nxt = b.cursor.nxt
		b.cursor.nxt.prv = b.cursor.prv
		b.cursor = b.cursor.nxt
	case b.cursor.prv != nil:
		// cursor is the tail
		b.cursor.prv.nxt = nil
		b.tail = b.cursor.prv
		b.cursor = nil
	case b.cursor.nxt != nil:
		// cursor is the head
		b.cursor.nxt.prv = nil
		b.head = b.cursor.nxt
		b.cursor = b.cursor.nxt
	default:
		// cursor is both head and tail
		b.head = nil
		b.tail = nil
		b.cursor = nil
	}

	if b.len == 0 {
		// NOTE: There should be no way to reach this panic unless there was a coding error
		panic(errors.New("removing from empty buffer is not allowed"))
	}
	b.notFull.Signal()
	b.len--

	if b.cursor == nil {
		return nil
	}
	return b.cursor.op
}

// This allows you to add an Operation to the tail of the Buffer. If the Buffer is full and errorOnFull is false, this method
// is blocking until the Operation can be added. If the Buffer is full and errorOnFull is true, this method returns BufferFullError.
func (b *buffer) enqueue(op Operation, errorOnFull bool) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.isShutdown {
		return BufferIsShutdown
	}

	for b.len >= b.cap {
		if errorOnFull {
			return BufferFullError
		}
		b.notFull.Wait()
	}

	switch {
	case b.head == nil:
		link := &links{op: op}
		b.head = link
		b.tail = link
	case b.tail == nil:
		// NOTE: There should be no way to reach this panic unless there was a coding error
		panic(errors.New("a buffer tail was not found"))
	default:
		link := &links{prv: b.tail, op: op}
		b.tail.nxt = link
		b.tail = link
	}

	b.len++

	return nil
}

// This clears the Buffer allowing all Operations to be garbage collected. Once shutdown, it cannot be used any longer
func (b *buffer) shutdown() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.head = nil
	b.tail = nil
	b.cursor = nil
	b.len = 0
	b.isShutdown = true
}
