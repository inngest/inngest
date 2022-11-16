package inmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
)

func init() {
	registration.RegisterState(func() any { return &Config{} })
}

// Config registers the configuration for the in-memory state store,
// and provides a factory for the state manager based off of the config.
type Config struct {
	l   sync.Mutex
	mem *mem
}

func (c *Config) StateName() string { return "inmemory" }

func (c *Config) Manager(ctx context.Context) (state.Manager, error) {
	c.l.Lock()
	defer c.l.Unlock()

	if c.mem == nil {
		c.mem = NewStateManager().(*mem)
	}
	return c.mem, nil
}

// NewStateManager returns a new in-memory queue and state manager for processing
// functions in-memory, for development and testing only.
func NewStateManager() state.Manager {
	return &mem{
		state:   map[string]state.State{},
		pauses:  map[uuid.UUID]state.Pause{},
		leases:  map[uuid.UUID]time.Time{},
		history: map[string][]state.History{},
		lock:    &sync.RWMutex{},
	}
}

type mem struct {
	state   map[string]state.State
	pauses  map[uuid.UUID]state.Pause
	leases  map[uuid.UUID]time.Time
	history map[string][]state.History
	lock    *sync.RWMutex
}

func (m *mem) IsComplete(ctx context.Context, id state.Identifier) (bool, error) {
	m.lock.RLock()
	s, ok := m.state[id.IdempotencyKey()]
	m.lock.RUnlock()
	if !ok {
		// TODO: Return error
		return false, nil
	}
	return s.Metadata().Pending == 0, nil
}

// New initializes state for a new run using the specifid ID and starting data.
func (m *mem) New(ctx context.Context, input state.Input) (state.State, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	s := memstate{
		metadata: state.Metadata{
			Pending:       1,
			Debugger:      input.Debugger,
			RunType:       input.RunType,
			OriginalRunID: input.OriginalRunID,
		},
		workflow:   input.Workflow,
		identifier: input.Identifier,
		event:      input.EventData,
		actions:    input.Steps,
		errors:     map[string]error{},
	}

	if _, ok := m.state[input.Identifier.IdempotencyKey()]; ok {
		return nil, state.ErrIdentifierExists
	}

	m.state[input.Identifier.IdempotencyKey()] = s

	m.setHistory(ctx, input.Identifier, state.History{
		Type:       enums.HistoryTypeFunctionStarted,
		Identifier: input.Identifier,
		CreatedAt:  time.UnixMilli(int64(input.Identifier.RunID.Time())),
	})

	return s, nil

}

func (m *mem) Load(ctx context.Context, i state.Identifier) (state.State, error) {
	m.lock.RLock()
	s, ok := m.state[i.IdempotencyKey()]
	m.lock.RUnlock()

	if ok {
		// TODO: Return an error.
		return s, nil
	}

	state := memstate{
		metadata:   state.Metadata{},
		identifier: i,
		event:      map[string]interface{}{},
		actions:    map[string]any{},
		errors:     map[string]error{},
	}

	m.lock.Lock()
	m.state[i.IdempotencyKey()] = state
	m.lock.Unlock()

	return state, nil
}

func (m *mem) Scheduled(ctx context.Context, i state.Identifier, stepID string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	s, ok := m.state[i.IdempotencyKey()]
	if !ok {
		return fmt.Errorf("identifier not found")
	}

	instance := s.(memstate)
	instance.metadata.Pending++
	m.state[i.IdempotencyKey()] = instance

	return nil
}

func (m *mem) Finalized(ctx context.Context, i state.Identifier, stepID string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	s, ok := m.state[i.IdempotencyKey()]
	if !ok {
		return fmt.Errorf("identifier not found")
	}

	instance := s.(memstate)
	instance.metadata.Pending--
	if instance.metadata.Pending == 0 {
		instance.metadata.Status = state.RunStatusCompleted
	}

	m.state[i.IdempotencyKey()] = instance

	return nil
}

func (m *mem) Cancel(ctx context.Context, i state.Identifier) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	s, ok := m.state[i.IdempotencyKey()]
	if !ok {
		return fmt.Errorf("identifier not found")
	}

	switch s.Metadata().Status {
	case state.RunStatusCompleted:
		return state.ErrFunctionComplete
	case state.RunStatusFailed:
		return state.ErrFunctionFailed
	case state.RunStatusCancelled:
		return state.ErrFunctionCancelled
	}

	instance := s.(memstate)
	instance.metadata.Status = state.RunStatusCancelled
	m.state[i.IdempotencyKey()] = instance

	return nil
}

func (m *mem) SaveResponse(ctx context.Context, i state.Identifier, r state.DriverResponse, attempt int) (state.State, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	s, ok := m.state[i.IdempotencyKey()]
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
		instance.metadata.Status = state.RunStatusFailed
	}

	m.state[i.IdempotencyKey()] = instance

	return instance, nil

}

func (m *mem) SavePause(ctx context.Context, p state.Pause) error {
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
	if !ok || pause.Expires.Time().Before(time.Now()) {
		return state.ErrPauseNotFound
	}

	lease, ok := m.leases[id]
	if ok && time.Now().Before(lease) {
		return state.ErrPauseLeased
	}

	m.leases[id] = time.Now().Add(state.PauseLeaseDuration)
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

func (m *mem) PauseByID(ctx context.Context, id uuid.UUID) (*state.Pause, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	pause, ok := m.pauses[id]
	if !ok {
		return nil, state.ErrPauseNotFound
	}

	return &pause, nil
}

func (m *mem) ConsumePause(ctx context.Context, id uuid.UUID, data any) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	pause, ok := m.pauses[id]
	if !ok {
		return state.ErrPauseNotFound
	}

	if pause.DataKey != "" {
		// Save data
		s, ok := m.state[pause.Identifier.IdempotencyKey()]
		if !ok {
			return fmt.Errorf("identifier not found")
		}
		instance := s.(memstate)
		// Copy the maps so that any previous state references aren't updated.
		instance.actions = copyMap(instance.actions)
		instance.errors = copyMap(instance.errors)
		instance.actions[pause.DataKey] = data
		m.state[pause.Identifier.IdempotencyKey()] = instance
	}

	delete(m.pauses, id)
	return nil
}

func (m *mem) History(ctx context.Context, i state.Identifier) ([]state.History, error) {
	history, ok := m.history[i.RunID.String()]
	if !ok {
		return nil, fmt.Errorf("history for run %s not found", i.RunID)
	}

	return history, nil
}

func (m *mem) Runs(ctx context.Context) ([]state.Metadata, error) {
	var metadata []state.Metadata

	m.lock.Lock()
	defer m.lock.Unlock()

	for _, s := range m.state {
		met := s.Metadata()
		id := s.RunID()

		metadata = append(metadata, state.Metadata{
			Status:        met.Status,
			Debugger:      met.Debugger,
			RunType:       met.RunType,
			OriginalRunID: &id,
		})
	}

	fmt.Printf("\n\n%#v\n\n", metadata)

	return metadata, nil
}

func (m *mem) setHistory(ctx context.Context, i state.Identifier, entry state.History) error {
	_, ok := m.history[i.RunID.String()]
	if !ok {
		m.history[i.RunID.String()] = []state.History{}
	}

	m.history[i.RunID.String()] = append(m.history[i.RunID.String()], entry)

	fmt.Printf("\n\n%#v\n\n", m.history)

	return nil
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	copied := map[K]V{}
	for k, v := range m {
		copied[k] = v
	}
	return copied
}
