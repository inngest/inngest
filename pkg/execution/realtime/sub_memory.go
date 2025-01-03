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
	calls  int32
	id     uuid.UUID
	writer func(m Message) error
}

func (s subMemory) ID() uuid.UUID {
	return s.id
}

func (s subMemory) WriteMessage(m Message) error {
	atomic.AddInt32(&s.calls, 1)
	return s.writer(m)
}

func (s subMemory) SendKeepalive() error {
	return nil
}

func (s subMemory) Close() error {
	return nil
}
