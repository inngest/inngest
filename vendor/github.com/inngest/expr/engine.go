package expr

import (
	"context"
)

type EngineType int

const (
	EngineTypeNone = iota

	EngineTypeStringHash
	EngineTypeNullMatch
	// EngineTypeART
	// EngineTypeBTree
)

// MatchingEngine represents an engine (such as a b-tree, radix trie, or
// simple hash map) which matches a predicate over many expressions.
type MatchingEngine interface {
	// Type returns the EngineType
	Type() EngineType
	// Match takes an input event, containing key:value pairs of data, and
	// matches the given data to any ExpressionParts stored in the engine.
	//
	// Each implementation of the engine may differ on granularity of
	// expression parts received.  Some may return false positives, but
	// each MatchingEngine should NEVER omit ExpressionParts which match
	// the given input.
	Match(ctx context.Context, input map[string]any) ([]*ExpressionPart, error)
	// Add adds a new expression part to the matching engine for future matches.
	Add(ctx context.Context, p ExpressionPart) error
	// Remove removes an expression part from the matching engine, ensuring that the
	// ExpressionPart will not be matched in the future.
	Remove(ctx context.Context, p ExpressionPart) error

	// Search searches for a given variable<>value match, returning any expression
	// parts that match.
	//
	// Similar to match, each implementation of the engine may differ on
	// granularity of expression parts received.  Some may return false positives by
	// ignoring the variable name.  Note that each MatchingEngine should NEVER
	// omit ExpressionParts which match the given input;  false positives are okay,
	// but not returning valid matches must be impossible.
	Search(ctx context.Context, variable string, input any) []*ExpressionPart
}

// Leaf represents the leaf within a tree.  This stores all expressions
// which match the given expression.
//
// For example, adding two expressions each matching "event.data == 'foo'"
// in an ART creates a leaf node with both evaluable expressions stored
// in Evals
//
// Note that there are many sub-clauses which need to be matched.  Each
// leaf is a subset of a full expression.  Therefore,
type Leaf struct {
	Evals []*ExpressionPart
}

// ExpressionPart represents a predicate group which is part of an expression.
// All parts for the given group ID must evaluate to true for the predicate to
// be matched.
type ExpressionPart struct {
	// GroupID represents a group ID for the expression part.
	//
	// Within an expression, multiple predicates may be chained with &&.  Each
	// of these must evaluate to `true` for an expression to match.  Group IDs
	// are shared amongst each predicate within an expression.
	//
	// This lets us determine whether the entire group has been matched.
	GroupID   groupID
	Predicate Predicate
	Parsed    *ParsedExpression
}

func (p ExpressionPart) Equals(n ExpressionPart) bool {
	if p.GroupID != n.GroupID {
		return false
	}
	if p.Predicate.String() != n.Predicate.String() {
		return false
	}
	return p.Parsed.Evaluable.GetExpression() == n.Parsed.Evaluable.GetExpression()
}
