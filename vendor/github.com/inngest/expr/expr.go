package expr

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/cel-go/common/operators"
	"github.com/ohler55/ojg/jp"
	"golang.org/x/sync/errgroup"
)

var (
	ErrEvaluableNotFound = fmt.Errorf("Evaluable instance not found in aggregator")
)

// errTreeUnimplemented is used while we develop the aggregate tree library when trees
// are not yet implemented.
var errTreeUnimplemented = fmt.Errorf("tree type unimplemented")

// ExpressionEvaluator is a function which evalues an expression given input data, returning
// a boolean and error.
type ExpressionEvaluator func(ctx context.Context, e Evaluable, input map[string]any) (bool, error)

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
	AggregateMatch(ctx context.Context, data map[string]any) ([]ExpressionPart, error)

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
) AggregateEvaluator {
	return &aggregator{
		eval:        eval,
		parser:      parser,
		artIdents:   map[string]PredicateTree{},
		nullLookups: map[string]PredicateTree{},
		lock:        &sync.RWMutex{},
	}
}

type aggregator struct {
	eval   ExpressionEvaluator
	parser TreeParser

	artIdents   map[string]PredicateTree
	nullLookups map[string]PredicateTree
	lock        *sync.RWMutex

	len int32

	// constants tracks evaluable instances that must always be evaluated, due to
	// the expression containing non-aggregateable clauses.
	constants []*ParsedExpression
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
	for _, expr := range a.constants {
		atomic.AddInt32(&matched, 1)

		if expr.Evaluable.GetExpression() == "" {
			result = append(result, expr.Evaluable)
			continue
		}

		// NOTE: We don't need to add lifted expression variables,
		// because match.Parsed.Evaluable() returns the original expression
		// string.
		ok, evalerr := a.eval(ctx, expr.Evaluable, data)
		if evalerr != nil {
			err = errors.Join(err, evalerr)
			continue
		}
		if ok {
			result = append(result, expr.Evaluable)
		}
	}

	matches, merr := a.AggregateMatch(ctx, data)
	if merr != nil {
		err = errors.Join(err, merr)
	}

	// Each match here is a potential success.  When other trees and operators which are walkable
	// are added (eg. >= operators on strings), ensure that we find the correct number of matches
	// for each group ID and then skip evaluating expressions if the number of matches is <= the group
	// ID's length.
	seen := map[groupID]struct{}{}

	for _, match := range matches {
		if _, ok := seen[match.GroupID]; ok {
			continue
		}

		atomic.AddInt32(&matched, 1)
		// NOTE: We don't need to add lifted expression variables,
		// because match.Parsed.Evaluable() returns the original expression
		// string.
		ok, evalerr := a.eval(ctx, match.Parsed.Evaluable, data)

		seen[match.GroupID] = struct{}{}

		if evalerr != nil {
			err = errors.Join(err, evalerr)
			continue
		}
		if ok {
			result = append(result, match.Parsed.Evaluable)
		}
	}

	return result, matched, nil
}

// AggregateMatch attempts to match incoming data to all PredicateTrees, resulting in a selection
// of parts of an expression that have matched.
func (a *aggregator) AggregateMatch(ctx context.Context, data map[string]any) ([]ExpressionPart, error) {
	result := []ExpressionPart{}

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
	found := map[groupID][]ExpressionPart{}
	// protect the above locks with a map.
	lock := &sync.Mutex{}
	// run lookups concurrently.
	eg := errgroup.Group{}

	add := func(all []ExpressionPart) {
		// This is called concurrently, so don't mess with maps in goroutines
		lock.Lock()
		defer lock.Unlock()

		for _, eval := range all {
			counts[eval.GroupID] += 1
			if _, ok := found[eval.GroupID]; !ok {
				found[eval.GroupID] = []ExpressionPart{}
			}
			found[eval.GroupID] = append(found[eval.GroupID], eval)
		}
	}

	// Iterate through all known variables/idents in the aggregate tree to see if
	// the data has those keys set.  If so, we can immediately evaluate the data with
	// the tree.
	//
	// TODO: we should iterate through the expression in a top-down order, ensuring that if
	// any of the top groups fail to match we quit early.
	for n, item := range a.artIdents {
		tree := item
		path := n
		eg.Go(func() error {
			x, err := jp.ParseString(path)
			if err != nil {
				return err
			}
			res := x.Get(data)
			if len(res) != 1 {
				return nil
			}

			cast, ok := res[0].(string)
			if !ok {
				// This isn't a string, so we can't compare within the radix tree.
				return nil
			}

			add(tree.Search(ctx, path, cast))
			return nil
		})
	}

	// Match on nulls.
	for n, item := range a.nullLookups {
		tree := item
		path := n
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
			add(tree.Search(ctx, path, res[0]))
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	for k, count := range counts {
		if int(k.Size()) > count {
			// The GroupID required more comparisons to equate to true than
			// we had, so this could never evaluate to true.  Skip this.
			continue
		}
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

	if eval.GetExpression() == "" {
		// This is an empty expression which always matches.
		a.lock.Lock()
		a.constants = append(a.constants, parsed)
		a.lock.Unlock()
		return false, nil
	}

	aggregateable := true
	for _, g := range parsed.RootGroups() {
		ok, err := a.iterGroup(ctx, g, parsed, a.addNode)
		if err != nil {
			return false, err
		}
		if !ok && aggregateable {
			// This is the first time we're seeing a non-aggregateable
			// group, so add it to the constants list.
			a.lock.Lock()
			a.constants = append(a.constants, parsed)
			a.lock.Unlock()
			aggregateable = false
		}
	}

	// Track the number of added expressions correctly.
	if aggregateable {
		atomic.AddInt32(&a.len, 1)
	}
	return aggregateable, nil
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
		if item.Evaluable.GetID() == eval.GetID() {
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
		if err == errTreeUnimplemented {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func treeType(p Predicate) TreeType {
	// switch on type of literal AND operator type.  int64/float64 literals require
	// btrees, texts require ARTs.
	switch p.Literal.(type) {
	case string:
		return TreeTypeART
	case int64, float64:
		return TreeTypeBTree
	case nil:
		// Only allow this if we're not comparing two idents.
		if p.LiteralIdent != nil {
			return TreeTypeNone
		}
		return TreeTypeNullMatch
	default:
		return TreeTypeNone
	}
}

// nodeOp represents an op eg. addNode or removeNode
type nodeOp func(ctx context.Context, n *Node, parsed *ParsedExpression) error

func (a *aggregator) addNode(ctx context.Context, n *Node, parsed *ParsedExpression) error {
	// Don't allow anything to update in parallel.  This enrues that Add() can be called
	// concurrently.
	a.lock.Lock()
	defer a.lock.Unlock()

	// Each node is aggregateable, so add this to the map for fast filtering.
	switch treeType(*n.Predicate) {
	case TreeTypeART:
		tree, ok := a.artIdents[n.Predicate.Ident]
		if !ok {
			tree = newArtTree()
		}
		err := tree.Add(ctx, ExpressionPart{
			GroupID:   n.GroupID,
			Predicate: *n.Predicate,
			Parsed:    parsed,
		})
		if err != nil {
			return err
		}
		a.artIdents[n.Predicate.Ident] = tree
		return nil
	case TreeTypeNullMatch:
		tree, ok := a.nullLookups[n.Predicate.Ident]
		if !ok {
			tree = newNullMatcher()
		}
		err := tree.Add(ctx, ExpressionPart{
			GroupID:   n.GroupID,
			Predicate: *n.Predicate,
			Parsed:    parsed,
		})
		if err != nil {
			return err
		}
		a.nullLookups[n.Predicate.Ident] = tree
		return nil
	}
	return errTreeUnimplemented
}

func (a *aggregator) removeNode(ctx context.Context, n *Node, parsed *ParsedExpression) error {
	// Don't allow anything to update in parallel.  This enrues that Add() can be called
	// concurrently.
	a.lock.Lock()
	defer a.lock.Unlock()

	// Each node is aggregateable, so add this to the map for fast filtering.
	switch treeType(*n.Predicate) {
	case TreeTypeART:
		tree, ok := a.artIdents[n.Predicate.Ident]
		if !ok {
			return ErrExpressionPartNotFound
		}
		err := tree.Remove(ctx, ExpressionPart{
			GroupID:   n.GroupID,
			Predicate: *n.Predicate,
			Parsed:    parsed,
		})
		if err != nil {
			return err
		}
		a.artIdents[n.Predicate.Ident] = tree
		return nil
	case TreeTypeNullMatch:
		tree, ok := a.nullLookups[n.Predicate.Ident]
		if !ok {
			return ErrExpressionPartNotFound
		}
		err := tree.Remove(ctx, ExpressionPart{
			GroupID:   n.GroupID,
			Predicate: *n.Predicate,
			Parsed:    parsed,
		})
		if err != nil {
			return err
		}
		a.nullLookups[n.Predicate.Ident] = tree
		return nil
	}
	return errTreeUnimplemented
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
		//
		// TODO: Add GT(e)/LT(e) matching with tree iteration.
		return n.Predicate.Operator == operators.Equals
	case int64, float64:
		// TODO: Add binary tree matching for ints/floats
		return false
	case nil:
		// This is null, which is supported and a simple lookup to check
		// if the event's key in question is present and is not nil.
		return true
	default:
		return false
	}
}
