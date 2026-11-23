package cluster

import (
	"reflect"
	"sync"
)

// Mux routes each inbound stream by the type of its first message.  one
// listener then serves replication, the client API and the coordinator.
type Mux struct {
	mu       sync.RWMutex
	handlers map[reflect.Type]func(first Message, s Stream)
}

// NewMux returns an empty Mux.  pass its Accept to Transport.Listen.
func NewMux() *Mux {
	return &Mux{handlers: map[reflect.Type]func(first Message, s Stream){}}
}

// Handle registers h for streams whose first message has the type of
// example.  h owns the stream and closes it.
func (m *Mux) Handle(example Message, h func(first Message, s Stream)) {
	m.mu.Lock()
	m.handlers[reflect.TypeOf(example)] = h
	m.mu.Unlock()
}

// Accept reads the first message of s and calls the handler for its type.
// a stream with no handler is closed.
func (m *Mux) Accept(s Stream) {
	first, err := s.Recv()
	if err != nil {
		_ = s.Close()
		return
	}
	m.mu.RLock()
	h := m.handlers[reflect.TypeOf(first)]
	m.mu.RUnlock()
	if h == nil {
		_ = s.Close()
		return
	}
	h(first, s)
}
