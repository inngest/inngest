package expr

import (
	"context"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
)

type EngineType int

const (
	EngineTypeNone = iota

	EngineTypeStringHash
	EngineTypeNullMatch
	EngineTypeBTree
	// EngineTypeART
)

// matchKey is a composite key combining evalID and groupID to avoid nested map allocations
type matchKey struct {
	evalID  uuid.UUID
	groupID groupID
}

func NewMatchResult() *MatchResult {
	return &MatchResult{
		Result: map[matchKey]int{},
	}
}

// MatchResult uses a flat map with composite keys to avoid allocating nested maps
type MatchResult struct {
	Result map[matchKey]int
}

func (m *MatchResult) Len() int {
	// Count unique evalIDs
	seen := make(map[uuid.UUID]struct{}, len(m.Result))
	for key := range m.Result {
		seen[key.evalID] = struct{}{}
	}
	return len(seen)
}

// Add increments the matched counter for the given eval's group ID
func (m *MatchResult) Add(evalID uuid.UUID, gID groupID) {
	key := matchKey{evalID: evalID, groupID: gID}
	m.Result[key]++
}

// AddExprs increments the matched counter for each stored expression part.
func (m *MatchResult) AddExprs(exprs ...*StoredExpressionPart) {
	for _, expr := range exprs {
		key := matchKey{evalID: expr.EvaluableID, groupID: expr.GroupID}
		m.Result[key]++
	}
}

// GroupMatches returns the total count of matches for a given eval's group ID.
func (m *MatchResult) GroupMatches(evalID uuid.UUID, gID groupID) int {
	key := matchKey{evalID: evalID, groupID: gID}
	return m.Result[key]
}

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
	//
	// Note that the MatchResult is mutated on input.
	Match(ctx context.Context, input map[string]any, result *MatchResult) (err error)

	// Add adds a new expression part to the matching engine for future matches.
	Add(ctx context.Context, p ExpressionPart) error
	// Remove removes multiple expression parts in a single batch operation.
	// Returns the number of parts successfully processed before any timeout/cancellation.
	Remove(ctx context.Context, parts []ExpressionPart) (int, error)

	// Search searches for a given variable<>value match, returning any expression
	// parts that match.
	//
	// Similar to match, each implementation of the engine may differ on
	// granularity of expression parts received.  Some may return false positives by
	// ignoring the variable name.  Note that each MatchingEngine should NEVER
	// omit ExpressionParts which match the given input;  false positives are okay,
	Search(ctx context.Context, variable string, input any, result *MatchResult)
	// but not returning valid matches must be impossible.
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
	Predicate *Predicate
	Parsed    *ParsedExpression
}

func (p ExpressionPart) Hash() uint64 {
	return xxhash.Sum64String(p.Predicate.String())
}

func (p ExpressionPart) EqualsStored(n *StoredExpressionPart) bool {
	if p.GroupID != n.GroupID {
		return false
	}
	return p.Hash() == n.PredicateID
}

func (p ExpressionPart) Equals(n ExpressionPart) bool {
	if p.GroupID != n.GroupID {
		return false
	}
	if p.Predicate.String() != n.Predicate.String() {
		return false
	}
	return p.Parsed.EvaluableID == n.Parsed.EvaluableID
}

func (p ExpressionPart) ToStored() *StoredExpressionPart {
	var id uuid.UUID
	if p.Parsed != nil {
		id = p.Parsed.EvaluableID
	}

	return &StoredExpressionPart{
		EvaluableID: id,
		GroupID:     p.GroupID,
		PredicateID: p.Hash(),
		Ident:       &p.Predicate.Ident,
	}
}

// StoredExpressionPart is a lightweight expression part which only stores
// a hash of the predicate to reduce memory usage.
type StoredExpressionPart struct {
	EvaluableID uuid.UUID
	GroupID     groupID
	PredicateID uint64
	Ident       *string
}
