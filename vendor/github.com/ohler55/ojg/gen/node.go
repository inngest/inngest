// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

import (
	"fmt"
)

// Node is the interface for typed generic data.
type Node interface {
	fmt.Stringer

	// Alter converts the node into it's native type. Note this will modify
	// Objects and Arrays in place making them no longer usable as the
	// original type. Use with care!
	Alter() any

	// Simplify makes a copy of the node but as simple types.
	Simplify() any

	// Dup returns a deep duplicate of the node.
	Dup() Node

	// Empty returns true if the node is empty.
	Empty() bool
}
