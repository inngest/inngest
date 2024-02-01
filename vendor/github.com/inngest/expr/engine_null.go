package expr

import (
	"context"
	"sync"

	"github.com/google/cel-go/common/operators"
	"github.com/ohler55/ojg/jp"
	"golang.org/x/sync/errgroup"
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

func (n *nullLookup) Match(ctx context.Context, data map[string]any) ([]*StoredExpressionPart, error) {
	found := []*StoredExpressionPart{}
	eg := errgroup.Group{}

	for item := range n.paths {
		path := item
		eg.Go(func() error {
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
			// This matches null, nil (as null), and any non-null items.
			found = append(found, n.Search(ctx, path, res[0])...)
			return nil
		})
	}

	return found, eg.Wait()
}

func (n *nullLookup) Search(ctx context.Context, variable string, input any) []*StoredExpressionPart {
	if input == nil {
		// The input data is null, so the only items that can match are equality
		// comparisons to null.
		all := n.null[variable]
		return all
	}

	all := n.not[variable]
	return all
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
		if p.EqualsStored(eval) {
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
