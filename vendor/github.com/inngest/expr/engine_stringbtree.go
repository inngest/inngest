package expr

import (
	"context"
	"sync"

	"github.com/google/cel-go/common/operators"
	"github.com/ohler55/ojg/jp"
	"github.com/tidwall/btree"
	"slices"
)

func newStringBTreeMatcher() MatchingEngine {
	return &stringBTree{
		lock:  &sync.RWMutex{},
		paths: map[string]int{},
		tree:  btree.NewMap[string, rangeNode](64),
	}
}

// rangeNode holds the three predicate slices for a single threshold key.
// All three live in the same btree node so one cache-line fetch covers them all.
type rangeNode struct {
	exact []*StoredExpressionPart // >= and <= (equality case)
	gt    []*StoredExpressionPart // > and >=
	lt    []*StoredExpressionPart // < and <=
}

// stringBTree matches string range predicates (<, >, <=, >=) using a single B-tree.
//
// For a stored threshold t and incoming event value v:
//   - exact fires when v == t  (covers the equality case of >= and <=)
//   - gt    fires when v > t   (gt scan stops at first threshold >= v)
//   - lt    fires when v < t   (lt reverse scan stops at first threshold <= v)
//
// A >= expression is stored in both exact and gt; a > expression only in gt.
// This prevents double-counting: exact fires on equality, gt's stop condition
// (threshold >= v) means it never fires on equality.
type stringBTree struct {
	lock  *sync.RWMutex
	paths map[string]int // ident -> count of stored ExpressionParts; deleted when 0
	tree  *btree.Map[string, rangeNode]
}

func (s *stringBTree) Type() EngineType { return EngineTypeStringBTree }

func (s *stringBTree) Match(ctx context.Context, input map[string]any, result *MatchResult) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for path := range s.paths {
		x, err := jp.ParseString(path)
		if err != nil {
			return err
		}
		res := x.Get(input)
		if len(res) == 0 {
			continue
		}
		val, ok := res[0].(string)
		if !ok {
			continue
		}
		s.search(path, val, result)
	}
	return nil
}

func (s *stringBTree) Search(ctx context.Context, variable string, input any, result *MatchResult) {
	val, ok := input.(string)
	if !ok {
		return
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	s.search(variable, val, result)
}

// search is the lock-free inner implementation; callers must hold s.lock.RLock.
func (s *stringBTree) search(variable string, val string, result *MatchResult) {
	if node, ok := s.tree.Get(val); ok {
		for _, m := range node.exact {
			if m.Ident != nil && *m.Ident != variable {
				continue
			}
			result.AddExprs(m)
		}
	}

	s.tree.Scan(func(threshold string, node rangeNode) bool {
		if threshold >= val {
			return false
		}
		for _, m := range node.gt {
			if m.Ident != nil && *m.Ident != variable {
				continue
			}
			result.AddExprs(m)
		}
		return true
	})

	s.tree.Reverse(func(threshold string, node rangeNode) bool {
		if threshold <= val {
			return false
		}
		for _, m := range node.lt {
			if m.Ident != nil && *m.Ident != variable {
				continue
			}
			result.AddExprs(m)
		}
		return true
	})
}

func (s *stringBTree) Add(ctx context.Context, p ExpressionPart) error {
	val := p.Predicate.LiteralAsString()
	stored := p.ToStored()

	s.lock.Lock()
	defer s.lock.Unlock()

	s.paths[p.Predicate.Ident]++

	node, _ := s.tree.Get(val)
	if p.Predicate.Operator == operators.GreaterEquals || p.Predicate.Operator == operators.LessEquals {
		node.exact = append(node.exact, stored)
	}
	switch p.Predicate.Operator {
	case operators.Greater, operators.GreaterEquals:
		node.gt = append(node.gt, stored)
	case operators.Less, operators.LessEquals:
		node.lt = append(node.lt, stored)
	}
	s.tree.Set(val, node)
	return nil
}

func (s *stringBTree) Remove(ctx context.Context, parts []ExpressionPart) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	removeFrom := func(slice []*StoredExpressionPart, p ExpressionPart) ([]*StoredExpressionPart, bool) {
		for i, eval := range slice {
			if p.EqualsStored(eval) {
				return slices.Delete(slice, i, i+1), true
			}
		}
		return slice, false
	}

	processedCount := 0
	for _, p := range parts {
		if ctx.Err() != nil {
			return processedCount, ctx.Err()
		}
		val := p.Predicate.LiteralAsString()
		node, ok := s.tree.Get(val)
		if !ok {
			processedCount++
			continue
		}
		var removed bool
		if p.Predicate.Operator == operators.GreaterEquals || p.Predicate.Operator == operators.LessEquals {
			var ok bool
			node.exact, ok = removeFrom(node.exact, p)
			removed = removed || ok
		}
		switch p.Predicate.Operator {
		case operators.Greater, operators.GreaterEquals:
			var ok bool
			node.gt, ok = removeFrom(node.gt, p)
			removed = removed || ok
		case operators.Less, operators.LessEquals:
			var ok bool
			node.lt, ok = removeFrom(node.lt, p)
			removed = removed || ok
		}
		if len(node.exact) == 0 && len(node.gt) == 0 && len(node.lt) == 0 {
			s.tree.Delete(val)
		} else {
			s.tree.Set(val, node)
		}
		if removed {
			s.paths[p.Predicate.Ident]--
			if s.paths[p.Predicate.Ident] == 0 {
				delete(s.paths, p.Predicate.Ident)
			}
		}
		processedCount++
	}
	return processedCount, nil
}
