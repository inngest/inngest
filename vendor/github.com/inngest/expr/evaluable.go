package expr

import "github.com/google/uuid"

// Evaluable represents an evaluable expression with a unique identifier.
type Evaluable interface {
	// GetID returns a unique identifier for the evaluable item.  If there are
	// two instances of the same expression, the identifier should return a unique
	// string for each instance of the expression (eg. for two pauses).
	//
	// It has the Get prefix to reduce collisions with implementations who expose an
	// ID member.
	GetID() uuid.UUID

	// GetExpression returns an expression as a raw string.
	//
	// It has the Get prefix to reduce collisions with implementations who expose an
	// Expression member.
	GetExpression() string
}

// StringExpression is a string type that implements Evaluable, useful for basic
// ephemeral expressions that have no lifetime.
type StringExpression string

func (s StringExpression) GetID() uuid.UUID {
	// deterministic IDs based off of expressions in testing.
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(s))
}
func (s StringExpression) GetExpression() string { return string(s) }
