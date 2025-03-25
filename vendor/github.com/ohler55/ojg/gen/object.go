// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

import (
	"sort"
	"unsafe"
)

// Sort if true sorts Object keys on output.
var Sort = false

// Object is a map of Nodes with string keys.
type Object map[string]Node

// String returns a string representation of the Node.
func (n Object) String() string {
	b := []byte{'{'}
	first := true

	if Sort {
		keys := make([]string, 0, len(n))
		for k := range n {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if 0 < i {
				b = append(b, ',')
			}
			b = append(b, '"')
			b = append(b, k...)
			b = append(b, '"')
			b = append(b, ':')
			if m := n[k]; m == nil {
				b = append(b, "null"...)
			} else {
				b = append(b, m.String()...)
			}
		}
	} else {
		for k, m := range n {
			if first {
				first = false
			} else {
				b = append(b, ',')
			}
			b = append(b, '"')
			b = append(b, k...)
			b = append(b, '"')
			b = append(b, ':')
			if m == nil {
				b = append(b, "null"...)
			} else {
				b = append(b, m.String()...)
			}
		}
	}
	b = append(b, '}')

	return string(b)
}

// Alter the Object into a simple map[string]any.
func (n Object) Alter() any {
	var simple map[string]any

	if n != nil {
		simple = *(*map[string]any)(unsafe.Pointer(&n))
		for k, m := range n {
			if m == nil {
				simple[k] = nil
			} else {
				simple[k] = m.Alter()
			}
		}
	}
	return simple
}

// Simplify creates a simplified version of the Node as a
// map[string]any.
func (n Object) Simplify() any {
	var dup map[string]any

	if n != nil {
		dup = map[string]any{}
		for k, m := range n {
			if m == nil {
				dup[k] = m
			} else {
				dup[k] = m.Simplify()
			}
		}
	}
	return dup
}

// Dup creates a deep duplicate of the Node.
func (n Object) Dup() Node {
	var o Object

	if n != nil {
		o = Object{}
		for k, m := range n {
			if m == nil {
				o[k] = nil
			} else {
				o[k] = m.Dup()
			}
		}
	}
	return o
}

// Empty returns true if the Object is empty.
func (n Object) Empty() bool {
	return len(n) == 0
}
