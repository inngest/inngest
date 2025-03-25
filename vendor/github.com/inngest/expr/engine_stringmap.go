package expr

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/google/cel-go/common/operators"
	"github.com/ohler55/ojg/jp"
)

func newStringEqualityMatcher(concurrency int64) MatchingEngine {
	return &stringLookup{
		lock:        &sync.RWMutex{},
		vars:        map[string]struct{}{},
		equality:    variableMap{},
		inequality:  inequalityMap{},
		concurrency: concurrency,
	}
}

type variableMap map[string][]*StoredExpressionPart
type inequalityMap map[string]variableMap

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
	// equality stores all strings referenced within expressions, mapped to the expression part.
	// this performs string equality lookups.
	equality variableMap

	// inequality stores all variables referenced within inequality checks mapped to the value,
	// which is then mapped to expression parts.
	//
	// this lets us quickly map neq in a fast manner
	inequality inequalityMap

	concurrency int64
}

func (s stringLookup) Type() EngineType {
	return EngineTypeStringHash
}

func (n *stringLookup) Match(ctx context.Context, input map[string]any) ([]*StoredExpressionPart, error) {
	l := &sync.Mutex{}

	matched := []*StoredExpressionPart{}

	pool := newErrPool(errPoolOpts{concurrency: n.concurrency})

	// First, handle equality matching.
	for item := range n.vars {
		path := item
		pool.Go(func() error {
			x, err := jp.ParseString(path)
			if err != nil {
				return err
			}

			// default to an empty string
			str := ""
			if res := x.Get(input); len(res) > 0 {
				if value, ok := res[0].(string); ok {
					str = value
				}
			}

			m := n.equalitySearch(ctx, path, str)

			l.Lock()
			matched = append(matched, m...)
			l.Unlock()
			return nil
		})
	}

	// Then, iterate through the inequality matches.
	for item := range n.inequality {
		path := item
		pool.Go(func() error {
			x, err := jp.ParseString(path)
			if err != nil {
				return err
			}

			// default to an empty string
			str := ""
			if res := x.Get(input); len(res) > 0 {
				if value, ok := res[0].(string); ok {
					str = value
				}
			}

			m := n.inequalitySearch(ctx, path, str)

			l.Lock()
			matched = append(matched, m...)
			l.Unlock()
			return nil
		})
	}

	return matched, pool.Wait()
}

// Search returns all ExpressionParts which match the given input, ignoring the variable name
// entirely.
//
// Note that Search does not match inequality items.
func (n *stringLookup) Search(ctx context.Context, variable string, input any) (matched []*StoredExpressionPart) {
	str, ok := input.(string)
	if !ok {
		return nil
	}

	return n.equalitySearch(ctx, variable, str)

}

func (n *stringLookup) equalitySearch(ctx context.Context, variable string, input string) (matched []*StoredExpressionPart) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	hashedInput := n.hash(input)

	// Iterate through all matching values, and only take those expressions which match our
	// current variable name.
	filtered := make([]*StoredExpressionPart, len(n.equality[hashedInput]))
	i := 0
	for _, part := range n.equality[hashedInput] {
		if part.Ident != nil && *part.Ident != variable {
			// The variables don't match.
			continue
		}
		filtered[i] = part
		i++
	}
	filtered = filtered[0:i]

	return filtered
}

func (n *stringLookup) inequalitySearch(ctx context.Context, variable string, input string) (matched []*StoredExpressionPart) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	hashedInput := n.hash(input)

	results := []*StoredExpressionPart{}
	for value, exprs := range n.inequality[variable] {
		if value == hashedInput {
			continue
		}
		results = append(results, exprs...)
	}
	return results
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
	// Primarily, we match `$string == lit` and `$string != lit`.
	//
	// Equality operators are easy:  link the matching string to
	// expressions that are candidates.
	switch p.Predicate.Operator {
	case operators.Equals:
		n.lock.Lock()
		defer n.lock.Unlock()
		val := n.hash(p.Predicate.LiteralAsString())

		n.vars[p.Predicate.Ident] = struct{}{}

		if _, ok := n.equality[val]; !ok {
			n.equality[val] = []*StoredExpressionPart{p.ToStored()}
			return nil
		}
		n.equality[val] = append(n.equality[val], p.ToStored())

	case operators.NotEquals:
		n.lock.Lock()
		defer n.lock.Unlock()
		val := n.hash(p.Predicate.LiteralAsString())

		// First, add the variable to inequality
		if _, ok := n.inequality[p.Predicate.Ident]; !ok {
			n.inequality[p.Predicate.Ident] = variableMap{
				val: []*StoredExpressionPart{p.ToStored()},
			}
			return nil
		}

		n.inequality[p.Predicate.Ident][val] = append(n.inequality[p.Predicate.Ident][val], p.ToStored())
		return nil
	default:
		return fmt.Errorf("StringHash engines only support string equality/inequality")
	}

	return nil
}

func (n *stringLookup) Remove(ctx context.Context, p ExpressionPart) error {
	switch p.Predicate.Operator {
	case operators.Equals:
		n.lock.Lock()
		defer n.lock.Unlock()

		val := n.hash(p.Predicate.LiteralAsString())

		coll, ok := n.equality[val]
		if !ok {
			// This could not exist as there's nothing mapping this variable for
			// the given event name.
			return ErrExpressionPartNotFound
		}

		// Remove the expression part from the leaf.
		for i, eval := range coll {
			if p.EqualsStored(eval) {
				coll = append(coll[:i], coll[i+1:]...)
				n.equality[val] = coll
				return nil
			}
		}

		return ErrExpressionPartNotFound

	case operators.NotEquals:
		n.lock.Lock()
		defer n.lock.Unlock()

		val := n.hash(p.Predicate.LiteralAsString())

		// If the var isn't found, we can't remove.
		if _, ok := n.inequality[p.Predicate.Ident]; !ok {
			return ErrExpressionPartNotFound
		}

		// then merge the expression into the value that the expression has.
		if _, ok := n.inequality[p.Predicate.Ident][val]; !ok {
			return nil
		}

		for i, eval := range n.inequality[p.Predicate.Ident][val] {
			if p.EqualsStored(eval) {
				n.inequality[p.Predicate.Ident][val] = append(n.inequality[p.Predicate.Ident][val][:i], n.inequality[p.Predicate.Ident][val][i+1:]...)
				return nil
			}
		}

		return ErrExpressionPartNotFound

	default:
		return fmt.Errorf("StringHash engines only support string equality/inequality")
	}
}
