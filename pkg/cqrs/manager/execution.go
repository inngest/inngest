package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/event_trigger_patterns"
	"github.com/inngest/inngest/pkg/inngest"
)

// functionsCache provides a short-TTL in-memory cache for the parsed
// []inngest.Function slice returned by Functions(). This eliminates
// repeated full table scans of the functions table on every incoming event.
type functionsCache struct {
	mu         sync.Mutex
	functions  []inngest.Function
	updatedAt  time.Time
	ttl        time.Duration
	generation uint64 // incremented on invalidate; prevents stale write-back
}

func (c *functionsCache) invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.functions = nil
	c.updatedAt = time.Time{}
	c.generation++
	c.mu.Unlock()
}

// invalidateFnCache clears the functions cache after a successful mutation.
// For transactional wrappers (noFnCache == true), it defers invalidation
// to Commit() by marking fnMutated, avoiding a race where concurrent
// Functions() callers repopulate the cache with pre-commit data.
func (w wrapper) invalidateFnCache() {
	if w.noFnCache {
		if w.fnMutated != nil {
			*w.fnMutated = true
		}
		return
	}
	w.fnCache.invalidate()
}

// Functions returns all functions as inngest functions, using a short-lived
// in-memory cache to avoid repeated full table scans.
func (w wrapper) Functions(ctx context.Context) ([]inngest.Function, error) {
	var genAtMiss uint64
	if w.fnCache != nil && !w.noFnCache {
		w.fnCache.mu.Lock()
		if !w.fnCache.updatedAt.IsZero() && time.Since(w.fnCache.updatedAt) < w.fnCache.ttl {
			result := slices.Clone(w.fnCache.functions)
			w.fnCache.mu.Unlock()
			return result, nil
		}
		genAtMiss = w.fnCache.generation
		w.fnCache.mu.Unlock()
	}

	all, err := w.GetFunctions(ctx)
	if err != nil {
		return nil, err
	}

	funcs := make([]inngest.Function, len(all))
	for n, i := range all {
		f := inngest.Function{}
		if err := json.Unmarshal([]byte(i.Config), &f); err != nil {
			return nil, fmt.Errorf("failed to unmarshal function config for %s: %w", i.ID, err)
		}
		funcs[n] = f
	}

	if w.fnCache != nil && !w.noFnCache {
		w.fnCache.mu.Lock()
		if w.fnCache.generation == genAtMiss {
			w.fnCache.functions = slices.Clone(funcs)
			w.fnCache.updatedAt = time.Now()
		}
		w.fnCache.mu.Unlock()
	}

	return funcs, nil
}

// FunctionsScheduled returns all scheduled functions available.
func (w wrapper) FunctionsScheduled(ctx context.Context) ([]inngest.Function, error) {
	// TODO: Make less naive by storing triggers and caching.
	fns, err := w.Functions(ctx)
	if err != nil {
		return nil, err
	}
	all := []inngest.Function{}
	for _, fn := range fns {
		for _, t := range fn.Triggers {
			if t.CronTrigger != nil {
				all = append(all, fn)
				break
			}
		}
	}
	return all, nil
}

// FunctionsByTrigger returns functions for the given trigger by event name.
func (w wrapper) FunctionsByTrigger(ctx context.Context, eventName string) ([]inngest.Function, error) {
	// TODO: Make less naive by storing triggers and caching.
	fns, err := w.Functions(ctx)
	if err != nil {
		return nil, err
	}

	// Generate matching patterns once for efficient trigger matching
	matchingPatterns := event_trigger_patterns.GenerateMatchingPatterns(eventName)

	all := []inngest.Function{}
	for _, fn := range fns {
		for _, t := range fn.Triggers {
			if t.EventTrigger != nil && t.EventTrigger.MatchesAnyPattern(matchingPatterns) {
				all = append(all, fn)
				break
			}
		}
	}
	return all, nil
}
