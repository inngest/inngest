package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/inngest/inngest/loadtest/internal/config"
	"github.com/inngest/inngest/loadtest/internal/storage"
)

// Manager owns all in-flight runs. The API layer talks to this — not to
// Runner directly — so it can list / stop / wait and also read live counters
// without caring how runs are implemented.
type Manager struct {
	opts  Options
	store *storage.Store

	mu      sync.RWMutex
	active  map[string]*activeRun
}

type activeRun struct {
	runner *Runner
	cancel context.CancelFunc
}

// NewManager constructs a Manager wired to the given store.
func NewManager(store *storage.Store, opts Options) *Manager {
	return &Manager{opts: opts, store: store, active: map[string]*activeRun{}}
}

// StartRun allocates a run id, persists a pending row, and launches the
// runner in a detached goroutine. It returns immediately with the run id.
func (m *Manager) StartRun(cfg config.RunConfig, hostID string) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	id := newRunID()
	if err := m.store.CreateRun(id, cfg, hostID); err != nil {
		return "", fmt.Errorf("persist run: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ar := &activeRun{cancel: cancel}
	m.mu.Lock()
	m.active[id] = ar
	m.mu.Unlock()

	reg := func(r *Runner) {
		m.mu.Lock()
		ar.runner = r
		m.mu.Unlock()
	}
	dereg := func(r *Runner) {
		m.mu.Lock()
		delete(m.active, id)
		m.mu.Unlock()
	}
	go func() {
		_ = Start(ctx, id, cfg, m.opts, m.store, reg, dereg)
	}()
	return id, nil
}

// StopRun cancels an active run. Returns false if the run is not active.
func (m *Manager) StopRun(id string) bool {
	m.mu.RLock()
	ar, ok := m.active[id]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	ar.cancel()
	return true
}

// LiveStats returns the current stats for an active run, or nil if the run
// is not in progress (callers should fall back to the persisted summary).
func (m *Manager) LiveStats(id string) *LiveStats {
	m.mu.RLock()
	ar, ok := m.active[id]
	m.mu.RUnlock()
	if !ok || ar.runner == nil {
		return nil
	}
	s := ar.runner.LiveStats()
	return &s
}

// Active returns the set of currently running run IDs.
func (m *Manager) Active() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.active))
	for id := range m.active {
		out = append(out, id)
	}
	return out
}

func newRunID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
