package actionloader

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/inngest/inngestctl/inngest"
)

func NewMemoryLoader() *MemoryLoader {
	loader := &MemoryLoader{
		Actions: map[string][]inngest.ActionVersion{},
		lock:    &sync.RWMutex{},
	}
	return loader
}

// MemoryLoader is an action loader which holds in-memory references to parsed actions.
type MemoryLoader struct {
	// actions stores all parsed actions, mapped by DSN to a slice representing each
	// action version.
	Actions map[string][]inngest.ActionVersion

	lock *sync.RWMutex
}

func (l *MemoryLoader) Add(action inngest.ActionVersion) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if _, ok := l.Actions[action.DSN]; !ok {
		l.Actions[action.DSN] = []inngest.ActionVersion{action}
		return
	}
	l.Actions[action.DSN] = append(l.Actions[action.DSN], action)
	l.sortActions()
}

func (l *MemoryLoader) sortActions() {
	l.lock.Lock()
	defer l.lock.Unlock()

	for dsn, actions := range l.Actions {
		copied := actions
		sort.SliceStable(copied, func(i, j int) bool {
			a, b := copied[i], copied[j]
			return a.Version.Major >= b.Version.Major && a.Version.Minor > b.Version.Minor
		})
		l.Actions[dsn] = copied
	}
}

func (l MemoryLoader) Load(ctx context.Context, dsn string, version *inngest.VersionConstraint) (*inngest.ActionVersion, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	actions, ok := l.Actions[dsn]
	if !ok {
		return nil, fmt.Errorf("action not found: %s", dsn)
	}

	if version == nil || version.Major == nil {
		// Always use the latest version and discard minor versions.
		return &actions[0], nil
	}

	for _, a := range actions {
		if a.Version.Major != *version.Major {
			continue
		}
		if version.Minor == nil {
			// Return the latest minor from this major version, which is first
			// as the slice is sorted.
			return &a, nil
		}
		if a.Version.Minor == *version.Minor {
			return &a, nil
		}
	}

	return nil, fmt.Errorf("action not found: %s", dsn)
}
