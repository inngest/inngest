package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/event_trigger_patterns"
	"github.com/inngest/inngest/pkg/inngest"
)

// functionsCache provides a short-TTL in-memory cache for the functions
// table. It caches both the raw []*cqrs.Function rows (used by
// GetFunctions) and the parsed []inngest.Function slice (used by
// Functions). This eliminates repeated full table scans on every
// incoming event, GraphQL query, and dev-server UI poll.
type functionsCache struct {
	mu            sync.Mutex
	rawFunctions  []*cqrs.Function   // cached GetFunctions result
	rawUpdatedAt  time.Time
	functions     []inngest.Function // cached Functions (parsed) result
	updatedAt     time.Time
	ttl           time.Duration
	generation    uint64 // incremented on invalidate; prevents stale write-back
}

func (c *functionsCache) invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.rawFunctions = nil
	c.rawUpdatedAt = time.Time{}
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

// cachedGetFunctions returns all functions using the raw cache layer.
// This is the cache-aware counterpart to the direct DB call in
// wrapper.GetFunctions (cqrs.go), used by all read paths including
// GraphQL resolvers, the dev-server UI, and MCP handlers.
func (w wrapper) cachedGetFunctions(ctx context.Context) ([]*cqrs.Function, error) {
	var genAtMiss uint64
	if w.fnCache != nil && !w.noFnCache {
		w.fnCache.mu.Lock()
		if !w.fnCache.rawUpdatedAt.IsZero() && time.Since(w.fnCache.rawUpdatedAt) < w.fnCache.ttl {
			result := deepCopyFunctions(w.fnCache.rawFunctions)
			w.fnCache.mu.Unlock()
			return result, nil
		}
		genAtMiss = w.fnCache.generation
		w.fnCache.mu.Unlock()
	}

	fns, err := w.q.GetFunctions(ctx)
	if err != nil {
		return nil, err
	}

	result := domainToCQRSList(fns, domainFunction)

	if w.fnCache != nil && !w.noFnCache {
		w.fnCache.mu.Lock()
		if w.fnCache.generation == genAtMiss {
			w.fnCache.rawFunctions = deepCopyFunctions(result)
			w.fnCache.rawUpdatedAt = time.Now()
		}
		w.fnCache.mu.Unlock()
	}

	return result, nil
}

// deepCopyFunctions returns a new slice where each *cqrs.Function is a
// distinct copy, so callers cannot mutate cached structs.
func deepCopyFunctions(src []*cqrs.Function) []*cqrs.Function {
	dst := make([]*cqrs.Function, len(src))
	for i, f := range src {
		cp := *f
		if f.Config != nil {
			cp.Config = make(json.RawMessage, len(f.Config))
			copy(cp.Config, f.Config)
		}
		dst[i] = &cp
	}
	return dst
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
