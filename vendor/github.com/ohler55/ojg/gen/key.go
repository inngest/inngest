// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

// Key use for parsing.
type Key string

// String returns the key as a string.
func (k Key) String() string {
	return string(k)
}

// Alter converts the node into it's native type. Note this will modify
// Objects and Arrays in place making them no longer usable as the
// original type. Use with care!
func (k Key) Alter() any {
	return string(k)
}

// Simplify makes a copy of the node but as simple types.
func (k Key) Simplify() any {
	return string(k)
}

// Dup returns a deep duplicate of the node.
func (k Key) Dup() Node {
	return k
}

// Empty returns true if the node is empty.
func (k Key) Empty() bool {
	return false
}
