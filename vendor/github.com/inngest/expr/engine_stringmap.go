package expr

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/google/cel-go/common/operators"
	"github.com/ohler55/ojg/jp"
	"golang.org/x/sync/errgroup"
)

func newStringEqualityMatcher() MatchingEngine {
	return &stringLookup{
		lock:    &sync.RWMutex{},
		vars:    map[string]struct{}{},
		strings: map[string][]*StoredExpressionPart{},
	}
}

// stringLookup represents a very dumb lookup for string equality matching within
// expressions.
//
// This does nothing fancy:  it takes strings from expressions then adds them a hashmap.
// For any incoming event, we take all strings and store them in a hashmap pointing to
// the ExpressionPart they match.
//
// Note that strings are (obviuously) hashed to store in a hashmap, leading to potential
// false postivies.  Because the aggregate merging filters invalid expressions, this is
// okay:  we still evaluate potential matches at the end of filtering.
//
// Due to this, we do not care about variable names for each string.  Matching on string
// equality alone down the cost of evaluating non-matchingexpressions by orders of magnitude.
type stringLookup struct {
	lock *sync.RWMutex

	// vars stores variable names seen within expressions.
	vars map[string]struct{}
	// strings stores all strings referenced within expressions, mapped to the expression part.
	// this performs string equality lookups.
	strings map[string][]*StoredExpressionPart
}

func (s stringLookup) Type() EngineType {
	return EngineTypeStringHash
}

func (n *stringLookup) Match(ctx context.Context, input map[string]any) ([]*StoredExpressionPart, error) {
	l := &sync.Mutex{}
	found := []*StoredExpressionPart{}
	eg := errgroup.Group{}

	for item := range n.vars {
		path := item
		eg.Go(func() error {
			x, err := jp.ParseString(path)
			if err != nil {
				return err
			}

			res := x.Get(input)
			if len(res) == 0 {
				return nil
			}
			str, ok := res[0].(string)
			if !ok {
				return nil
			}

			// This matches null, nil (as null), and any non-null items.
			l.Lock()
			found = append(found, n.Search(ctx, path, str)...)
			l.Unlock()
			return nil
		})
	}

	return found, eg.Wait()
}

// Search returns all ExpressionParts which match the given input, ignoring the variable name
// entirely.
func (n *stringLookup) Search(ctx context.Context, variable string, input any) []*StoredExpressionPart {
	n.lock.RLock()
	defer n.lock.RUnlock()
	str, ok := input.(string)
	if !ok {
		return nil
	}
	return n.strings[n.hash(str)]
}

// hash hashes strings quickly via xxhash.  this provides a _somewhat_ collision-free
// lookup while reducing memory for strings.  note that internally, go maps store the
// raw key as a string, which uses extra memory.  by compressing all strings via this
// hash, memory usage grows predictably even with long strings.
func (n *stringLookup) hash(input string) string {
	ui := xxhash.Sum64String(input)
	return strconv.FormatUint(ui, 36)
}

func (n *stringLookup) Add(ctx context.Context, p ExpressionPart) error {
	if p.Predicate.Operator != operators.Equals {
		return fmt.Errorf("StringHash engines only support string equality")
	}

	n.lock.Lock()
	defer n.lock.Unlock()
	val := n.hash(p.Predicate.LiteralAsString())

	n.vars[p.Predicate.Ident] = struct{}{}

	if _, ok := n.strings[val]; !ok {
		n.strings[val] = []*StoredExpressionPart{p.ToStored()}
		return nil
	}
	n.strings[val] = append(n.strings[val], p.ToStored())

	return nil
}

func (n *stringLookup) Remove(ctx context.Context, p ExpressionPart) error {
	if p.Predicate.Operator != operators.Equals {
		return fmt.Errorf("StringHash engines only support string equality")
	}

	n.lock.Lock()
	defer n.lock.Unlock()

	val := n.hash(p.Predicate.LiteralAsString())

	coll, ok := n.strings[val]
	if !ok {
		// This could not exist as there's nothing mapping this variable for
		// the given event name.
		return ErrExpressionPartNotFound
	}

	// Remove the expression part from the leaf.
	for i, eval := range coll {
		if p.EqualsStored(eval) {
			coll = append(coll[:i], coll[i+1:]...)
			n.strings[val] = coll
			return nil
		}
	}

	return ErrExpressionPartNotFound
}
