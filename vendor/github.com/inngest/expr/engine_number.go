package expr

import (
	"context"
	"fmt"
	"sync"

	// "github.com/google/btree"
	"github.com/google/cel-go/common/operators"
	"github.com/ohler55/ojg/jp"
	"github.com/tidwall/btree"
)

func newNumberMatcher() MatchingEngine {
	return &numbers{
		lock: &sync.RWMutex{},

		paths: map[string]struct{}{},

		exact: btree.NewMap[float64, []*StoredExpressionPart](64),
		gt:    btree.NewMap[float64, []*StoredExpressionPart](64),
		lt:    btree.NewMap[float64, []*StoredExpressionPart](64),
	}
}

type numbers struct {
	lock *sync.RWMutex

	// paths stores all variable names as JSON paths used within the engine.
	paths map[string]struct{}

	exact *btree.Map[float64, []*StoredExpressionPart]
	gt    *btree.Map[float64, []*StoredExpressionPart]
	lt    *btree.Map[float64, []*StoredExpressionPart]
}

func (n numbers) Type() EngineType {
	return EngineTypeBTree
}

func (n *numbers) Match(ctx context.Context, input map[string]any, result *MatchResult) (err error) {
	for path := range n.paths {
		x, err := jp.ParseString(path)
		if err != nil {
			return err
		}

		res := x.Get(input)

		if len(res) == 0 {
			continue
		}

		// This matches null, nil (as null), and any non-null items.
		n.Search(ctx, path, res[0], result)
	}

	return nil
}

// Search returns all ExpressionParts which match the given input, ignoring the variable name
// entirely.
func (n *numbers) Search(ctx context.Context, variable string, input any, result *MatchResult) {
	var val float64

	switch v := input.(type) {
	case int:
		val = float64(v)
	case int64:
		val = float64(v)
	case float64:
		val = v
	default:
		return
	}

	// First, find exact matches.
	if vals, _ := n.exact.Get(val); len(vals) > 0 {
		// Save memory by re-assigning vals, no need to append to an empty list
		for _, m := range vals {
			if m.Ident != nil && *m.Ident != variable {
				continue
			}

			// This is a candidate.
			result.AddExprs(m)
		}
	}

	// Then, find all expressions that match GT this number by walking tree
	// from beginning to this number.
	n.gt.Scan(func(n float64, matches []*StoredExpressionPart) bool {
		if n >= val {
			return false
		}

		for _, m := range matches {
			if m.Ident != nil && *m.Ident != variable {
				continue
			}
			// This is a candidate.
			result.AddExprs(m)
		}
		return true
	})

	// Then, find all expressions that match LT this number by walking tree
	// from beginning to this number.
	n.lt.Reverse(func(n float64, matches []*StoredExpressionPart) bool {
		if n <= val {
			return false
		}
		// This is a candidatre.
		for _, m := range matches {
			if m.Ident != nil && *m.Ident != variable {
				continue
			}
			// This is a candidate.
			result.AddExprs(m)
		}
		return true
	})
}

func (n *numbers) Add(ctx context.Context, p ExpressionPart) error {
	// If this is not equals, ignore.
	if p.Predicate.Operator == operators.NotEquals {
		return fmt.Errorf("number engine does not support !=")
	}

	// Add the number to the btree.
	val, err := p.Predicate.LiteralAsFloat64()
	if err != nil {
		return err
	}

	n.paths[p.Predicate.Ident] = struct{}{}

	n.lock.Lock()
	defer n.lock.Unlock()

	switch p.Predicate.Operator {
	// Each of these have at least one equality match.
	case operators.Equals, operators.GreaterEquals, operators.LessEquals:
		item, ok := n.exact.Get(val)
		if !ok {
			item = []*StoredExpressionPart{}
		}
		item = append(item, p.ToStored())
		n.exact.Set(val, item)
	}

	// Check for >=, >, <, <= separately.

	switch p.Predicate.Operator {
	case operators.Greater, operators.GreaterEquals:
		item, ok := n.gt.Get(val)
		if !ok {
			item = []*StoredExpressionPart{}
		}
		item = append(item, p.ToStored())
		n.gt.Set(val, item)
	case operators.Less, operators.LessEquals:
		item, ok := n.lt.Get(val)
		if !ok {
			item = []*StoredExpressionPart{}
		}
		item = append(item, p.ToStored())
		n.lt.Set(val, item)
	}

	return nil
}

func (n *numbers) Remove(ctx context.Context, parts []ExpressionPart) (int, error) {
	n.lock.Lock()
	defer n.lock.Unlock()

	processedCount := 0
	for _, p := range parts {
		// Check for context cancellation/timeout
		if ctx.Err() != nil {
			return processedCount, ctx.Err()
		}

		if p.Predicate.Operator == operators.NotEquals {
			processedCount++
			continue
		}

		val, err := p.Predicate.LiteralAsFloat64()
		if err != nil {
			processedCount++
			continue
		}

		remove := func(b *btree.Map[float64, []*StoredExpressionPart]) {
			item, ok := b.Get(val)
			if !ok {
				return
			}
			for i, eval := range item {
				if p.EqualsStored(eval) {
					item = append(item[:i], item[i+1:]...)
					b.Set(val, item)
					return
				}
			}
		}

		switch p.Predicate.Operator {
		case operators.Equals, operators.GreaterEquals, operators.LessEquals:
			remove(n.exact)
		}

		switch p.Predicate.Operator {
		case operators.Greater, operators.GreaterEquals:
			remove(n.gt)
		case operators.Less, operators.LessEquals:
			remove(n.lt)
		}
		processedCount++
	}

	return processedCount, nil
}
