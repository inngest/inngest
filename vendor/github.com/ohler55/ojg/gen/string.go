// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

// String is a string Node.
type String string

// String returns a string representation of the Node.
func (n String) String() string {
	return `"` + string(n) + `"`
}

// Alter returns the backing float64 value of the Node.
func (n String) Alter() any {
	return string(n)
}

// Simplify returns the backing float64 value of the Node.
func (n String) Simplify() any {
	return string(n)
}

// Dup returns the backing float64 value of the Node.
func (n String) Dup() Node {
	return n
}

// Empty returns false if the string has no characters and true otherwise.
func (n String) Empty() bool {
	return len(string(n)) == 0
}
