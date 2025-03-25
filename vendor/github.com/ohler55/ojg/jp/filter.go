// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/gen"
)

// Filter is a script used as a filter.
type Filter struct {
	Script
}

// NewFilter creates a new Filter.
func NewFilter(str string) (f *Filter, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ojg.NewError(r)
		}
	}()
	f = MustNewFilter(str)
	return
}

// MustNewFilter creates a new Filter and panics on error.
func MustNewFilter(str string) (f *Filter) {
	p := &parser{buf: []byte(str)}
	if len(p.buf) <= 3 ||
		p.buf[0] != '[' || p.buf[1] != '?' || p.buf[len(p.buf)-1] != ']' {
		panic(fmt.Errorf("a filter must start with a '[?' and end with ']'"))
	}
	p.buf = p.buf[2 : len(p.buf)-1]
	eq := precedentCorrect(p.readEq())
	eq = reduceGroups(eq, nil)

	return eq.Filter()
}

// String representation of the filter.
func (f *Filter) String() string {
	return string(f.Append([]byte{}, true, false))
}

// Append a fragment string representation of the fragment to the buffer
// then returning the expanded buffer.
func (f Filter) Append(buf []byte, _, _ bool) []byte {
	buf = append(buf, "[?"...)
	buf = f.Script.Append(buf)
	buf = append(buf, ']')

	return buf
}

func (f Filter) remove(value any) (out any, changed bool) {
	out = value
	switch tv := value.(type) {
	case []any:
		ns := make([]any, 0, len(tv))
		for _, v := range tv {
			if f.Match(v) {
				changed = true
			} else {
				ns = append(ns, v)
			}
		}
		if changed {
			out = ns
		}
	case map[string]any:
		for k, v := range tv {
			if f.Match(v) {
				delete(tv, k)
				changed = true
			}
		}
	case gen.Array:
		ns := make(gen.Array, 0, len(tv))
		for _, v := range tv {
			if f.Match(v) {
				changed = true
			} else {
				ns = append(ns, v)
			}
		}
		if changed {
			out = ns
		}
	case gen.Object:
		for k, v := range tv {
			if f.Match(v) {
				delete(tv, k)
				changed = true
			}
		}
	case RemovableIndexed:
		size := tv.Size()
		for i := (size - 1); i >= 0; i-- {
			v := tv.ValueAtIndex(i)
			if f.Match(v) {
				tv.RemoveValueAtIndex(i)
				changed = true
			}
		}
	case Keyed:
		keys := tv.Keys()
		for _, key := range keys {
			v, _ := tv.ValueForKey(key)
			if f.Match(v) {
				tv.RemoveValueForKey(key)
				changed = true
			}
		}
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice:
			// You would think that ns.SetLen() would work in a case like
			// this but it panics as unaddressable so instead the length
			// is calculated and then a second pass is made to assign the
			// new slice values.
			cnt := rv.Len()
			nc := 0
			for i := 0; i < cnt; i++ {
				if f.Match(rv.Index(i).Interface()) {
					changed = true
				} else {
					nc++
				}
			}
			if changed {
				changed = false
				ni := 0
				ns := reflect.MakeSlice(rv.Type(), nc, nc)
				for i := 0; i < cnt; i++ {
					iv := rv.Index(i)
					if f.Match(iv.Interface()) {
						changed = true
					} else {
						ns.Index(ni).Set(iv)
						ni++
					}
				}
				out = ns.Interface()
			}
		case reflect.Map:
			keys := rv.MapKeys()
			for _, k := range keys {
				mv := rv.MapIndex(k)
				if f.Match(mv.Interface()) {
					rv.SetMapIndex(k, reflect.Value{})
					changed = true
				}
			}
		}
	}
	return
}

func (f Filter) removeOne(value any) (out any, changed bool) {
	out = value
	switch tv := value.(type) {
	case []any:
		ns := make([]any, 0, len(tv))
		for _, v := range tv {
			if !changed && f.Match(v) {
				changed = true
			} else {
				ns = append(ns, v)
			}
		}
		if changed {
			out = ns
		}
	case map[string]any:
		if 0 < len(tv) {
			keys := make([]string, 0, len(tv))
			for k := range tv {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if f.Match(tv[k]) {
					delete(tv, k)
					changed = true
					break
				}
			}
		}
	case gen.Array:
		ns := make(gen.Array, 0, len(tv))
		for _, v := range tv {
			if !changed && f.Match(v) {
				changed = true
			} else {
				ns = append(ns, v)
			}
		}
		if changed {
			out = ns
		}
	case gen.Object:
		if 0 < len(tv) {
			keys := make([]string, 0, len(tv))
			for k := range tv {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if f.Match(tv[k]) {
					delete(tv, k)
					changed = true
					break
				}
			}
		}
	case RemovableIndexed:
		size := tv.Size()
		for i := 0; i < size; i++ {
			v := tv.ValueAtIndex(i)
			if f.Match(v) {
				tv.RemoveValueAtIndex(i)
				changed = true
				break
			}
		}
	case Keyed:
		keys := tv.Keys()
		sort.Strings(keys)
		for _, key := range keys {
			v, _ := tv.ValueForKey(key)
			if f.Match(v) {
				tv.RemoveValueForKey(key)
				changed = true
				break
			}
		}
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice:
			// You would think that ns.SetLen() would work in a case like
			// this but it panics as unaddressable so instead the length
			// is calculated and then a second pass is made to assign the
			// new slice values.
			cnt := rv.Len()
			nc := 0
			for i := 0; i < cnt; i++ {
				if !changed && f.Match(rv.Index(i).Interface()) {
					changed = true
				} else {
					nc++
				}
			}
			if changed {
				changed = false
				ni := 0
				ns := reflect.MakeSlice(rv.Type(), nc, nc)
				for i := 0; i < cnt; i++ {
					iv := rv.Index(i)
					if !changed && f.Match(iv.Interface()) {
						changed = true
					} else {
						ns.Index(ni).Set(iv)
						ni++
					}
				}
				out = ns.Interface()
			}
		case reflect.Map:
			keys := rv.MapKeys()
			sort.Slice(keys, func(i, j int) bool {
				return strings.Compare(keys[i].String(), keys[j].String()) < 0
			})
			for _, k := range keys {
				mv := rv.MapIndex(k)
				if f.Match(mv.Interface()) {
					rv.SetMapIndex(k, reflect.Value{})
					changed = true
					break
				}
			}
		}
	}
	return
}

func (f Filter) locate(pp Expr, data any, rest Expr, max int) (locs []Expr) {
	ns, lcs := f.evalWithRoot([]any{}, data, nil)
	stack, _ := ns.([]any)
	if len(rest) == 0 { // last one
		for _, lc := range lcs {
			locs = locateAppendFrag(locs, pp, lc)
			if 0 < max && max <= len(locs) {
				break
			}
		}
	} else {
		cp := append(pp, nil) // place holder
		for i, lc := range lcs {
			cp[len(pp)] = lc
			locs = locateContinueFrag(locs, cp, stack[i], rest, max)
			if 0 < max && max <= len(locs) {
				break
			}
		}
	}
	return
}
