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

func newNumberMatcher(concurrency int64) MatchingEngine {
	return &numbers{
		lock: &sync.RWMutex{},

		paths:       map[string]struct{}{},
		concurrency: concurrency,

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

	concurrency int64
}

func (n numbers) Type() EngineType {
	return EngineTypeBTree
}

func (n *numbers) Match(ctx context.Context, input map[string]any) (matched []*StoredExpressionPart, err error) {
	l := &sync.Mutex{}
	matched = []*StoredExpressionPart{}

	pool := newErrPool(errPoolOpts{concurrency: n.concurrency})

	for item := range n.paths {
		path := item
		pool.Go(func() error {
			x, err := jp.ParseString(path)
			if err != nil {
				return err
			}

			res := x.Get(input)

			if len(res) == 0 {
				return nil
			}

			var val float64
			switch v := res[0].(type) {
			case int:
				val = float64(v)
			case int64:
				val = float64(v)
			case float64:
				val = v
			default:
				return nil
			}

			// This matches null, nil (as null), and any non-null items.
			l.Lock()
			found := n.Search(ctx, path, val)
			matched = append(matched, found...)
			l.Unlock()

			return nil
		})
	}

	return matched, pool.Wait()
}

// Search returns all ExpressionParts which match the given input, ignoring the variable name
// entirely.
func (n *numbers) Search(ctx context.Context, variable string, input any) (matched []*StoredExpressionPart) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	// initialize matched
	matched = []*StoredExpressionPart{}

	var val float64

	switch v := input.(type) {
	case int:
		val = float64(v)
	case int64:
		val = float64(v)
	case float64:
		val = v
	default:
		return nil
	}

	// First, find exact matches.
	if vals, _ := n.exact.Get(val); len(vals) > 0 {
		// Save memory by re-assigning vals, no need to append to an empty list
		for _, m := range vals {
			if m.Ident != nil && *m.Ident != variable {
				continue
			}
			// This is a candidatre.
			matched = append(matched, m)
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
			// This is a candidatre.
			matched = append(matched, m)
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
			// This is a candidatre.
			matched = append(matched, m)
		}
		return true
	})

	return matched
}

func (n *numbers) Add(ctx context.Context, p ExpressionPart) error {
	// If this is not equals, ignore.
	if p.Predicate.Operator == operators.NotEquals {
		return fmt.Errorf("Number engine does not support !=")
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

func (n *numbers) Remove(ctx context.Context, p ExpressionPart) error {
	// If this is not equals, ignore.
	if p.Predicate.Operator == operators.NotEquals {
		return fmt.Errorf("Number engine does not support !=")
	}

	// Add the number to the btree.
	val, err := p.Predicate.LiteralAsFloat64()
	if err != nil {
		return err
	}

	n.lock.Lock()
	defer n.lock.Unlock()

	remove := func(b *btree.Map[float64, []*StoredExpressionPart]) error {
		item, ok := b.Get(val)
		if !ok {
			return ErrExpressionPartNotFound
		}
		// Remove the expression part from the leaf.
		for i, eval := range item {
			if p.EqualsStored(eval) {
				item = append(item[:i], item[i+1:]...)
				b.Set(val, item)
				return nil
			}
		}
		return nil
	}

	var equalErr, gtErr, ltErr error

	switch p.Predicate.Operator {
	// Each of these have at least one equality match.
	case operators.Equals, operators.GreaterEquals, operators.LessEquals:
		equalErr = remove(n.exact)
	}

	// Check for >=, >, <, <= separately.

	switch p.Predicate.Operator {
	case operators.Greater, operators.GreaterEquals:
		gtErr = remove(n.gt)
	case operators.Less, operators.LessEquals:
		ltErr = remove(n.lt)
	}

	if equalErr != nil && gtErr != nil && ltErr != nil {
		return ErrExpressionPartNotFound
	}

	// At least one expr part was found.
	return nil
}
