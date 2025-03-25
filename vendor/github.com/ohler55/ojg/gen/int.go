// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

import (
	"strconv"
)

// Int is a int64 Node.
type Int int64

// String returns a string representation of the Node.
func (n Int) String() string {
	return strconv.FormatInt(int64(n), 10)
}

// Alter returns the backing int64 value of the Node.
func (n Int) Alter() any {
	return int64(n)
}

// Simplify returns the backing int64 value of the Node.
func (n Int) Simplify() any {
	return int64(n)
}

// Dup returns the backing int64 value of the Node.
func (n Int) Dup() Node {
	return n
}

// Empty returns false.
func (n Int) Empty() bool {
	return false
}
