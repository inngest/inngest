package expr

import (
	"context"
	"sync"

	"github.com/google/cel-go/common/operators"
)

// TODO: Rename PredicateTrees as these may not be trees -.-
func newNullMatcher() PredicateTree {
	return &nullLookup{
		lock: &sync.RWMutex{},
		null: map[string][]ExpressionPart{},
		not:  map[string][]ExpressionPart{},
	}
}

type nullLookup struct {
	lock *sync.RWMutex
	null map[string][]ExpressionPart
	not  map[string][]ExpressionPart
}

func (n nullLookup) Add(ctx context.Context, p ExpressionPart) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	varName := p.Predicate.Ident

	// If we're comparing to null ("a" == null), we want the variable
	// to be null and should place this in the `null` map.
	//
	// Any other comparison is a not-null comparison.
	if p.Predicate.Operator == operators.Equals {
		if _, ok := n.null[varName]; !ok {
			n.null[varName] = []ExpressionPart{p}
			return nil
		}
		n.null[varName] = append(n.null[varName], p)
		return nil
	}

	if _, ok := n.not[varName]; !ok {
		n.not[varName] = []ExpressionPart{p}
		return nil
	}
	n.not[varName] = append(n.not[varName], p)
	return nil
}

func (n *nullLookup) Remove(ctx context.Context, p ExpressionPart) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	coll, ok := n.not[p.Predicate.Ident]
	if p.Predicate.Operator == operators.Equals {
		coll, ok = n.null[p.Predicate.Ident]
	}

	if !ok {
		// This could not exist as there's nothing mapping this variable for
		// the given event name.
		return ErrExpressionPartNotFound
	}

	// Remove the expression part from the leaf.
	for i, eval := range coll {
		if p.Equals(eval) {
			coll = append(coll[:i], coll[i+1:]...)
			if p.Predicate.Operator == operators.Equals {
				n.null[p.Predicate.Ident] = coll
			} else {
				n.not[p.Predicate.Ident] = coll
			}
			return nil
		}
	}

	return ErrExpressionPartNotFound
}

func (n *nullLookup) Search(ctx context.Context, variable string, input any) []ExpressionPart {
	if input == nil {
		// The input data is null, so the only items that can match are equality
		// comparisons to null.
		all := n.null[variable]
		return all
	}

	all := n.not[variable]
	return all
}
