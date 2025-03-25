// Copyright (c) 2023, Peter Ohler, All rights reserved.

package jp

// Indexed describes an interface for a collection that is indexed by a
// integers similar to a []any.
type Indexed interface {

	// ValueAtIndex should return the value at the provided index or nil if no
	// entry exists at the index.
	ValueAtIndex(index int) any

	// SetValueAtIndex should set the value at the provided index.
	SetValueAtIndex(index int, value any)

	// Size should return the size for the collection.
	Size() int
}

// RemovableIndexed describes an indexed collection that can remove items.
// Must be implemented to use [Expr.Remove].
type RemovableIndexed interface {
	Indexed

	// RemoveValueAtIndex removes an item from the collection.
	RemoveValueAtIndex(index int)
}
