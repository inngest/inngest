package expr

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/cespare/xxhash/v2"
	"github.com/google/cel-go/common/operators"
	"github.com/ohler55/ojg/jp"
)

func newStringEqualityMatcher(concurrency int64) MatchingEngine {
	return &stringLookup{
		lock:       &sync.RWMutex{},
		vars:       map[string]struct{}{},
		equality:   variableMap{},
		inequality: inequalityMap{},
		// in stores all `in` operators, eg `"foo" in vars.a`.  This lets us
		// properly iterate over variables for in equaltiy matching.
		in:          variableMap{},
		concurrency: concurrency,
	}
}

type (
	variableMap   map[string][]*StoredExpressionPart
	inequalityMap map[string]variableMap
)

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

	// in stores a list of all variables mapped to `in` operators, performing lookups across arrays.
	in variableMap

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

func (n *stringLookup) Match(ctx context.Context, input map[string]any, result *MatchResult) error {
	neqOptimized := int32(0)

	// First, handle equality matching.
	pool := newErrPool(errPoolOpts{concurrency: n.concurrency})
	for item := range n.vars {
		path := item
		pool.Go(func() error {
			x, err := jp.ParseString(path)
			if err != nil {
				return err
			}

			// default to an empty string
			res := x.Get(input)
			if len(res) == 0 {
				res = []any{""}
			}

			var optimized int32
			switch val := res[0].(type) {
			case string:
				if n.equalitySearch(ctx, path, val, result) {
					atomic.AddInt32(&optimized, 1)
				}
			case []any:
				for _, item := range val {
					if n.inSearch(ctx, path, item, result) {
						atomic.AddInt32(&optimized, 1)
					}
				}
			case []string:
				for _, item := range val {
					if n.inSearch(ctx, path, item, result) {
						atomic.AddInt32(&optimized, 1)
					}
				}
			}

			if optimized > 0 {
				// Set optimized to true in every case.
				atomic.AddInt32(&neqOptimized, 1)
			}
			return nil
		})
	}
	if err := pool.Wait(); err != nil {
		return err
	}

	pool = newErrPool(errPoolOpts{concurrency: n.concurrency})
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

			n.inequalitySearch(ctx, path, str, atomic.LoadInt32(&neqOptimized) > 0, result)

			return nil
		})
	}

	return pool.Wait()
}

// Search returns all ExpressionParts which match the given input, ignoring the variable name
// entirely.
//
// Note that Search does not match inequality items.
func (n *stringLookup) Search(ctx context.Context, variable string, input any, result *MatchResult) {
	switch val := input.(type) {
	case string:
		n.equalitySearch(ctx, variable, val, result)
	case []any:
		for _, item := range val {
			n.inSearch(ctx, variable, item, result)
		}
	case []string:
		for _, item := range val {
			n.inSearch(ctx, variable, item, result)
		}
	}
}

func (n *stringLookup) equalitySearch(ctx context.Context, variable string, input string, result *MatchResult) (neqOptimized bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	hashedInput := n.hash(input)

	for _, part := range n.equality[hashedInput] {
		if part.Ident != nil && *part.Ident != variable {
			// The variables don't match.
			continue
		}
		if part.GroupID.Flag() != OptimizeNone {
			neqOptimized = true
		}
		result.Add(part.EvaluableID, part.GroupID)
	}

	return neqOptimized
}

func (n *stringLookup) inSearch(ctx context.Context, variable string, input any, result *MatchResult) (neqOptimized bool) {
	str, ok := input.(string)
	if !ok {
		return
	}

	hashedInput := n.hash(str)
	for _, part := range n.in[hashedInput] {
		if part.Ident != nil && *part.Ident != variable {
			// The variables don't match.
			continue
		}
		if part.GroupID.Flag() != OptimizeNone {
			neqOptimized = true
		}
		result.Add(part.EvaluableID, part.GroupID)
	}
	return
}

// inequalitySearch performs lookups for != matches.
func (n *stringLookup) inequalitySearch(ctx context.Context, variable string, input string, neqOptimized bool, result *MatchResult) (matched []*StoredExpressionPart) {
	if len(n.inequality[variable]) == 0 {
		return nil
	}

	n.lock.RLock()
	defer n.lock.RUnlock()

	hashedInput := n.hash(input)

	results := []*StoredExpressionPart{}
	for value, exprs := range n.inequality[variable] {
		if value == hashedInput {
			continue
		}

		if !neqOptimized {
			result.AddExprs(exprs...)
			continue
		}

		for _, expr := range exprs {
			res := result.GroupMatches(expr.EvaluableID, expr.GroupID)
			if int8(res) < int8(expr.GroupID.Flag()) {
				continue
			}
			result.AddExprs(expr)
		}
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

	case operators.In:
		// If this is an "in" operator, take the predicate's literal and ensure that we
		// check appropriately.

		switch v := p.Predicate.Literal.(type) {
		case string:
			// Assume that we're going to match an array in the event.

			n.lock.Lock()
			defer n.lock.Unlock()
			val := n.hash(v)

			n.vars[p.Predicate.Ident] = struct{}{}

			if _, ok := n.in[val]; !ok {
				n.in[val] = []*StoredExpressionPart{p.ToStored()}
				return nil
			}
			n.in[val] = append(n.in[val], p.ToStored())
		}

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
