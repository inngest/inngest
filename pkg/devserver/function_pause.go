package devserver

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/queue"
)

type functionPauses struct {
	mu        sync.RWMutex
	functions map[uuid.UUID]*functionPause
}

type functionPause struct {
	mu       sync.RWMutex
	pausedAt time.Time
}

func (p *functionPauses) state(id uuid.UUID) *functionPause {
	p.mu.RLock()
	state := p.functions[id]
	p.mu.RUnlock()
	if state != nil {
		return state
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.functions == nil {
		p.functions = make(map[uuid.UUID]*functionPause)
	}
	if state, ok := p.functions[id]; ok {
		return state
	}
	// Keep the entry so concurrent callers always use the same lock.
	state = &functionPause{}
	p.functions[id] = state
	return state
}

func (p *functionPauses) pause(id uuid.UUID) time.Time {
	state := p.state(id)
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.pausedAt.IsZero() {
		state.pausedAt = time.Now()
	}
	return state.pausedAt
}

func (p *functionPauses) get(id uuid.UUID) time.Time {
	state := p.state(id)
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.pausedAt
}

func (p *functionPauses) unpause(id uuid.UUID) {
	state := p.state(id)
	state.mu.Lock()
	defer state.mu.Unlock()
	state.pausedAt = time.Time{}
}

func (p *functionPauses) resume(id uuid.UUID, wake func() error) error {
	state := p.state(id)
	state.mu.Lock()
	defer state.mu.Unlock()

	// Readers must not see the wake before the pause is cleared.
	if err := wake(); err != nil {
		return err
	}
	state.pausedAt = time.Time{}
	return nil
}

func (p *functionPauses) partitionPaused(_ context.Context, id uuid.UUID) queue.PartitionPausedInfo {
	return queue.PartitionPausedInfo{Paused: !p.get(id).IsZero()}
}
