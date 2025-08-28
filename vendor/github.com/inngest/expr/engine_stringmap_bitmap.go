package expr

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/RoaringBitmap/roaring"
	"github.com/cespare/xxhash/v2"
	"github.com/google/cel-go/common/operators"
	"github.com/ohler55/ojg/jp"
)

// bitmapStringLookup is an optimized version of stringLookup that uses Roaring Bitmaps
// for much faster set operations and reduced memory usage
type bitmapStringLookup struct {
	// Use sharded locks to reduce contention
	shards [64]struct {
		mu sync.RWMutex
		// For each field path, store bitmaps of pause IDs that match specific values
		equality   map[string]map[string]*roaring.Bitmap // fieldPath -> hashedValue -> bitmap
		inequality map[string]map[string]*roaring.Bitmap // fieldPath -> hashedValue -> bitmap  
		in         map[string]map[string]*roaring.Bitmap // fieldPath -> hashedValue -> bitmap
	}
	
	// Global tracking of all fields we've seen
	vars map[string]struct{}
	varsMu sync.RWMutex
	
	// Mapping from pause ID to stored expression parts for final lookups
	pauseIndex map[uint32]*StoredExpressionPart
	pauseIndexMu sync.RWMutex
	
	concurrency int64
	nextPauseID uint32
	idMu sync.Mutex
}

func newBitmapStringEqualityMatcher(concurrency int64) MatchingEngine {
	engine := &bitmapStringLookup{
		vars:        make(map[string]struct{}),
		pauseIndex:  make(map[uint32]*StoredExpressionPart),
		concurrency: concurrency,
	}
	
	// Initialize shards
	for i := range engine.shards {
		engine.shards[i].equality = make(map[string]map[string]*roaring.Bitmap)
		engine.shards[i].inequality = make(map[string]map[string]*roaring.Bitmap)
		engine.shards[i].in = make(map[string]map[string]*roaring.Bitmap)
	}
	
	return engine
}

func (b *bitmapStringLookup) Type() EngineType {
	return EngineTypeStringHash
}

func (b *bitmapStringLookup) getShard(key string) *struct {
	mu sync.RWMutex
	equality   map[string]map[string]*roaring.Bitmap
	inequality map[string]map[string]*roaring.Bitmap  
	in         map[string]map[string]*roaring.Bitmap
} {
	hash := xxhash.Sum64String(key)
	return &b.shards[hash%64]
}

func (b *bitmapStringLookup) getNextPauseID() uint32 {
	b.idMu.Lock()
	defer b.idMu.Unlock()
	b.nextPauseID++
	return b.nextPauseID
}

func (b *bitmapStringLookup) hash(input string) string {
	ui := xxhash.Sum64String(input)
	return strconv.FormatUint(ui, 36)
}

func (b *bitmapStringLookup) Match(ctx context.Context, input map[string]any, result *MatchResult) error {
	// Instead of doing complex bitmap operations, let's use the same logic as the original
	// but optimize the storage with bitmaps. We'll collect all matching pause IDs
	// and let the group validation logic in the main aggregator handle the filtering.
	
	b.varsMu.RLock()
	fieldPaths := make([]string, 0, len(b.vars))
	for path := range b.vars {
		fieldPaths = append(fieldPaths, path)
	}
	b.varsMu.RUnlock()
	
	// For each field path we track, check if it exists in the input and collect matches
	for _, path := range fieldPaths {
		shard := b.getShard(path)
		shard.mu.RLock()
		
		x, err := jp.ParseString(path)
		if err != nil {
			shard.mu.RUnlock()
			continue
		}
		
		res := x.Get(input)
		if len(res) == 0 {
			res = []any{""}
		}
		
		switch val := res[0].(type) {
		case string:
			hashedVal := b.hash(val)
			
			// Check equality matches
			if valueMap, exists := shard.equality[path]; exists {
				if bitmap, exists := valueMap[hashedVal]; exists {
					b.addBitmapMatches(bitmap, result)
				}
			}
			
			// Check inequality matches (all except this value)
			if valueMap, exists := shard.inequality[path]; exists {
				for value, bitmap := range valueMap {
					if value != hashedVal {
						b.addBitmapMatches(bitmap, result)
					}
				}
			}
			
		case []any:
			// Handle 'in' operations for arrays
			for _, item := range val {
				if str, ok := item.(string); ok {
					hashedVal := b.hash(str)
					if valueMap, exists := shard.in[path]; exists {
						if bitmap, exists := valueMap[hashedVal]; exists {
							b.addBitmapMatches(bitmap, result)
						}
					}
				}
			}
		case []string:
			// Handle 'in' operations for string arrays
			for _, str := range val {
				hashedVal := b.hash(str)
				if valueMap, exists := shard.in[path]; exists {
					if bitmap, exists := valueMap[hashedVal]; exists {
						b.addBitmapMatches(bitmap, result)
					}
				}
			}
		}
		
		shard.mu.RUnlock()
	}
	
	return nil
}

// addBitmapMatches converts bitmap results to MatchResult format
func (b *bitmapStringLookup) addBitmapMatches(bitmap *roaring.Bitmap, result *MatchResult) {
	b.pauseIndexMu.RLock()
	defer b.pauseIndexMu.RUnlock()
	
	for _, pauseID := range bitmap.ToArray() {
		if part, exists := b.pauseIndex[pauseID]; exists {
			result.Add(part.EvaluableID, part.GroupID)
		}
	}
}

func (b *bitmapStringLookup) Search(ctx context.Context, variable string, input any, result *MatchResult) {
	// This method is kept for interface compatibility but uses the same logic as Match
	testInput := map[string]any{variable: input}
	_ = b.Match(ctx, testInput, result) // Error is already handled in Match
}

func (b *bitmapStringLookup) Add(ctx context.Context, p ExpressionPart) error {
	// Generate a unique pause ID for this expression part
	pauseID := b.getNextPauseID()
	
	// Store the mapping from pause ID to expression part
	b.pauseIndexMu.Lock()
	b.pauseIndex[pauseID] = p.ToStored()
	b.pauseIndexMu.Unlock()
	
	// Track the variable
	b.varsMu.Lock()
	b.vars[p.Predicate.Ident] = struct{}{}
	b.varsMu.Unlock()
	
	shard := b.getShard(p.Predicate.Ident)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	
	switch p.Predicate.Operator {
	case operators.Equals:
		hashedVal := b.hash(p.Predicate.LiteralAsString())
		
		if shard.equality[p.Predicate.Ident] == nil {
			shard.equality[p.Predicate.Ident] = make(map[string]*roaring.Bitmap)
		}
		if shard.equality[p.Predicate.Ident][hashedVal] == nil {
			shard.equality[p.Predicate.Ident][hashedVal] = roaring.New()
		}
		shard.equality[p.Predicate.Ident][hashedVal].Add(pauseID)
		
	case operators.NotEquals:
		hashedVal := b.hash(p.Predicate.LiteralAsString())
		
		if shard.inequality[p.Predicate.Ident] == nil {
			shard.inequality[p.Predicate.Ident] = make(map[string]*roaring.Bitmap)
		}
		if shard.inequality[p.Predicate.Ident][hashedVal] == nil {
			shard.inequality[p.Predicate.Ident][hashedVal] = roaring.New()
		}
		shard.inequality[p.Predicate.Ident][hashedVal].Add(pauseID)
		
	case operators.In:
		if str, ok := p.Predicate.Literal.(string); ok {
			hashedVal := b.hash(str)
			
			if shard.in[p.Predicate.Ident] == nil {
				shard.in[p.Predicate.Ident] = make(map[string]*roaring.Bitmap)
			}
			if shard.in[p.Predicate.Ident][hashedVal] == nil {
				shard.in[p.Predicate.Ident][hashedVal] = roaring.New()
			}
			shard.in[p.Predicate.Ident][hashedVal].Add(pauseID)
		}
		
	default:
		return fmt.Errorf("BitmapStringHash engines only support string equality/inequality/in operations")
	}
	
	return nil
}

func (b *bitmapStringLookup) Remove(ctx context.Context, p ExpressionPart) error {
	// Find the pause ID for this expression part
	var pauseIDToRemove uint32
	var found bool
	
	b.pauseIndexMu.RLock()
	for pauseID, stored := range b.pauseIndex {
		if p.EqualsStored(stored) {
			pauseIDToRemove = pauseID
			found = true
			break
		}
	}
	b.pauseIndexMu.RUnlock()
	
	if !found {
		return ErrExpressionPartNotFound
	}
	
	// Remove from pause index
	b.pauseIndexMu.Lock()
	delete(b.pauseIndex, pauseIDToRemove)
	b.pauseIndexMu.Unlock()
	
	shard := b.getShard(p.Predicate.Ident)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	
	switch p.Predicate.Operator {
	case operators.Equals:
		hashedVal := b.hash(p.Predicate.LiteralAsString())
		if valueMap, exists := shard.equality[p.Predicate.Ident]; exists {
			if bitmap, exists := valueMap[hashedVal]; exists {
				bitmap.Remove(pauseIDToRemove)
				if bitmap.IsEmpty() {
					delete(valueMap, hashedVal)
				}
			}
		}
		
	case operators.NotEquals:
		hashedVal := b.hash(p.Predicate.LiteralAsString())
		if valueMap, exists := shard.inequality[p.Predicate.Ident]; exists {
			if bitmap, exists := valueMap[hashedVal]; exists {
				bitmap.Remove(pauseIDToRemove)
				if bitmap.IsEmpty() {
					delete(valueMap, hashedVal)
				}
			}
		}
		
	case operators.In:
		if str, ok := p.Predicate.Literal.(string); ok {
			hashedVal := b.hash(str)
			if valueMap, exists := shard.in[p.Predicate.Ident]; exists {
				if bitmap, exists := valueMap[hashedVal]; exists {
					bitmap.Remove(pauseIDToRemove)
					if bitmap.IsEmpty() {
						delete(valueMap, hashedVal)
					}
				}
			}
		}
		
	default:
		return fmt.Errorf("BitmapStringHash engines only support string equality/inequality/in operations")
	}
	
	return nil
}