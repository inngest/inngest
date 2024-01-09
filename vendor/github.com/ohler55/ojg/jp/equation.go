// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"regexp"
	"strconv"
)

// Equation represents JSON Path script and filter equations. They are used to
// build a script. The purpose of the Equation is to allow scripts or filters
// to be created without using a parser which could return an error if an
// invalid string representation of the script is provided.
type Equation struct {
	o      *op
	result any
	left   *Equation
	right  *Equation
}

// Script creates and returns a Script that implements the equation.
func (e *Equation) Script() (s *Script) {
	s = &Script{template: e.buildScript([]any{})}
	return
}

// Filter creates and returns a Script that implements the equation.
func (e *Equation) Filter() (f *Filter) {
	f = &Filter{Script: Script{template: e.buildScript([]any{})}}
	return
}

// ConstNil creates and returns an Equation for a constant of nil.
func ConstNil() *Equation {
	return &Equation{result: nil}
}

// ConstNothing creates and returns an Equation for a constant of nothing.
func ConstNothing() *Equation {
	return &Equation{result: Nothing}
}

// ConstBool creates and returns an Equation for a bool constant.
func ConstBool(b bool) *Equation {
	return &Equation{result: b}
}

// ConstInt creates and returns an Equation for an int64 constant.
func ConstInt(i int64) *Equation {
	return &Equation{result: i}
}

// ConstFloat creates and returns an Equation for a float64 constant.
func ConstFloat(f float64) *Equation {
	return &Equation{result: f}
}

// ConstString creates and returns an Equation for a string constant.
func ConstString(s string) *Equation {
	return &Equation{result: s}
}

// ConstList creates and returns an Equation for an []any constant.
func ConstList(list []any) *Equation {
	return &Equation{result: list}
}

// ConstRegex creates and returns an Equation for a regex constant.
func ConstRegex(rx *regexp.Regexp) *Equation {
	return &Equation{result: rx}
}

// Get creates and returns an Equation for an expression get of the form
// @.child.
func Get(x Expr) *Equation {
	return &Equation{o: get, left: &Equation{result: x}}
}

// Eq creates and returns an Equation for an == operator.
func Eq(left, right *Equation) *Equation {
	return &Equation{o: eq, left: left, right: right}
}

// Neq creates and returns an Equation for a != operator.
func Neq(left, right *Equation) *Equation {
	return &Equation{o: neq, left: left, right: right}
}

// Lt creates and returns an Equation for a < operator.
func Lt(left, right *Equation) *Equation {
	return &Equation{o: lt, left: left, right: right}
}

// Gt creates and returns an Equation for a > operator.
func Gt(left, right *Equation) *Equation {
	return &Equation{o: gt, left: left, right: right}
}

// Lte creates and returns an Equation for a <= operator.
func Lte(left, right *Equation) *Equation {
	return &Equation{o: lte, left: left, right: right}
}

// Gte creates and returns an Equation for a >= operator.
func Gte(left, right *Equation) *Equation {
	return &Equation{o: gte, left: left, right: right}
}

// Or creates and returns an Equation for a || operator.
func Or(left, right *Equation) *Equation {
	return &Equation{o: or, left: left, right: right}
}

// And creates and returns an Equation for a && operator.
func And(left, right *Equation) *Equation {
	return &Equation{o: and, left: left, right: right}
}

// Not creates and returns an Equation for a ! operator.
func Not(arg *Equation) *Equation {
	return &Equation{o: not, left: arg}
}

// Add creates and returns an Equation for a + operator.
func Add(left, right *Equation) *Equation {
	return &Equation{o: add, left: left, right: right}
}

// Sub creates and returns an Equation for a - operator.
func Sub(left, right *Equation) *Equation {
	return &Equation{o: sub, left: left, right: right}
}

// Multiply creates and returns an Equation for a * operator.
func Multiply(left, right *Equation) *Equation {
	return &Equation{o: mult, left: left, right: right}
}

// Divide creates and returns an Equation for a / operator.
func Divide(left, right *Equation) *Equation {
	return &Equation{o: divide, left: left, right: right}
}

// In creates and returns an Equation for an in operator.
func In(left, right *Equation) *Equation {
	return &Equation{o: in, left: left, right: right}
}

// Empty creates and returns an Equation for an empty operator.
func Empty(left, right *Equation) *Equation {
	return &Equation{o: empty, left: left, right: right}
}

// Has creates and returns an Equation for a has operator.
func Has(left, right *Equation) *Equation {
	return &Equation{o: has, left: left, right: right}
}

// Exists creates and returns an Equation for a exists operator.
func Exists(left, right *Equation) *Equation {
	return &Equation{o: exists, left: left, right: right}
}

// Regex creates and returns an Equation for a regex operator.
func Regex(left, right *Equation) *Equation {
	return &Equation{o: rx, left: left, right: right}
}

// Length creates and returns an Equation for a length function.
func Length(x Expr) *Equation {
	return &Equation{o: length, left: &Equation{result: x}}
}

// Count creates and returns an Equation for a count function.
func Count(x Expr) *Equation {
	return &Equation{o: count, left: &Equation{result: x}}
}

// Match creates and returns an Equation for a match function.
func Match(left, right *Equation) *Equation {
	return &Equation{o: match, left: left, right: right}
}

// Search creates and returns an Equation for a search function.
func Search(left, right *Equation) *Equation {
	return &Equation{o: search, left: left, right: right}
}

// Append a fragment string representation of the fragment to the buffer
// then returning the expanded buffer.
func (e *Equation) Append(buf []byte, parens bool) []byte {
	if e.o != nil {
		switch e.o.code {
		case not.code, length.code, count.code, match.code, search.code:
			parens = false
		}
	}
	if parens {
		buf = append(buf, '(')
	}
	if e.o == nil {
		buf = e.appendValue(buf, e.result)
	} else {
		switch e.o.code {
		case not.code:
			buf = append(buf, '!')
			if e.left != nil {
				buf = e.left.Append(buf, e.left.o != nil && e.left.o.prec >= e.o.prec)
			}
		case get.code:
			if e.left != nil {
				buf = e.appendValue(buf, e.left.result)
			}
		case length.code, count.code:
			buf = append(buf, e.o.name...)
			buf = append(buf, '(')
			buf = e.appendValue(buf, e.left.result)
			buf = append(buf, ')')
		case match.code, search.code:
			buf = append(buf, e.o.name...)
			buf = append(buf, '(')
			buf = e.left.Append(buf, false)
			buf = append(buf, ',', ' ')
			buf = e.right.Append(buf, false)
			buf = append(buf, ')')
		default:
			if e.left != nil {
				buf = e.left.Append(buf, e.left.o != nil && e.left.o.prec >= e.o.prec)
			}
			buf = append(buf, ' ')
			buf = append(buf, e.o.name...)
			buf = append(buf, ' ')
			if e.right != nil {
				buf = e.right.Append(buf, e.left.o != nil && e.left.o.prec >= e.o.prec)
			}
		}
	}
	if parens {
		buf = append(buf, ')')
	}
	return buf
}

func (e *Equation) appendValue(buf []byte, v any) []byte {
	switch tv := v.(type) {
	case nil:
		buf = append(buf, "null"...)
	case nothing:
		buf = append(buf, "Nothing"...)
	case string:
		buf = AppendString(buf, tv, '\'')
	case int64:
		buf = append(buf, strconv.FormatInt(tv, 10)...)
	case float64:
		buf = append(buf, strconv.FormatFloat(tv, 'g', -1, 64)...)
	case bool:
		if tv {
			buf = append(buf, "true"...)
		} else {
			buf = append(buf, "false"...)
		}
	case []any:
		buf = append(buf, '[')
		for i, ev := range tv {
			if 0 < i {
				buf = append(buf, ',')
			}
			buf = e.appendValue(buf, ev)
		}
		buf = append(buf, ']')
	case Expr:
		buf = tv.Append(buf)
	case *regexp.Regexp:
		buf = AppendString(buf, tv.String(), '/')
	}
	return buf
}

// String representation of the equation.
func (e *Equation) String() string {
	return string(e.Append([]byte{}, true))
}

func (e *Equation) buildScript(stack []any) []any {
	if e.o == nil {
		stack = append(stack, e.result)
		return stack
	}
	switch e.o.code {
	case get.code:
		if e.left != nil {
			stack = append(stack, e.left.result) // should always be an Expr
		}
	case not.code, length.code, count.code:
		stack = append(stack, e.o)
		if e.left == nil {
			stack = append(stack, nil)
		} else {
			stack = e.left.buildScript(stack)
		}
	default:
		stack = append(stack, e.o)
		if e.left == nil {
			stack = append(stack, nil)
		} else {
			stack = e.left.buildScript(stack)
		}
		if e.right == nil {
			stack = append(stack, nil)
		} else {
			stack = e.right.buildScript(stack)
		}
	}
	return stack
}
