package realtime

import (
	"encoding/json"
	"sync/atomic"

	"github.com/google/uuid"
)

func NewInmemorySubscription(id uuid.UUID, writer func(b []byte) error) Subscription {
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

	writer func(b []byte) error
}

func (s subMemory) ID() uuid.UUID {
	return s.id
}

func (s subMemory) Protocol() string {
	return "memory"
}

func (s subMemory) Write(b []byte) error {
	atomic.AddInt32(&s.writeCalls, 1)
	if s.writer != nil {
		return s.writer(b)
	}
	return nil
}

func (s subMemory) WriteMessage(m Message) error {
	byt, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return s.Write(byt)
}

func (s subMemory) WriteChunk(c Chunk) error {
	atomic.AddInt32(&s.streamCalls, 1)
	byt, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.Write(byt)
}

func (s subMemory) SendKeepalive(m Message) error {
	return nil
}

func (s subMemory) Close() error {
	return nil
}
