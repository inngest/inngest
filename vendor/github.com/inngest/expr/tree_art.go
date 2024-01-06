package expr

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	art "github.com/plar/go-adaptive-radix-tree"
)

var (
	ErrInvalidType            = fmt.Errorf("invalid type for tree")
	ErrExpressionPartNotFound = fmt.Errorf("expression part not found")
)

func newArtTree() PredicateTree {
	return &artTree{
		lock: &sync.RWMutex{},
		Tree: art.New(),
	}
}

type artTree struct {
	lock *sync.RWMutex
	art.Tree
}

func (a *artTree) Search(ctx context.Context, input any) (*Leaf, bool) {
	var key art.Key

	switch val := input.(type) {
	case art.Key:
		key = val
	case []byte:
		key = val
	case string:
		key = artKeyFromString(val)
	}

	if len(key) == 0 {
		return nil, false
	}

	val, ok := a.Tree.Search(key)
	if !ok {
		return nil, false
	}
	return val.(*Leaf), true
}

func (a *artTree) Remove(ctx context.Context, p ExpressionPart) error {
	str, ok := p.Predicate.Literal.(string)
	if !ok {
		return ErrInvalidType
	}

	key := artKeyFromString(str)

	// Don't allow multiple gorutines to modify the tree simultaneously.
	a.lock.Lock()
	defer a.lock.Unlock()

	val, ok := a.Tree.Search(key)
	if !ok {
		return ErrExpressionPartNotFound
	}

	next := val.(*Leaf)
	// Remove the expression part from the leaf.
	for n, eval := range next.Evals {
		if p.Equals(eval) {
			next.Evals = append(next.Evals[:n], next.Evals[n+1:]...)
			a.Insert(key, next)
			return nil
		}
	}

	return ErrExpressionPartNotFound
}

func (a *artTree) Add(ctx context.Context, p ExpressionPart) error {
	str, ok := p.Predicate.Literal.(string)
	if !ok {
		return ErrInvalidType
	}

	key := artKeyFromString(str)

	// Don't allow multiple gorutines to modify the tree simultaneously.
	a.lock.Lock()
	defer a.lock.Unlock()

	val, ok := a.Tree.Search(key)
	if !ok {
		// Insert the ExpressionPart as-is.
		a.Insert(key, art.Value(&Leaf{
			Evals: []ExpressionPart{p},
		}))
		return nil
	}

	// Add the expressionpart as an expression matched by the already-existing
	// value.  Many expressions may match on the same string, eg. a user may set
	// up 3 matches for order ID "abc".  All 3 matches must be evaluated.
	next := val.(*Leaf)
	next.Evals = append(next.Evals, p)
	a.Insert(key, next)
	return nil
}

func artKeyFromString(str string) art.Key {
	// Zero-allocation string to byte conversion for speed.
	strd := unsafe.StringData(str)
	return art.Key(unsafe.Slice(strd, len(str)))

}
