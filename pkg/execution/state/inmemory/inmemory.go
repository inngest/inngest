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

type InmemoryLoader interface {
	state.Loader

	// Runs loads all run metadata
	Runs(ctx context.Context, eventId string) ([]state.Metadata, error)
}

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

	callbacks []state.FunctionCallback
}

// OnFunctionStatus adds a callback to be called whenever functions
// transition status.
func (m *mem) OnFunctionStatus(f state.FunctionCallback) {
	m.callbacks = append(m.callbacks, f)
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
		Data:       input.EventData,
	})

	go m.runCallbacks(ctx, input.Identifier, enums.RunStatusRunning)

	return s, nil

}

func (m *mem) Load(ctx context.Context, i state.Identifier) (state.State, error) {
	m.lock.RLock()
	s, ok := m.state[i.IdempotencyKey()]
	m.lock.RUnlock()

	if ok {
		return s, nil
	}

	return nil, fmt.Errorf("state not found with identifier: %s", i.RunID.String())
}

func (m *mem) Started(ctx context.Context, i state.Identifier, stepID string, attempt int) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.setHistory(ctx, i, state.History{
		Type:       enums.HistoryTypeStepStarted,
		Identifier: i,
		CreatedAt:  time.UnixMilli(time.Now().UnixMilli()),
		Data: state.HistoryStep{
			ID:      stepID,
			Name:    stepID,
			Attempt: attempt,
		},
	})
	return nil
}

func (m *mem) Scheduled(ctx context.Context, i state.Identifier, stepID string, attempt int, at *time.Time) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	s, ok := m.state[i.IdempotencyKey()]
	if !ok {
		return fmt.Errorf("identifier not found")
	}

	instance := s.(memstate)
	instance.metadata.Pending++
	m.state[i.IdempotencyKey()] = instance

	m.setHistory(ctx, i, state.History{
		Type:       enums.HistoryTypeStepScheduled,
		Identifier: i,
		CreatedAt:  time.UnixMilli(int64(i.RunID.Time())),
		Data: state.HistoryStep{
			ID:      stepID,
			Attempt: attempt,
			Data:    at,
		},
	})

	return nil
}

func (m *mem) Finalized(ctx context.Context, i state.Identifier, stepID string, attempt int) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	s, ok := m.state[i.IdempotencyKey()]
	if !ok {
		return fmt.Errorf("identifier not found")
	}

	instance := s.(memstate)
	instance.metadata.Pending--
	if instance.metadata.Pending == 0 && instance.metadata.Status == enums.RunStatusRunning {
		instance.metadata.Status = enums.RunStatusCompleted
		go m.runCallbacks(ctx, i, enums.RunStatusCompleted)

		m.setHistory(ctx, i, state.History{
			Type:       enums.HistoryTypeFunctionCompleted,
			Identifier: i,
			CreatedAt:  time.UnixMilli(time.Now().UnixMilli()),
		})
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
	case enums.RunStatusCompleted:
		return state.ErrFunctionComplete
	case enums.RunStatusFailed:
		return state.ErrFunctionFailed
	case enums.RunStatusCancelled:
		return state.ErrFunctionCancelled
	}

	instance := s.(memstate)
	instance.metadata.Status = enums.RunStatusCancelled
	m.state[i.IdempotencyKey()] = instance

	go m.runCallbacks(ctx, i, enums.RunStatusCancelled)

	m.setHistory(ctx, i, state.History{
		Type:       enums.HistoryTypeFunctionCancelled,
		Identifier: i,
		CreatedAt:  time.UnixMilli(time.Now().UnixMilli()),
	})

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

	now := time.UnixMilli(time.Now().UnixMilli())

	if r.Err == nil {
		instance.actions[r.Step.ID] = r.Output
		delete(instance.errors, r.Step.ID)

		m.setHistory(ctx, i, state.History{
			Type:       enums.HistoryTypeStepCompleted,
			Identifier: i,
			CreatedAt:  now,
			Data: state.HistoryStep{
				ID:      r.Step.ID,
				Name:    r.Step.Name,
				Data:    r.Output,
				Attempt: attempt,
			},
		})
	} else {
		instance.errors[r.Step.ID] = r.Err

		typ := enums.HistoryTypeStepErrored
		if r.Final() {
			typ = enums.HistoryTypeStepFailed
		}

		m.setHistory(ctx, i, state.History{
			Type:       typ,
			Identifier: i,
			CreatedAt:  now,
			Data: state.HistoryStep{
				ID:      r.Step.ID,
				Name:    r.Step.Name,
				Data:    r.Err.Error(),
				Attempt: attempt,
			},
		})
	}

	if r.Final() {
		instance.metadata.Pending--
		instance.metadata.Status = enums.RunStatusFailed
		go m.runCallbacks(ctx, i, enums.RunStatusFailed)
		m.setHistory(ctx, i, state.History{
			Type:       enums.HistoryTypeFunctionFailed,
			Identifier: i,
			CreatedAt:  now,
		})
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

	m.setHistory(ctx, p.Identifier, state.History{
		Type:       enums.HistoryTypeStepWaiting,
		Identifier: p.Identifier,
		CreatedAt:  time.UnixMilli(time.Now().UnixMilli()),
		Data: state.HistoryStepWaiting{
			EventName:  p.Event,
			Expression: p.Expression,
			ExpiryTime: time.Time(p.Expires),
		},
	})

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
	m.lock.RLock()
	defer m.lock.RUnlock()

	history, ok := m.history[i.RunID.String()]
	if !ok {
		return nil, fmt.Errorf("history for run %s not found", i.RunID)
	}

	return history, nil
}

// Returns function runs, optionally filtering to only those triggered by a
// specific event if `eventId` is provided.
func (m *mem) Runs(ctx context.Context, eventId string) ([]state.Metadata, error) {
	var metadata []state.Metadata

	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, s := range m.state {
		if eventId != "" {
			evt := s.Event()
			if evt == nil || evt["id"] != eventId {
				continue
			}
		}

		met := s.Metadata()
		id := s.RunID()

		metadata = append(metadata, state.Metadata{
			Status:        met.Status,
			Debugger:      met.Debugger,
			RunType:       met.RunType,
			OriginalRunID: &id,
			Pending:       met.Pending,
			Name:          s.Workflow().Name,
		})
	}

	return metadata, nil
}

func (m *mem) setHistory(ctx context.Context, i state.Identifier, entry state.History) {
	_, ok := m.history[i.RunID.String()]
	if !ok {
		m.history[i.RunID.String()] = []state.History{}
	}
	m.history[i.RunID.String()] = append(m.history[i.RunID.String()], entry)
}

func (m mem) runCallbacks(ctx context.Context, id state.Identifier, status enums.RunStatus) {
	for _, f := range m.callbacks {
		go func(fn state.FunctionCallback) {
			fn(ctx, id, status)
		}(f)
	}
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	copied := map[K]V{}
	for k, v := range m {
		copied[k] = v
	}
	return copied
}
