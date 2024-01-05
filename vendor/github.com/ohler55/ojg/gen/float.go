// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

import (
	"strconv"
)

// Float is a float64 Node.
type Float float64

// String returns a string representation of the Node.
func (n Float) String() string {
	return strconv.FormatFloat(float64(n), 'g', -1, 64)
}

// Alter returns the backing float64 value of the Node.
func (n Float) Alter() any {
	return float64(n)
}

// Simplify returns the backing float64 value of the Node.
func (n Float) Simplify() any {
	return float64(n)
}

// Dup returns the backing float64 value of the Node.
func (n Float) Dup() Node {
	return n
}

// Empty returns false.
func (n Float) Empty() bool {
	return false
}
