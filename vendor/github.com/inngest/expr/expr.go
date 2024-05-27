package expr

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/cel-go/common/operators"
	"github.com/google/uuid"
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
	// Add adds an expression to the tree evaluator.  This returns true
	// if the expression is aggregateable, or false if the expression will be
	// evaluated each time an event is received.
	Add(ctx context.Context, eval Evaluable) (bool, error)

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

	// AggregateableLen returns the number of expressions being matched by aggregated trees.
	AggregateableLen() int

	// ConstantLen returns the total number of expressions that must constantly
	// be matched due to non-aggregateable clauses in their expressions.
	ConstantLen() int
}

func NewAggregateEvaluator(
	parser TreeParser,
	eval ExpressionEvaluator,
	evalLoader EvaluableLoader,
) AggregateEvaluator {
	return &aggregator{
		eval:   eval,
		parser: parser,
		loader: evalLoader,
		engines: map[EngineType]MatchingEngine{
			EngineTypeStringHash: newStringEqualityMatcher(),
			EngineTypeNullMatch:  newNullMatcher(),
			EngineTypeBTree:      newNumberMatcher(),
		},
		lock: &sync.RWMutex{},
	}
}

type aggregator struct {
	eval   ExpressionEvaluator
	parser TreeParser
	loader EvaluableLoader

	// engines records all engines
	engines map[EngineType]MatchingEngine

	// lock prevents concurrent updates of data
	lock *sync.RWMutex
	// len stores the current len of aggregable expressions.
	len int32
	// constants tracks evaluable IDs that must always be evaluated, due to
	// the expression containing non-aggregateable clauses.
	constants []uuid.UUID
}

// Len returns the total number of aggregateable and constantly matched expressions
// stored in the evaluator.
func (a aggregator) Len() int {
	return int(a.len) + len(a.constants)
}

// AggregateableLen returns the number of expressions being matched by aggregated trees.
func (a aggregator) AggregateableLen() int {
	return int(a.len)
}

// ConstantLen returns the total number of expressions that must constantly
// be matched due to non-aggregateable clauses in their expressions.
func (a aggregator) ConstantLen() int {
	return len(a.constants)
}

func (a *aggregator) Evaluate(ctx context.Context, data map[string]any) ([]Evaluable, int32, error) {
	var (
		err     error
		matched = int32(0)
		result  = []Evaluable{}
	)

	// TODO: Concurrently match constant expressions using a semaphore for capacity.
	// Match constant expressions always.
	constantEvals, err := a.loader(ctx, a.constants...)
	if err != nil {
		return nil, 0, err
	}
	for _, expr := range constantEvals {
		atomic.AddInt32(&matched, 1)

		if expr.GetExpression() == "" {
			result = append(result, expr)
			continue
		}

		// NOTE: We don't need to add lifted expression variables,
		// because match.Parsed.Evaluable() returns the original expression
		// string.
		ok, evalerr := a.eval(ctx, expr, data)
		if evalerr != nil {
			err = errors.Join(err, evalerr)
			continue
		}
		if ok {
			result = append(result, expr)
		}
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
	seen := map[uuid.UUID]struct{}{}

	for _, match := range evaluables {
		if _, ok := seen[match.GetID()]; ok {
			continue
		}

		atomic.AddInt32(&matched, 1)
		// NOTE: We don't need to add lifted expression variables,
		// because match.Parsed.Evaluable() returns the original expression
		// string.
		ok, evalerr := a.eval(ctx, match, data)

		seen[match.GetID()] = struct{}{}

		if evalerr != nil {
			err = errors.Join(err, evalerr)
			continue
		}
		if ok {
			result = append(result, match)
		}
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
	// protect the above locks with a map.
	lock := &sync.Mutex{}

	for _, engine := range a.engines {
		matched, err := engine.Match(ctx, data)
		if err != nil {
			return nil, err
		}

		// Add all found items from the engine to the above list.
		lock.Lock()
		for _, eval := range matched {
			counts[eval.GroupID] += 1
			if _, ok := found[eval.GroupID]; !ok {
				found[eval.GroupID] = []*StoredExpressionPart{}
			}
			found[eval.GroupID] = append(found[eval.GroupID], eval)
		}
		lock.Unlock()
	}

	// Validate that groups meet the minimum size.
	for k, count := range counts {
		// if int(k.Size()) > count {
		// 	// The GroupID required more comparisons to equate to true than
		// 	// we had, so this could never evaluate to true.  Skip this.
		// 	//
		// 	// TODO: Optimize and fix.
		// 	continue
		// }
		_ = count
		result = append(result, found[k]...)
	}

	return result, nil
}

// Add adds an Evaluable to the aggregate tree engine for matching.  It returns
// a boolean indicating whether the expression is suitable for aggregate tree
// matching, allowing rapid exclusion of non-matching expressions.
func (a *aggregator) Add(ctx context.Context, eval Evaluable) (bool, error) {
	// parse the expression using our tree parser.
	parsed, err := a.parser.Parse(ctx, eval)
	if err != nil {
		return false, err
	}

	if eval.GetExpression() == "" || parsed.HasMacros {
		// This is an empty expression which always matches.
		a.lock.Lock()
		a.constants = append(a.constants, parsed.EvaluableID)
		a.lock.Unlock()
		return false, nil
	}

	for _, g := range parsed.RootGroups() {
		ok, err := a.iterGroup(ctx, g, parsed, a.addNode)

		if err != nil || !ok {
			// This is the first time we're seeing a non-aggregateable
			// group, so add it to the constants list and don't do anything else.
			a.lock.Lock()
			a.constants = append(a.constants, parsed.EvaluableID)
			a.lock.Unlock()
			return false, err
		}
	}

	// Track the number of added expressions correctly.
	atomic.AddInt32(&a.len, 1)
	return true, nil
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

	aggregateable := true
	for _, g := range parsed.RootGroups() {
		ok, err := a.iterGroup(ctx, g, parsed, a.removeNode)
		if err == ErrExpressionPartNotFound {
			return ErrEvaluableNotFound
		}
		if err != nil {
			return err
		}
		if !ok && aggregateable {
			if err := a.removeConstantEvaluable(ctx, eval); err != nil {
				return err
			}
			aggregateable = false
		}
	}

	if aggregateable {
		atomic.AddInt32(&a.len, -1)
	}

	return nil
}

func (a *aggregator) removeConstantEvaluable(ctx context.Context, eval Evaluable) error {
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

	a.lock.Lock()
	a.constants = append(a.constants[:idx], a.constants[idx+1:]...)
	a.lock.Unlock()
	return nil
}

func (a *aggregator) iterGroup(ctx context.Context, node *Node, parsed *ParsedExpression, op nodeOp) (bool, error) {
	if len(node.Ors) > 0 {
		// If there are additional branches, don't bother to add this to the aggregate tree.
		// Mark this as a non-exhaustive addition and skip immediately.
		//
		// TODO: Allow ORs _only if_ the ORs are not nested, eg. the ORs are basic predicate
		// groups that themselves have no branches.
		return false, nil
	}

	if len(node.Ands) > 0 {
		for _, n := range node.Ands {
			if !n.HasPredicate() || len(n.Ors) > 0 {
				// Don't handle sub-branching for now.
				return false, nil
			}
			if !isAggregateable(n) {
				return false, nil
			}
		}
	}

	all := node.Ands
	if node.Predicate != nil {
		if !isAggregateable(node) {
			return false, nil
		}
		// Merge all of the nodes together and check whether each node is aggregateable.
		all = append(node.Ands, node)
	}

	// Create a new group ID which tracks the number of expressions that must match
	// within this group in order for the group to pass.
	//
	// This includes ALL ands, plus at least one OR.
	//
	// When checking an incoming event, we match the event against each node's
	// ident/variable.  Using the group ID, we can see if we've matched N necessary
	// items from the same identifier.  If so, the evaluation is true.
	for _, n := range all {
		err := op(ctx, n, parsed)
		if err == errEngineUnimplemented {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}

	return true, nil
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
