// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

import (
	"unsafe"
)

// Array represents an array of nodes.
type Array []Node

// EmptyArray is a array of nodes of zero length.
var EmptyArray = Array{}

func (n Array) String() string {
	b := []byte{'['}
	for i, m := range n {
		if 0 < i {
			b = append(b, ',')
		}
		if m == nil {
			b = append(b, "null"...)
		} else {
			b = append(b, m.String()...)
		}
	}
	b = append(b, ']')

	return string(b)
}

// Alter the array into a simple []any.
func (n Array) Alter() any {
	var simple []any

	if n != nil {
		simple = *(*[]any)(unsafe.Pointer(&n))
		for i, m := range n {
			if m == nil {
				simple[i] = nil
			} else {
				simple[i] = m.Alter()
			}
		}
	}
	return simple
}

// Simplify creates a simplified version of the Node as a []any.
func (n Array) Simplify() any {
	var dup []any

	if n != nil {
		dup = make([]any, 0, len(n))
		for _, m := range n {
			if m == nil {
				dup = append(dup, nil)
			} else {
				dup = append(dup, m.Simplify())
			}
		}
	}
	return dup
}

// Dup creates a deep duplicate of the Node.
func (n Array) Dup() Node {
	var a Array

	if n != nil {
		a = make(Array, 0, len(n))
		for _, m := range n {
			if m == nil {
				a = append(a, nil)
			} else {
				a = append(a, m.Dup())
			}
		}
	}
	return a
}

// Empty returns true if the Array is empty.
func (n Array) Empty() bool {
	return len(n) == 0
}
