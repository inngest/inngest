package inmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/oklog/ulid/v2"
)

// Queue is a simplistic, **non production ready** queue for processing steps
// of functions, keepign the queue in-memory with zero persistence.  It is used
// to simulate a production environment for local testing.
type Queue interface {
	// Embed the state.Manager interface for processing state items.
	state.Manager

	// Channel returns a channel which receives available jobs on the queue.
	Channel() chan QueueItem

	// Enqueue enqueues a new item for scheduling at the specific time.
	Enqueue(item QueueItem, at time.Time)

	// Pauses returns all available pauses.
	//
	// This is _not_ the smartest implementation;  most state stores should
	// return all pauses for a specific event, or for a specific run ID &
	// step ID combination.
	//
	// TODO: Create interfaces for the above methods.
	Pauses() map[uuid.UUID]state.Pause
}

type QueueItem struct {
	ID         state.Identifier
	Edge       inngest.Edge
	ErrorCount int
}

// NewStateManager returns a new in-memory queue and state manager for processing
// functions in-memory, for development and testing only.
func NewStateManager() Queue {
	return &mem{
		state:  map[ulid.ULID]state.State{},
		pauses: map[uuid.UUID]state.Pause{},
		lock:   &sync.RWMutex{},
		q:      make(chan QueueItem),
	}
}

type mem struct {
	state map[ulid.ULID]state.State

	pauses map[uuid.UUID]state.Pause

	lock *sync.RWMutex

	q chan QueueItem
}

func (m *mem) Enqueue(item QueueItem, at time.Time) {
	go func() {
		<-time.After(time.Until(at))
		m.q <- item
	}()
}

func (m *mem) Channel() chan QueueItem {
	return m.q
}

// New initializes state for a new run using the specifid ID and starting data.
func (m *mem) New(ctx context.Context, workflow inngest.Workflow, runID ulid.ULID, event map[string]any) (state.State, error) {
	state := memstate{
		workflow:   workflow,
		runID:      runID,
		workflowID: workflow.UUID,
		event:      event,
		actions:    map[string]map[string]interface{}{},
		errors:     map[string]error{},
	}

	m.lock.RLock()
	if _, ok := m.state[runID]; ok {
		return nil, fmt.Errorf("run ID already exists: %s", runID)
	}
	m.lock.RUnlock()

	m.lock.Lock()
	m.state[runID] = state
	m.lock.Unlock()

	return state, nil

}

func (m *mem) Load(ctx context.Context, i state.Identifier) (state.State, error) {
	m.lock.RLock()
	s, ok := m.state[i.RunID]
	m.lock.RUnlock()

	if ok {
		return s, nil
	}

	state := memstate{
		workflowID: i.WorkflowID,
		runID:      i.RunID,
		event:      map[string]interface{}{},
		actions:    map[string]map[string]interface{}{},
		errors:     map[string]error{},
	}

	m.lock.Lock()
	m.state[i.RunID] = state
	m.lock.Unlock()

	return state, nil
}

func (m *mem) SaveActionOutput(ctx context.Context, i state.Identifier, actionID string, data map[string]interface{}) (state.State, error) {
	s, _ := m.Load(ctx, i)

	state := s.(memstate)

	// Copy the maps so that any previous state references aren't updated.
	state.actions = copyMap(state.actions)
	state.errors = copyMap(state.errors)

	state.actions[actionID] = data
	delete(state.errors, actionID)

	m.lock.Lock()
	m.state[i.RunID] = state

	m.lock.Unlock()

	return state, nil
}

func (m *mem) SaveActionError(ctx context.Context, i state.Identifier, actionID string, err error) (state.State, error) {
	s, _ := m.Load(ctx, i)

	state := s.(memstate)

	// Copy the maps so that any previous state references aren't updated.
	state.actions = copyMap(state.actions)
	state.errors = copyMap(state.errors)

	if err == nil {
		delete(state.errors, actionID)
	} else {
		state.errors[actionID] = err
	}

	m.lock.Lock()
	m.state[i.RunID] = state
	m.lock.Unlock()

	return state, nil
}

func (m *mem) SavePause(ctx context.Context, p state.Pause) error {
	go func() {
		<-time.After(time.Until(p.Expires))
		m.lock.Lock()
		defer m.lock.Unlock()
		// If the pause exists, it can't have been consumed
		// and is therefore timed out.  Enqueue the edge as
		// we only want this to be scheduled on timeout.
		if p.OnTimeout {
			if _, ok := m.pauses[p.ID]; ok {
				m.Enqueue(QueueItem{
					ID: p.Identifier,
					Edge: inngest.Edge{
						Outgoing: p.Outgoing,
						Incoming: p.Incoming,
					},
				}, time.Now())
			}
		}
		delete(m.pauses, p.ID)
	}()

	m.lock.Lock()
	defer m.lock.Unlock()

	m.pauses[p.ID] = p
	return nil
}

func (m *mem) ConsumePause(ctx context.Context, id uuid.UUID) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if pause, ok := m.pauses[id]; !ok || pause.Expires.Before(time.Now()) {
		return state.ErrPauseNotFound
	}
	delete(m.pauses, id)
	return nil
}

func (m *mem) Pauses() map[uuid.UUID]state.Pause {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// We need to copy the pauses available such that we don't
	// return the same map to prevent data races.
	copied := copyMap(m.pauses)

	return copied
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	copied := map[K]V{}
	for k, v := range m {
		copied[k] = v
	}
	return copied
}
