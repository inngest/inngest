// Copyright (c) 2022, Peter Ohler, All rights reserved.

package jp

import (
	"fmt"
	"strings"

	"github.com/ohler55/ojg"
)

type remover interface {
	remove(element any) (altered any, changed bool)
}

type oneRemover interface {
	removeOne(element any) (altered any, changed bool)
}

// MustRemove removes matching nodes and panics on an expression error but
// silently makes no changes if there is no match for the expression. Removed
// slice elements are removed and the remaining elements are moveed to fill in
// the removed element. The slice is shortened.
func (x Expr) MustRemove(data any) any {
	if len(x) == 0 {
		panic("can not remove with an empty expression")
	}
	last := x[len(x)-1]

	sx := x[:len(x)-1]
	if len(sx) == 0 {
		sx = Expr{Root(0)}
	}
	if r, ok := last.(remover); ok {
		return sx.modify(data, r.remove, false)
	}
	ta := strings.Split(fmt.Sprintf("%T", last), ".")
	panic(fmt.Sprintf("can not remove with an expression where the last fragment is a %s", ta[len(ta)-1]))
}

// MustRemoveOne removes matching nodes and panics on an expression error
// but silently makes no changes if there is no match for the
// expression. Removed slice elements are removed and the remaining elements
// are moveed to fill in the removed element. The slice is shortened.
func (x Expr) MustRemoveOne(data any) any {
	if len(x) == 0 {
		panic("can not remove with an empty expression")
	}
	last := x[len(x)-1]

	sx := x[:len(x)-1]
	if len(sx) == 0 {
		sx = Expr{Root(0)}
	}
	if r, ok := last.(oneRemover); ok {
		return sx.modify(data, r.removeOne, true)
	}
	if r, ok := last.(remover); ok {
		return sx.modify(data, r.remove, true)
	}
	ta := strings.Split(fmt.Sprintf("%T", last), ".")
	panic(fmt.Sprintf("can not remove with an expression where the last fragment is a %s", ta[len(ta)-1]))
}

// Remove removes matching nodes. An error is returned for an expression error
// but silently makes no changes if there is no match for the
// expression. Removed slice elements are removed and the remaining elements
// are moveed to fill in the removed element. The slice is shortened.
func (x Expr) Remove(data any) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ojg.NewError(r)
		}
	}()
	result = x.MustRemove(data)

	return
}

// RemoveOne removes at most one node. An error is returned for an expression
// error but silently makes no changes if there is no match for the
// expression. Removed slice elements are removed and the remaining elements
// are moveed to fill in the removed element. The slice is shortened.
func (x Expr) RemoveOne(data any) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ojg.NewError(r)
		}
	}()
	result = x.MustRemoveOne(data)

	return
}
