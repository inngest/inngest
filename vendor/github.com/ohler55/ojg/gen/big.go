// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

// Big represents a number too large to be an int64 or a float64.
type Big string

// String representation of the number.
func (n Big) String() string {
	return string(n)
}

// Alter returns the backing string.
func (n Big) Alter() any {
	return string(n)
}

// Simplify the Node into a string.
func (n Big) Simplify() any {
	return string(n)
}

// Dup returns itself since it is immutable.
func (n Big) Dup() Node {
	return n
}

// Empty returns true if the backing string is empty.
func (n Big) Empty() bool {
	return len(string(n)) == 0
}
