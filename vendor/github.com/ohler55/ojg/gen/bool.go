// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

// Bool repreents a boolean value.
type Bool bool

// True is a true boolean value.
var True = Bool(true)

// False is a false boolean value.
var False = Bool(false)

// String returns a string representation of the Node.
func (n Bool) String() (s string) {
	if n {
		s = "true"
	} else {
		s = "false"
	}
	return
}

// Alter returns the backing boolean value of the Node.
func (n Bool) Alter() any {
	return bool(n)
}

// Simplify returns the backing boolean value.
func (n Bool) Simplify() any {
	return bool(n)
}

// Dup returns itself.
func (n Bool) Dup() Node {
	return n
}

// Empty returns false.
func (n Bool) Empty() bool {
	return false
}
