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
		metadata: state.Metadata{
			StartedAt: time.Now(),
		},
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

	// TODO: Return an error.
	state := memstate{
		metadata:   state.Metadata{},
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

func (m *mem) Scheduled(ctx context.Context, i state.Identifier, stepID string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	s, ok := m.state[i.RunID]
	if !ok {
		return fmt.Errorf("identifier not found")
	}

	instance := s.(memstate)
	instance.metadata.Pending++
	m.state[i.RunID] = instance

	return nil
}

func (m *mem) Finalized(ctx context.Context, i state.Identifier, stepID string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	s, ok := m.state[i.RunID]
	if !ok {
		return fmt.Errorf("identifier not found")
	}

	instance := s.(memstate)
	instance.metadata.Pending--
	m.state[i.RunID] = instance

	return nil
}

func (m *mem) SaveResponse(ctx context.Context, i state.Identifier, r state.DriverResponse, attempt int) (state.State, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	s, ok := m.state[i.RunID]
	if !ok {
		return s, fmt.Errorf("identifier not found")
	}
	instance := s.(memstate)

	// Copy the maps so that any previous state references aren't updated.
	instance.actions = copyMap(instance.actions)
	instance.errors = copyMap(instance.errors)

	if r.Err == nil {
		instance.actions[r.Step.ID] = r.Output
		delete(instance.errors, r.Step.ID)
	} else {
		instance.errors[r.Step.ID] = r.Err
	}

	if r.Final() {
		instance.metadata.Pending--
	}

	m.state[i.RunID] = instance

	return instance, nil

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

	if _, ok := m.pauses[p.ID]; ok {
		return fmt.Errorf("pause already exists")
	}

	m.pauses[p.ID] = p
	return nil
}

func (m *mem) LeasePause(ctx context.Context, id uuid.UUID) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	pause, ok := m.pauses[id]
	if !ok || pause.Expires.Before(time.Now()) {
		return state.ErrPauseNotFound
	}
	if pause.LeasedUntil != nil && time.Now().Before(*pause.LeasedUntil) {
		return state.ErrPauseLeased
	}

	lease := time.Now().Add(state.PauseLeaseDuration)
	pause.LeasedUntil = &lease
	m.pauses[id] = pause

	return nil
}

func (m *mem) PausesByEvent(ctx context.Context, eventName string) (state.PauseIterator, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	subset := []*state.Pause{}
	for _, p := range m.pauses {
		copied := p
		if p.Event != nil && *p.Event == eventName {
			subset = append(subset, &copied)
		}
	}

	i := &pauseIterator{pauses: subset}
	return i, nil
}

func (m *mem) PauseByStep(ctx context.Context, i state.Identifier, actionID string) (*state.Pause, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, p := range m.pauses {
		if p.Identifier.RunID == i.RunID && p.Outgoing == actionID {
			return &p, nil
		}
	}
	return nil, state.ErrPauseNotFound
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

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	copied := map[K]V{}
	for k, v := range m {
		copied[k] = v
	}
	return copied
}
