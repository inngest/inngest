package expr

import (
	"context"
	"sync"

	"github.com/google/cel-go/common/operators"
	"github.com/ohler55/ojg/jp"
)

func newNullMatcher() MatchingEngine {
	return &nullLookup{
		lock:  &sync.RWMutex{},
		paths: map[string]struct{}{},
		null:  map[string][]*StoredExpressionPart{},
		not:   map[string][]*StoredExpressionPart{},
	}
}

type nullLookup struct {
	lock *sync.RWMutex

	// paths stores all variable names as JSON paths used within the engine.
	paths map[string]struct{}

	null map[string][]*StoredExpressionPart
	not  map[string][]*StoredExpressionPart
}

func (n *nullLookup) Type() EngineType {
	return EngineTypeNullMatch
}

func (n *nullLookup) Match(ctx context.Context, data map[string]any, result *MatchResult) (err error) {
	for path := range n.paths {
		x, err := jp.ParseString(path)
		if err != nil {
			return err
		}

		res := x.Get(data)
		if len(res) == 0 {
			// This isn't present, which matches null in our overloads.  Set the
			// value to nil.
			res = []any{nil}
		}

		// XXX: This engine hasn't been updated with denied items for !=.  It needs consideration
		// in how to handle these cases appropriately.
		n.Search(ctx, path, res[0], result)
	}

	return nil
}

func (n *nullLookup) Search(ctx context.Context, variable string, input any, result *MatchResult) {
	if input == nil {
		// The input data is null, so the only items that can match are equality
		// comparisons to null.
		result.AddExprs(n.null[variable]...)
		return
	}

	result.AddExprs(n.not[variable]...)
}

func (n *nullLookup) Add(ctx context.Context, p ExpressionPart) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	varName := p.Predicate.Ident

	n.paths[varName] = struct{}{}

	// If we're comparing to null ("a" == null), we want the variable
	// to be null and should place this in the `null` map.
	//
	// Any other comparison is a not-null comparison.
	if p.Predicate.Operator == operators.Equals {
		if _, ok := n.null[varName]; !ok {
			n.null[varName] = []*StoredExpressionPart{p.ToStored()}
			return nil
		}
		n.null[varName] = append(n.null[varName], p.ToStored())
		return nil
	}

	if _, ok := n.not[varName]; !ok {
		n.not[varName] = []*StoredExpressionPart{p.ToStored()}
		return nil
	}
	n.not[varName] = append(n.not[varName], p.ToStored())
	return nil
}

func (n *nullLookup) Remove(ctx context.Context, parts []ExpressionPart) (int, error) {
	n.lock.Lock()
	defer n.lock.Unlock()

	processedCount := 0
	for _, p := range parts {
		// Check for context cancellation/timeout
		if ctx.Err() != nil {
			return processedCount, ctx.Err()
		}

		coll, ok := n.not[p.Predicate.Ident]
		if p.Predicate.Operator == operators.Equals {
			coll, ok = n.null[p.Predicate.Ident]
		}

		if !ok {
			processedCount++
			continue
		}

		for i, eval := range coll {
			if p.EqualsStored(eval) {
				coll = append(coll[:i], coll[i+1:]...)
				if p.Predicate.Operator == operators.Equals {
					n.null[p.Predicate.Ident] = coll
				} else {
					n.not[p.Predicate.Ident] = coll
				}
				break
			}
		}
		processedCount++
	}

	return processedCount, nil
}
