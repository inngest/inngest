// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

// X creates an empty Expr.
func X() Expr {
	return Expr{}
}

// A creates an Expr with a At (@) fragment.
func A() Expr {
	return Expr{At('@')}
}

// B creates an Expr with a Bracket fragment.
func B() Expr {
	return Expr{Bracket(' ')}
}

// C creates an Expr with a Child fragment.
func C(key string) Expr {
	return Expr{Child(key)}
}

// D creates an Expr with a recursive Descent fragment.
func D() Expr {
	return Expr{Descent('.')}
}

// F creates an Expr with a Filter fragment.
func F(e *Equation) Expr {
	return Expr{e.Filter()}
}

// N creates an Expr with an Nth fragment.
func N(n int) Expr {
	return Expr{Nth(n)}
}

// R creates an Expr with a Root fragment.
func R() Expr {
	return Expr{Root('$')}
}

// S creates an Expr with a Slice fragment.
func S(start int, rest ...int) Expr {
	return Expr{Slice(append([]int{start}, rest...))}
}

// U creates an Expr with an Union fragment.
func U(keys ...any) Expr {
	return Expr{NewUnion(keys...)}
}

// W creates an Expr with a Wildcard fragment.
func W() Expr {
	return Expr{Wildcard('*')}
}

// A appends an At fragment to the Expr.
func (x Expr) A() Expr {
	return append(x, At('@'))
}

// At appends an At fragment to the Expr.
func (x Expr) At() Expr {
	return append(x, At('@'))
}

// B appends a Bracket fragment to the Expr.
func (x Expr) B() Expr {
	return append(x, Bracket(' '))
}

// C appends a Child fragment to the Expr.
func (x Expr) C(key string) Expr {
	return append(x, Child(key))
}

// Child appends a Child fragment to the Expr.
func (x Expr) Child(key string) Expr {
	return append(x, Child(key))
}

// D appends a recursive Descent fragment to the Expr.
func (x Expr) D() Expr {
	return append(x, Descent('.'))
}

// Descent appends a recursive Descent fragment to the Expr.
func (x Expr) Descent() Expr {
	return append(x, Descent('.'))
}

// F appends a Filter fragment to the Expr.
func (x Expr) F(e *Equation) Expr {
	return append(x, e.Filter())
}

// Filter appends a Filter fragment to the Expr.
func (x Expr) Filter(e *Equation) Expr {
	return append(x, e.Filter())
}

// N appends an Nth fragment to the Expr.
func (x Expr) N(n int) Expr {
	return append(x, Nth(n))
}

// Nth appends an Nth fragment to the Expr.
func (x Expr) Nth(n int) Expr {
	return append(x, Nth(n))
}

// R appends a Root fragment to the Expr.
func (x Expr) R() Expr {
	return append(x, Root('$'))
}

// Root appends a Root fragment to the Expr.
func (x Expr) Root() Expr {
	return append(x, Root('$'))
}

// S appends a Slice fragment to the Expr.
func (x Expr) S(start int, rest ...int) Expr {
	return append(x, Slice(append([]int{start}, rest...)))
}

// Slice appends a Slice fragment to the Expr.
func (x Expr) Slice(start int, rest ...int) Expr {
	return append(x, Slice(append([]int{start}, rest...)))
}

// U appends a Union fragment to the Expr.
func (x Expr) U(keys ...any) Expr {
	return append(x, NewUnion(keys...))
}

// Union appends a Union fragment to the Expr.
func (x Expr) Union(keys ...any) Expr {
	return append(x, NewUnion(keys...))
}

// W appends a Wildcard fragment to the Expr.
func (x Expr) W() Expr {
	return append(x, Wildcard('*'))
}

// Wildcard appends a Wildcard fragment to the Expr.
func (x Expr) Wildcard() Expr {
	return append(x, Wildcard('*'))
}
