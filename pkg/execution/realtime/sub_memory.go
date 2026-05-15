package realtime

import (
	"encoding/json"

	"github.com/google/uuid"
)

func NewInmemorySubscription(id uuid.UUID, writer func(b []byte) error) Subscription {
	return subMemory{
		id:     id,
		writer: writer,
	}
}

// subMemory represents an in-memory subscription backed by a writer callback.
type subMemory struct {
	id     uuid.UUID
	writer func(b []byte) error
}

func (s subMemory) ID() uuid.UUID {
	return s.id
}

func (s subMemory) Protocol() string {
	return "memory"
}

func (s subMemory) Write(b []byte) error {
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
