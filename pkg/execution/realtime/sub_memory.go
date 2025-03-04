package realtime

import (
	"sync/atomic"

	"github.com/google/uuid"
)

func NewInmemorySubscription(id uuid.UUID, writer func(m Message) error) Subscription {
	return subMemory{
		id:     id,
		writer: writer,
	}
}

// subMemory represents an in-memory noop subscription
type subMemory struct {
	writeCalls  int32
	streamCalls int32
	id          uuid.UUID

	writer      func(m Message) error
	chunkWriter func(c Chunk) error
}

func (s subMemory) ID() uuid.UUID {
	return s.id
}

func (s subMemory) Protocol() string {
	return "memory"
}

func (s subMemory) WriteMessage(m Message) error {
	atomic.AddInt32(&s.writeCalls, 1)
	if s.writer != nil {
		return s.writer(m)
	}
	return nil
}

func (s subMemory) WriteChunk(c Chunk) error {
	atomic.AddInt32(&s.streamCalls, 1)
	if s.chunkWriter != nil {
		return s.chunkWriter(c)
	}
	return nil
}

func (s subMemory) SendKeepalive(m Message) error {
	return nil
}

func (s subMemory) Close() error {
	return nil
}
