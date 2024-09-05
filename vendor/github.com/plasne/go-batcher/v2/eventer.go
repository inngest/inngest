package batcher

import (
	"sync"

	"github.com/google/uuid"
)

type EventerBase struct {
	listenerMutex sync.RWMutex
	listeners     map[uuid.UUID]func(event string, val int, msg string, metadata interface{})
}

type Eventer interface {
	AddListener(fn func(event string, val int, msg string, metadata interface{})) uuid.UUID
	RemoveListener(id uuid.UUID)
	Emit(event string, val int, msg string, metadata interface{})
}

// You can add a listener to catch events that are raised by Batcher or a RateLimiter.
func (r *EventerBase) AddListener(fn func(event string, val int, msg string, metadata interface{})) uuid.UUID {

	// lock
	r.listenerMutex.Lock()
	defer r.listenerMutex.Unlock()

	// allocate
	if r.listeners == nil {
		r.listeners = make(map[uuid.UUID]func(event string, val int, msg string, metadata interface{}))
	}

	// add a new listener
	id := uuid.New()
	r.listeners[id] = fn

	return id
}

// If you no longer need to catch events that are raised by Batcher or a RateLimiter, you can use this method to remove the listener.
func (r *EventerBase) RemoveListener(id uuid.UUID) {

	// lock
	r.listenerMutex.Lock()
	defer r.listenerMutex.Unlock()

	// remove
	delete(r.listeners, id)

}

// To raise an event, you may emit a unique string for the event along with val, msg, and metadata as appropriate to describe the event.
func (r *EventerBase) Emit(event string, val int, msg string, metadata interface{}) {

	// lock
	r.listenerMutex.RLock()
	defer r.listenerMutex.RUnlock()

	// emit
	for _, fn := range r.listeners {
		fn(event, val, msg, metadata)
	}

}
