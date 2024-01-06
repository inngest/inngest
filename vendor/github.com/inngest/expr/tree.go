package expr

import (
	"context"
)

type TreeType int

const (
	TreeTypeNone TreeType = iota

	TreeTypeART
	TreeTypeBTree
)

// PredicateTree represents a tree which matches a predicate over
// N expressions.
//
// For example, an expression may check string equality using an
// ART tree, while LTE operations may check against a b+-tree.
type PredicateTree interface {
	Add(ctx context.Context, p ExpressionPart) error
	Remove(ctx context.Context, p ExpressionPart) error
	Search(ctx context.Context, input any) (*Leaf, bool)
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
	Evals []ExpressionPart
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
	return p.Parsed.Evaluable.Expression() == n.Parsed.Evaluable.Expression()
}
