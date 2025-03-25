// Copyright (c) 2022, Peter Ohler, All rights reserved.

package jp

// Form represents a component of a JSON Path script and filter. They are used
// inspect a Script or Filter. The general template for a Form is (left op
// right). For an operations such as not (!) the right side is left as nil. As
// an example a Filter fragment of [?(@.x == 3)] whould be representing in a
// Form as
//
//	Form{Op: "==", Left: jp.Expr{jp.At('@'), jp.Child("x")}, Right: 3}.
type Form struct {
	// Op is the operation to perform.
	Op string

	// Left is the left side a form. The type can be a *Form, Expr, or any of
	// the simple types.
	Left any

	// Right is the left side a form. The type can be a *Form, Expr, or any of
	// the simple types.
	Right any
}

// Simplify the form.
func (f *Form) Simplify() any {
	simple := map[string]any{"op": f.Op}
	switch tv := f.Left.(type) {
	case Expr:
		simple["left"] = tv.String()
	case *Form:
		simple["left"] = tv.Simplify()
	default:
		simple["left"] = tv
	}
	switch tv := f.Right.(type) {
	case Expr:
		simple["right"] = tv.String()
	case *Form:
		simple["right"] = tv.Simplify()
	default:
		simple["right"] = tv
	}
	return simple
}
