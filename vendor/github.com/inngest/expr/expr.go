package expr

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/cel-go/common/operators"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

var (
	ErrEvaluableNotFound      = fmt.Errorf("Evaluable instance not found in aggregator")
	ErrInvalidType            = fmt.Errorf("invalid type for tree")
	ErrExpressionPartNotFound = fmt.Errorf("expression part not found")
)

// errEngineUnimplemented is used while we develop the aggregate tree library when trees
// are not yet implemented.
var errEngineUnimplemented = fmt.Errorf("tree type unimplemented")

// ExpressionEvaluator is a function which evalues an expression given input data, returning
// a boolean and error.
type ExpressionEvaluator func(ctx context.Context, e Evaluable, input map[string]any) (bool, error)

// EvaluableLoader returns one or more evaluable items given IDs.
type EvaluableLoader func(ctx context.Context, evaluableIDs ...uuid.UUID) ([]Evaluable, error)

// AggregateEvaluator represents a group of expressions that must be evaluated for a single
// event received.
//
// An AggregateEvaluator instance exists for every event name being matched.
type AggregateEvaluator interface {
	// Add adds an expression to the tree evaluator.  This returns the ratio
	// of aggregate to slow parts in the expression, or an error if there was an
	// issue.
	//
	// Purely aggregateable expressions have a ratio of 1.
	// Mixed expressions return the ratio of fast:slow expressions, as a float.
	// Slow, non-aggregateable expressions return 0.
	Add(ctx context.Context, eval Evaluable) (float64, error)

	// Remove removes an expression from the aggregate evaluator
	Remove(ctx context.Context, eval Evaluable) error

	// Evaluate checks input data against all exrpesssions in the aggregate in an optimal
	// manner, only evaluating expressions when necessary (based off of tree matching).
	//
	// Note that any expressions added that cannot be evaluated optimally by trees
	// are evaluated every time this function is called.
	//
	// Evaluate returns all matching Evaluables, plus the total number of evaluations
	// executed.
	Evaluate(ctx context.Context, data map[string]any) ([]Evaluable, int32, error)

	// AggregateMatch returns all expression parts which are evaluable given the input data.
	AggregateMatch(ctx context.Context, data map[string]any) ([]*StoredExpressionPart, error)

	// Len returns the total number of aggregateable and constantly matched expressions
	// stored in the evaluator.
	Len() int

	// FastLen returns the number of expressions being matched by aggregated trees.
	FastLen() int

	// MixedLen returns the number of expressions being matched by aggregated trees.
	MixedLen() int

	// SlowLen returns the total number of expressions that must constantly
	// be matched due to non-aggregateable clauses in their expressions.
	SlowLen() int
}

func NewAggregateEvaluator(
	parser TreeParser,
	eval ExpressionEvaluator,
	evalLoader EvaluableLoader,
	concurrency int64,
) AggregateEvaluator {
	if concurrency <= 0 {
		concurrency = 1
	}

	return &aggregator{
		eval:   eval,
		parser: parser,
		loader: evalLoader,
		sem:    semaphore.NewWeighted(concurrency),
		engines: map[EngineType]MatchingEngine{
			EngineTypeStringHash: newStringEqualityMatcher(),
			EngineTypeNullMatch:  newNullMatcher(),
			EngineTypeBTree:      newNumberMatcher(),
		},
		lock:  &sync.RWMutex{},
		mixed: map[uuid.UUID]struct{}{},
	}
}

type aggregator struct {
	eval   ExpressionEvaluator
	parser TreeParser
	loader EvaluableLoader

	// engines records all engines
	engines map[EngineType]MatchingEngine

	sem *semaphore.Weighted

	// lock prevents concurrent updates of data
	lock *sync.RWMutex

	// fastLen stores the current len of purely aggregable expressions.
	fastLen int32

	// mixed stores the current len of mixed aggregable expressions,
	// eg "foo == '1' && bar != '1'".  This is becasue != isn't aggregateable,
	// but the first `==` is used as a prefilter.
	//
	// This stores all evaluable IDs for fast lookup with Evaluable.
	mixed map[uuid.UUID]struct{}

	// constants tracks evaluable IDs that must always be evaluated, due to
	// the expression containing non-aggregateable clauses.
	constants []uuid.UUID
}

// Len returns the total number of aggregateable and constantly matched expressions
// stored in the evaluator.
func (a *aggregator) Len() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return int(a.fastLen) + len(a.mixed) + len(a.constants)
}

// FastLen returns the number of expressions being matched by aggregated trees.
func (a *aggregator) FastLen() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return int(a.fastLen)
}

// MixedLen returns the number of expressions being matched by aggregated trees.
func (a *aggregator) MixedLen() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return len(a.mixed)
}

// SlowLen returns the total number of expressions that must constantly
// be matched due to non-aggregateable clauses in their expressions.
func (a *aggregator) SlowLen() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return len(a.constants)
}

func (a *aggregator) Evaluate(ctx context.Context, data map[string]any) ([]Evaluable, int32, error) {
	var (
		err     error
		matched = int32(0)
		result  = []Evaluable{}
		s       sync.Mutex
	)

	// TODO: Concurrently match constant expressions using a semaphore for capacity.
	// Match constant expressions always.
	a.lock.RLock()
	constantEvals, err := a.loader(ctx, a.constants...)
	a.lock.RUnlock()
	if err != nil {
		return nil, 0, err
	}

	eg := errgroup.Group{}
	for _, item := range constantEvals {
		if err := a.sem.Acquire(ctx, 1); err != nil {
			return result, matched, err
		}

		expr := item
		eg.Go(func() error {
			defer a.sem.Release(1)
			defer func() {
				if r := recover(); r != nil {
					s.Lock()
					err = errors.Join(err, fmt.Errorf("recovered from panic in evaluate: %v", r))
					s.Unlock()
				}
			}()

			atomic.AddInt32(&matched, 1)

			if expr.GetExpression() == "" {
				s.Lock()
				result = append(result, expr)
				s.Unlock()
				return nil
			}

			// NOTE: We don't need to add lifted expression variables,
			// because match.Parsed.Evaluable() returns the original expression
			// string.
			ok, evalerr := a.eval(ctx, expr, data)
			if evalerr != nil {
				return evalerr
			}
			if ok {
				s.Lock()
				result = append(result, expr)
				s.Unlock()
			}
			return nil
		})
	}

	if werr := eg.Wait(); werr != nil {
		err = errors.Join(err, werr)
	}

	matches, merr := a.AggregateMatch(ctx, data)
	if merr != nil {
		err = errors.Join(err, merr)
	}

	// Load all evaluable instances directly.
	uuids := make([]uuid.UUID, len(matches))
	for n, m := range matches {
		uuids[n] = m.Parsed.EvaluableID
	}
	evaluables, lerr := a.loader(ctx, uuids...)
	if err != nil {
		err = errors.Join(err, lerr)
	}

	// Each match here is a potential success.  When other trees and operators which are walkable
	// are added (eg. >= operators on strings), ensure that we find the correct number of matches
	// for each group ID and then skip evaluating expressions if the number of matches is <= the group
	// ID's length.
	seenMu := &sync.Mutex{}
	seen := map[uuid.UUID]struct{}{}

	eg = errgroup.Group{}
	for _, match := range evaluables {
		if err := a.sem.Acquire(ctx, 1); err != nil {
			return result, matched, err
		}

		expr := match
		eg.Go(func() error {
			defer a.sem.Release(1)
			defer func() {
				if r := recover(); r != nil {
					s.Lock()
					err = errors.Join(err, fmt.Errorf("recovered from panic in evaluate: %v", r))
					s.Unlock()
				}
			}()

			seenMu.Lock()
			if _, ok := seen[expr.GetID()]; ok {
				seenMu.Unlock()
				return nil
			} else {
				seen[expr.GetID()] = struct{}{}
				seenMu.Unlock()
			}

			atomic.AddInt32(&matched, 1)

			// NOTE: We don't need to add lifted expression variables,
			// because match.Parsed.Evaluable() returns the original expression
			// string.
			ok, evalerr := a.eval(ctx, expr, data)

			if evalerr != nil {
				return evalerr
			}
			if ok {
				s.Lock()
				result = append(result, expr)
				s.Unlock()
			}
			return nil
		})
	}

	if werr := eg.Wait(); werr != nil {
		err = errors.Join(err, werr)
	}

	return result, matched, err
}

// AggregateMatch attempts to match incoming data to all PredicateTrees, resulting in a selection
// of parts of an expression that have matched.
func (a *aggregator) AggregateMatch(ctx context.Context, data map[string]any) ([]*StoredExpressionPart, error) {
	result := []*StoredExpressionPart{}

	a.lock.RLock()
	defer a.lock.RUnlock()

	// Each match here is a potential success.  Ensure that we find the correct number of matches
	// for each group ID and then skip evaluating expressions if the number of matches is <= the group
	// ID's length.  For example, (A && B && C) is a single group ID and must have a count >= 3,
	// else we know a required comparason did not match.
	//
	// Note that having a count >= the group ID value does not guarantee that the expression is valid.
	counts := map[groupID]int{}
	// Store all expression parts per group ID for returning.
	found := map[groupID][]*StoredExpressionPart{}

	for _, engine := range a.engines {
		matched, err := engine.Match(ctx, data)
		if err != nil {
			return nil, err
		}

		// Add all found items from the engine to the above list.
		for _, eval := range matched {
			counts[eval.GroupID] += 1
			if _, ok := found[eval.GroupID]; !ok {
				found[eval.GroupID] = []*StoredExpressionPart{}
			}
			found[eval.GroupID] = append(found[eval.GroupID], eval)
		}
	}

	// Validate that groups meet the minimum size.
	for groupID, matchingCount := range counts {
		requiredSize := int(groupID.Size()) // The total req size from the group ID

		if matchingCount >= requiredSize {
			// The matching count met the group size;  all results are safe.
			result = append(result, found[groupID]...)
			continue
		}

		// If this is a partial eval, always add it if there's a match for now.

		// The GroupID required more comparisons to equate to true than
		// we had, so this could never evaluate to true.  Skip this.
		//
		// NOTE: We currently don't add items with OR predicates to the
		// matching engine, so we cannot use group sizes if the expr part
		// has an OR.
		for _, i := range found[groupID] {
			if _, ok := a.mixed[i.Parsed.EvaluableID]; ok {
				// for now, mark this as viable as it had an OR
				result = append(result, i)
			}

			if len(i.Parsed.Root.Ors) > 0 {
				// for now, mark this as viable as it had an OR
				result = append(result, i)
			}
		}
	}

	return result, nil
}

// Add adds an expression to the tree evaluator.  This returns the ratio
// of aggregate to slow parts in the expression, or an error if there was an
// issue.
//
// Purely aggregateable expressions have a ratio of 1.
// Mixed expressions return the ratio of fast:slow expressions, as a float.
// Slow, non-aggregateable expressions return 0.
func (a *aggregator) Add(ctx context.Context, eval Evaluable) (float64, error) {
	// parse the expression using our tree parser.
	parsed, err := a.parser.Parse(ctx, eval)
	if err != nil {
		return -1, err
	}

	if eval.GetExpression() == "" || parsed.HasMacros {
		// This is an empty expression which always matches.
		a.lock.Lock()
		a.constants = append(a.constants, parsed.EvaluableID)
		a.lock.Unlock()
		return -1, nil
	}

	stats := &exprAggregateStats{}
	for _, g := range parsed.RootGroups() {
		s, err := a.iterGroup(ctx, g, parsed, a.addNode)

		if err != nil {
			// This is the first time we're seeing a non-aggregateable
			// group, so add it to the constants list and don't do anything else.
			a.lock.Lock()
			a.constants = append(a.constants, parsed.EvaluableID)
			a.lock.Unlock()
			return -1, err
		}

		stats.Merge(s)
	}

	if stats.Fast() == 0 {
		// This is a non-aggregateable, slow expression.
		// Add it to the constants list and don't do anything else.
		a.lock.Lock()
		a.constants = append(a.constants, parsed.EvaluableID)
		a.lock.Unlock()
		return stats.Ratio(), err
	}

	if stats.Slow() == 0 {
		// This is a purely aggregateable expression.
		atomic.AddInt32(&a.fastLen, 1)
		return stats.Ratio(), err
	}

	a.lock.Lock()
	a.mixed[parsed.EvaluableID] = struct{}{}
	a.lock.Unlock()

	return stats.Ratio(), err
}

func (a *aggregator) Remove(ctx context.Context, eval Evaluable) error {
	if eval.GetExpression() == "" {
		return a.removeConstantEvaluable(ctx, eval)
	}

	// parse the expression using our tree parser.
	parsed, err := a.parser.Parse(ctx, eval)
	if err != nil {
		return err
	}

	stats := &exprAggregateStats{}

	for _, g := range parsed.RootGroups() {
		s, err := a.iterGroup(ctx, g, parsed, a.removeNode)
		if errors.Is(err, ErrExpressionPartNotFound) {
			return ErrEvaluableNotFound
		}

		if err != nil {
			_ = a.removeConstantEvaluable(ctx, eval)
			return err
		}
		stats.Merge(s)
	}

	if stats.Fast() == 0 {
		// This is a non-aggregateable, slow expression.
		if err := a.removeConstantEvaluable(ctx, eval); err != nil {
			return err
		}
		return nil
	}

	if stats.Slow() == 0 {
		// This is a purely aggregateable expression.
		atomic.AddInt32(&a.fastLen, -1)
		return nil
	}

	a.lock.Lock()
	delete(a.mixed, parsed.EvaluableID)
	a.lock.Unlock()

	return nil
}

func (a *aggregator) removeConstantEvaluable(_ context.Context, eval Evaluable) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	// Find the index of the evaluable in constants and yank out.
	idx := -1
	for n, item := range a.constants {
		if item == eval.GetID() {
			idx = n
			break
		}
	}
	if idx == -1 {
		return ErrEvaluableNotFound
	}

	a.constants = append(a.constants[:idx], a.constants[idx+1:]...)
	return nil
}

type exprAggregateStats [2]int

// Fast returns the number of aggregateable predicates in the iterated expr
func (e *exprAggregateStats) Fast() int {
	return e[0]
}

// Slow returns the number of non-aggregateable predicates in the iterated expr
func (e *exprAggregateStats) Slow() int {
	return e[1]
}

func (e *exprAggregateStats) AddFast() {
	e[0] += 1
}

func (e *exprAggregateStats) AddSlow() {
	e[1] += 1
}

func (e *exprAggregateStats) Merge(other exprAggregateStats) {
	e[0] += other[0]
	e[1] += other[1]
}

// Ratio returns the ratio of fast to slow expressions as a float, eg. 9 fast
// aggregateable parts and 1 slow part returns a ratio of 0.9.
func (e *exprAggregateStats) Ratio() float64 {
	if e[0] == 0 && e[1] == 0 {
		// Failure.
		return -1
	}

	if e[1] == 0 {
		// Always fast, return 1
		return 1
	}

	if e[0] == 0 {
		// Always slow, return 0
		return 0
	}

	// return ratio of fast:slow
	return float64(e[0]) / (float64(e[0]) + float64(e[1]))
}

// iterGroup iterates the entire expression, returning statistics on how "aggregateable" the expression is
func (a *aggregator) iterGroup(ctx context.Context, node *Node, parsed *ParsedExpression, op nodeOp) (exprAggregateStats, error) {
	stats := &exprAggregateStats{}

	// It's possible that if there are additional branches, don't bother to add this to the aggregate tree.
	// Mark this as a non-exhaustive addition and skip immediately.
	if len(node.Ands) > 0 {
		for _, n := range node.Ands {
			if !n.HasPredicate() || len(n.Ors) > 0 {
				// Don't handle sub-branching for now.
				// TODO: Recursively iterate.
				stats.AddSlow()
				continue
			}
		}
	}

	// XXX: Here we must add the OR groups to make group IDs a success.

	all := node.Ands
	if node.Predicate != nil {
		if !isAggregateable(node) {
			stats.AddSlow()
		} else {
			// Merge all of the nodes together and check whether each node is aggregateable.
			all = append(node.Ands, node)
		}
	}

	// Iterate through and add every predicate to each engine.
	for _, n := range all {
		err := op(ctx, n, parsed)

		switch {
		case err == nil:
			// This is okay.
			stats.AddFast()
			continue
		case errors.Is(err, errEngineUnimplemented):
			// Not yet added to aggregator
			stats.AddSlow()
			continue
		default:
			// Some other error.
			stats.AddSlow()
			continue
		}
	}

	return *stats, nil
}

func engineType(p Predicate) EngineType {
	// switch on type of literal AND operator type.  int64/float64 literals require
	// btrees, texts require ARTs, and so on.
	switch p.Literal.(type) {
	case int, int64, float64:
		if p.Operator == operators.NotEquals {
			// StringHash is only used for matching on equality.
			return EngineTypeNone
		}
		// return EngineTypeNone
		return EngineTypeBTree
	case string:
		if p.Operator == operators.Equals {
			// StringHash is only used for matching on equality.
			return EngineTypeStringHash
		}
	case nil:
		// Only allow this if we're not comparing two idents.
		if p.LiteralIdent != nil {
			return EngineTypeNone
		}
		return EngineTypeNullMatch
	}
	// case int64, float64:
	// 	return EngineTypeBTree

	return EngineTypeNone
}

// nodeOp represents an op eg. addNode or removeNode
type nodeOp func(ctx context.Context, n *Node, parsed *ParsedExpression) error

func (a *aggregator) addNode(ctx context.Context, n *Node, parsed *ParsedExpression) error {
	if n.Predicate == nil {
		return nil
	}
	e := a.engine(n)
	if e == nil {
		return errEngineUnimplemented
	}

	// Don't allow anything to update in parallel.  This ensures that Add() can be called
	// concurrently.
	a.lock.Lock()
	defer a.lock.Unlock()
	return e.Add(ctx, ExpressionPart{
		GroupID:   n.GroupID,
		Predicate: n.Predicate,
		Parsed:    parsed,
	})
}

func (a *aggregator) removeNode(ctx context.Context, n *Node, parsed *ParsedExpression) error {
	if n.Predicate == nil {
		return nil
	}
	e := a.engine(n)
	if e == nil {
		return errEngineUnimplemented
	}

	// Don't allow anything to update in parallel.  This enrues that Add() can be called
	// concurrently.
	a.lock.Lock()
	defer a.lock.Unlock()
	return e.Remove(ctx, ExpressionPart{
		GroupID:   n.GroupID,
		Predicate: n.Predicate,
		Parsed:    parsed,
	})
}

func (a *aggregator) engine(n *Node) MatchingEngine {
	requiredEngine := engineType(*n.Predicate)
	if requiredEngine == EngineTypeNone {
		return nil
	}
	for _, engine := range a.engines {
		if engine.Type() != requiredEngine {
			continue
		}
		return engine
	}
	return nil
}

func isAggregateable(n *Node) bool {
	if n.Predicate == nil {
		// This is a parent node.  We skip aggregateable checks and only
		// return false based off of predicate information.
		return true
	}
	if n.Predicate.LiteralIdent != nil {
		// We're matching idents together, so this is not aggregateable.
		return false
	}

	if n.Predicate.Operator == "comprehension" {
		return false
	}

	switch v := n.Predicate.Literal.(type) {
	case string:
		if len(v) == 0 {
			return false
		}
		if n.Predicate.Operator == operators.NotEquals {
			// NOTE: NotEquals is _not_ supported.  This requires selecting all leaf nodes _except_
			// a given leaf, iterating over a tree.  We may as well execute every expressiona s the difference
			// is negligible.
			return false
		}
		// Right now, we only support equality checking.
		// TODO: Add GT(e)/LT(e) matching with tree iteration.
		return n.Predicate.Operator == operators.Equals
	case int, int64, float64:
		return true
	case nil:
		// This is null, which is supported and a simple lookup to check
		// if the event's key in question is present and is not nil.
		return true
	default:
		return false
	}
}
