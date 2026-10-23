package devserver

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/queue"
)

type functionPauses struct {
	mu       sync.RWMutex
	pausedAt map[uuid.UUID]time.Time
}

func (p *functionPauses) pause(id uuid.UUID) time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	if at, ok := p.pausedAt[id]; ok {
		return at
	}
	at := time.Now()
	p.pausedAt[id] = at
	return at
}

func (p *functionPauses) get(id uuid.UUID) time.Time {
	p.mu.RLock()
	at := p.pausedAt[id]
	p.mu.RUnlock()
	return at
}

func (p *functionPauses) unpause(id uuid.UUID) {
	p.mu.Lock()
	delete(p.pausedAt, id)
	p.mu.Unlock()
}

func (p *functionPauses) partitionPaused(_ context.Context, id uuid.UUID) queue.PartitionPausedInfo {
	return queue.PartitionPausedInfo{Paused: !p.get(id).IsZero()}
}
