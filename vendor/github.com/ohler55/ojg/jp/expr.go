// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"unsafe"
)

// Expr is a JSON path expression composed of fragments. An Expr implements
// JSONPath as described by https://goessner.net/articles/JsonPath. Where the
// definition is unclear Oj has implemented the description based on the best
// judgement of the author.
type Expr []Frag

// String returns a string representation of the expression.
func (x Expr) String() string {
	return string(x.Append(nil))
}

// BracketString returns a string representation of the expression using the
// bracket notation.
func (x Expr) BracketString() string {
	return string(x.Append(nil, true))
}

// Append a string representation of the expression to a byte slice and return
// the expanded buffer.
func (x Expr) Append(buf []byte, brackets ...bool) []byte {
	bracket := 0 < len(brackets) && brackets[0]
	for i, frag := range x {
		if _, ok := frag.(Bracket); ok {
			bracket = true
			continue
		}
		buf = frag.Append(buf, bracket, i == 0)
	}
	if 0 < len(x) {
		if _, ok := x[len(x)-1].(Descent); ok {
			buf = append(buf, '.')
		}
	}
	return buf
}

func isNil(v any) bool {
	return (*[2]uintptr)(unsafe.Pointer(&v))[1] == 0
}
