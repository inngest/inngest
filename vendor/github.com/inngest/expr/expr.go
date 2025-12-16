package expr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/pebble/vfs"
	"github.com/google/cel-go/common/operators"
	"github.com/google/uuid"
)

var (
	ErrEvaluableNotFound      = fmt.Errorf("Evaluable instance not found in aggregator")
	ErrInvalidType            = fmt.Errorf("invalid type for tree")
	ErrExpressionPartNotFound = fmt.Errorf("expression part not found")
)

const (
	defaultConcurrency = 1000
)

// errEngineUnimplemented is used while we develop the aggregate tree library when trees
// are not yet implemented.
var errEngineUnimplemented = fmt.Errorf("tree type unimplemented")

// ExpressionEvaluator is a function which evalues an expression given input data, returning
// a boolean and error.
type ExpressionEvaluator func(ctx context.Context, e Evaluable, input map[string]any) (bool, error)

// AggregateEvaluator represents a group of expressions that must be evaluated for a single
// event received.
//
// An AggregateEvaluator instance exists for every event name being matched.
type AggregateEvaluator[T Evaluable] interface {
	// Add adds an expression to the tree evaluator.  This returns the ratio
	// of aggregate to slow parts in the expression, or an error if there was an
	// issue.
	//
	// Purely aggregateable expressions have a ratio of 1.
	// Mixed expressions return the ratio of fast:slow expressions, as a float.
	// Slow, non-aggregateable expressions return 0.
	Add(ctx context.Context, eval T) (float64, error)

	// Remove removes an expression from the aggregate evaluator
	Remove(ctx context.Context, eval T) error

	// Evaluate checks input data against all exrpesssions in the aggregate in an optimal
	// manner, only evaluating expressions when necessary (based off of tree matching).
	//
	// Note that any expressions added that cannot be evaluated optimally by trees
	// are evaluated every time this function is called.
	//
	// Evaluate returns all matching Evaluables, plus the total number of evaluations
	// executed.
	Evaluate(ctx context.Context, data map[string]any) ([]T, int32, error)

	// AggregateMatch returns all expression parts which are evaluable given the input data.
	AggregateMatch(ctx context.Context, data map[string]any) ([]*uuid.UUID, error)

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

type AggregateEvaluatorOpts[T Evaluable] struct {
	// Parser is the parser to use which compiles expressions into a *ParsedExpression tree.
	Parser TreeParser
	// Eval is the evaluator function to use which, given an Evaluable and some input data,
	// returns whether the expression evaluated to true or false.
	Eval ExpressionEvaluator
	// Concurrency is the number of evaluable instances to evaluate at once, if there
	// are multiple matches for a given AggregateMatch or Evaluate call.
	Concurrency int64
	// KV represents storage for evaluables.
	KV KV[T]
	// Log is a stdlib logger used for logging.  If nil, this will be slog.Default().
	Log *slog.Logger
}

func NewAggregateEvaluator[T Evaluable](
	opts AggregateEvaluatorOpts[T],
) AggregateEvaluator[T] {
	if opts.Concurrency <= 0 {
		opts.Concurrency = defaultConcurrency
	}

	if opts.Log == nil {
		opts.Log = slog.Default()
	}

	// Create a new KV store.
	if opts.KV == nil {
		var err error
		kvopts := KVOpts[T]{
			Marshal: func(eval T) ([]byte, error) {
				return json.Marshal(eval)
			},
			Unmarshal: func(byt []byte) (t T, err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("error unmarshalling type %T: %s. Did you pass an interface type to NewAggregateEvaluator?", t, r)
					}
				}()

				val := reflect.New(reflect.TypeOf(t)).Interface()
				err = json.Unmarshal(byt, val)
				if err != nil {
					return t, err
				}

				return reflect.ValueOf(val).Elem().Interface().(T), err
			},
			FS: vfs.NewMem(),
		}
		// Attempt to unmarshal an empty byte slice, ensuring that we have
		// a concrete type instead of an interface.
		if _, err := kvopts.Unmarshal([]byte("{}")); err != nil {
			panic(fmt.Sprintf("unable to make KV for aggregate evaluator without concrete type: %s", err))
		}
		opts.KV, err = NewKV[T](kvopts)
		if err != nil {
			panic(fmt.Sprintf("unable to make KV for aggregate evaluator: %s", err))
		}
	}

	return &aggregator[T]{
		kv:     opts.KV,
		eval:   opts.Eval,
		parser: opts.Parser,
		engines: map[EngineType]MatchingEngine{
			EngineTypeStringHash: newBitmapStringEqualityMatcher(opts.Concurrency),
			EngineTypeNullMatch:  newNullMatcher(opts.Concurrency),
			EngineTypeBTree:      newNumberMatcher(opts.Concurrency),
		},
		lock:        &sync.RWMutex{},
		constants:   map[uuid.UUID]struct{}{},
		mixed:       map[uuid.UUID]struct{}{},
		concurrency: opts.Concurrency,
		log:         opts.Log,
	}
}

type aggregator[T Evaluable] struct {
	eval   ExpressionEvaluator
	parser TreeParser

	log *slog.Logger

	kv KV[T]

	// engines records all engines
	engines map[EngineType]MatchingEngine

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
	constants map[uuid.UUID]struct{}

	concurrency int64
}

// Len returns the total number of aggregateable and constantly matched expressions
// stored in the evaluator.
func (a *aggregator[T]) Len() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return int(a.fastLen) + len(a.mixed) + len(a.constants)
}

// FastLen returns the number of expressions being matched by aggregated trees.
func (a *aggregator[T]) FastLen() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return int(a.fastLen)
}

// MixedLen returns the number of expressions being matched by aggregated trees.
func (a *aggregator[T]) MixedLen() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return len(a.mixed)
}

// SlowLen returns the total number of expressions that must constantly
// be matched due to non-aggregateable clauses in their expressions.
func (a *aggregator[T]) SlowLen() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return len(a.constants)
}

func (a *aggregator[T]) Evaluate(ctx context.Context, data map[string]any) ([]T, int32, error) {
	var (
		err     error
		matched = int32(0)
		result  = []T{}
		s       sync.Mutex
	)

	napool := newErrPool(errPoolOpts{concurrency: a.concurrency})

	a.lock.RLock()
	for uuid := range a.constants {
		item, err := a.kv.Get(uuid)
		if err != nil {
			continue
		}

		expr := item
		napool.Go(func() error {
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
	a.lock.RUnlock()

	if werr := napool.Wait(); werr != nil {
		err = errors.Join(err, werr)
	}

	matches, merr := a.AggregateMatch(ctx, data)
	if merr != nil {
		err = errors.Join(err, merr)
	}

	// Each match here is a potential success.  When other trees and operators which are walkable
	// are added (eg. >= operators on strings), ensure that we find the correct number of matches
	// for each group ID and then skip evaluating expressions if the number of matches is <= the group
	// ID's length.
	seenMu := &sync.Mutex{}
	seen := map[uuid.UUID]struct{}{}

	mpool := newErrPool(errPoolOpts{concurrency: a.concurrency})

	a.lock.RLock()
	for _, id := range matches {
		eval, err := a.kv.Get(*id)
		if err != nil {
			continue
		}

		mpool.Go(func() error {
			defer func() {
				if r := recover(); r != nil {
					s.Lock()
					err = errors.Join(err, fmt.Errorf("recovered from panic in evaluate: %v", r))
					s.Unlock()
				}
			}()

			seenMu.Lock()
			if _, ok := seen[eval.GetID()]; ok {
				seenMu.Unlock()
				return nil
			} else {
				seen[eval.GetID()] = struct{}{}
				seenMu.Unlock()
			}

			atomic.AddInt32(&matched, 1)

			// NOTE: We don't need to add lifted expression variables,
			// because match.Parsed.Evaluable() returns the original expression
			// string.
			ok, evalerr := a.eval(ctx, eval, data)

			if evalerr != nil {
				return evalerr
			}
			if ok {
				s.Lock()
				result = append(result, eval)
				s.Unlock()
			}
			return nil
		})
	}
	a.lock.RUnlock()

	if werr := mpool.Wait(); werr != nil {
		err = errors.Join(err, werr)
	}

	return result, matched, err
}

// AggregateMatch attempts to match incoming data to all PredicateTrees, resulting in a selection
// of parts of an expression that have matched.
func (a *aggregator[T]) AggregateMatch(ctx context.Context, data map[string]any) ([]*uuid.UUID, error) {
	result := []*uuid.UUID{}

	a.lock.RLock()
	defer a.lock.RUnlock()

	// Each match here is a potential success.  Ensure that we find the correct number of matches
	// for each group ID and then skip evaluating expressions if the number of matches is <= the group
	// ID's length.  For example, (A && B && C) is a single group ID and must have a count >= 3,
	// else we know a required comparason did not match.
	//
	// Note that having a count >= the group ID value does not guarantee that the expression is valid.
	//
	// Note that we break this down per evaluable ID (UUID)
	found := NewMatchResult()

	for _, engine := range a.engines {
		// we explicitly ignore the deny path for now.
		if err := engine.Match(ctx, data, found); err != nil {
			return nil, err
		}
	}

	a.log.Debug("ran matching engines", "len_matched_no_filter", found.Len())

	// Validate that groups meet the minimum size.
	for evalID, groups := range found.Result {
		for groupID, matchingCount := range groups {
			requiredSize := int(groupID.Size()) // The total req size from the group ID

			// If this group isn't the required size, delete the group
			// from our map
			if matchingCount < requiredSize {
				delete(groups, groupID)
			}

		}
		// After iterating through each group, we now know:
		//
		// if len(groups) > 0, we have enough matches in this eval group for
		// it to be a candidate.
		hasMatchedGroups := len(groups) > 0

		// NOTE: We currently don't add items with OR predicates to the
		// matching engine, so we cannot use group sizes if the expr part
		// has an OR.
		_, isMixedOrs := a.mixed[evalID]

		if hasMatchedGroups || isMixedOrs {
			result = append(result, &evalID)
		}
	}

	a.log.Debug("filtered invalid groups", "len_matched", len(result))

	return result, nil
}

// Add adds an expression to the tree evaluator.  This returns the ratio
// of aggregate to slow parts in the expression, or an error if there was an
// issue.
//
// Purely aggregateable expressions have a ratio of 1.
// Mixed expressions return the ratio of fast:slow expressions, as a float.
// Slow, non-aggregateable expressions return 0.
func (a *aggregator[T]) Add(ctx context.Context, eval T) (float64, error) {
	// parse the expression using our tree parser.
	parsed, err := a.parser.Parse(ctx, eval)
	if err != nil {
		return -1, err
	}

	if err := a.kv.Set(eval); err != nil {
		return -1, err
	}

	if eval.GetExpression() == "" || parsed.HasMacros {
		// This is an empty expression which always matches.
		a.lock.Lock()
		a.constants[parsed.EvaluableID] = struct{}{}
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
			a.constants[parsed.EvaluableID] = struct{}{}
			a.lock.Unlock()
			return -1, err
		}

		stats.Merge(s)
	}

	if stats.Fast() == 0 {
		// This is a non-aggregateable, slow expression.
		// Add it to the constants list and don't do anything else.
		a.lock.Lock()
		a.constants[parsed.EvaluableID] = struct{}{}
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

func (a *aggregator[T]) Remove(ctx context.Context, eval T) error {
	if err := a.kv.Remove(eval.GetID()); err != nil {
		return err
	}

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

func (a *aggregator[T]) removeConstantEvaluable(_ context.Context, eval Evaluable) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	// Find the index of the evaluable in constants and yank out.
	if _, ok := a.constants[eval.GetID()]; !ok {
		return ErrEvaluableNotFound
	}

	delete(a.constants, eval.GetID())
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
func (a *aggregator[T]) iterGroup(ctx context.Context, node *Node, parsed *ParsedExpression, op nodeOp) (exprAggregateStats, error) {
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

	all := node.Ands

	// XXX: Here we must add the OR groups to make group IDs a success.
	if len(node.Ors) > 0 {
		// Mark this as a mixed/slow expression to be fully tested.
		stats.AddSlow()
	}

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
	switch v := p.Literal.(type) {
	case int, int64, float64:
		if p.Operator == operators.NotEquals || p.Operator == operators.In {
			return EngineTypeNone
		}
		return EngineTypeBTree
	case string:
		if len(v) == 0 {
			return EngineTypeNone
		}
		// NOTE: operators.In acts as operators.Equals, but iterates over the given
		// array to check each item.
		if p.Operator == operators.In || p.Operator == operators.Equals || p.Operator == operators.NotEquals {
			// StringHash is only used for matching on in/equality.
			return EngineTypeStringHash
		}
	case nil:
		// Only allow this if we're not comparing two idents.each element of the array and
		if p.LiteralIdent != nil {
			return EngineTypeNone
		}
		return EngineTypeNullMatch
	}

	return EngineTypeNone
}

// nodeOp represents an op eg. addNode or removeNode
type nodeOp func(ctx context.Context, n *Node, parsed *ParsedExpression) error

func (a *aggregator[T]) addNode(ctx context.Context, n *Node, parsed *ParsedExpression) error {
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

func (a *aggregator[T]) removeNode(ctx context.Context, n *Node, parsed *ParsedExpression) error {
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

func (a *aggregator[T]) engine(n *Node) MatchingEngine {
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

	// If the engine type is none... this is non-aggregateable
	return engineType(*n.Predicate) != EngineTypeNone
}
