// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ohler55/ojg/gen"
)

// Union is a union operation for a JSON path expression which is a union of a
// Child and Nth fragment.
type Union []any

// Append a fragment string representation of the fragment to the buffer
// then returning the expanded buffer.
func (f Union) Append(buf []byte, _, _ bool) []byte {
	buf = append(buf, '[')
	for i, x := range f {
		if 0 < i {
			buf = append(buf, ',')
		}
		switch tx := x.(type) {
		case string:
			buf = append(buf, '\'')
			buf = append(buf, tx...)
			buf = append(buf, '\'')
		case int64:
			buf = append(buf, strconv.FormatInt(tx, 10)...)
		}
	}
	buf = append(buf, ']')

	return buf
}

// NewUnion creates a new Union with the provide keys.
func NewUnion(keys ...any) (u Union) {
	for _, k := range keys {
		switch tk := k.(type) {
		case string:
			u = append(u, k)
		case int:
			u = append(u, int64(tk))
		case int64:
			u = append(u, tk)
		}
	}
	return
}

func (f Union) hasN(n int64) bool {
	for _, x := range f {
		if ix, ok := x.(int64); ok && ix == n {
			return true
		}
	}
	return false
}

func (f Union) hasKey(key string) bool {
	for _, x := range f {
		if sx, ok := x.(string); ok && sx == key {
			return true
		}
	}
	return false
}

func (f Union) removeOne(value any) (out any, changed bool) {
	out = value
	switch tv := value.(type) {
	case []any:
		ns := make([]any, 0, len(tv))
		for i, v := range tv {
			if !changed && f.hasN(int64(i)) {
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
				if f.hasKey(k) {
					delete(tv, k)
					changed = true
					break
				}
			}
		}
	case gen.Array:
		ns := make(gen.Array, 0, len(tv))
		for i, v := range tv {
			if !changed && f.hasN(int64(i)) {
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
				if f.hasKey(k) {
					delete(tv, k)
					changed = true
					break
				}
			}
		}
	case RemovableIndexed:
		size := tv.Size()
		for i := 0; i < size; i++ {
			if f.hasN(int64(i)) {
				tv.RemoveValueAtIndex(i)
				changed = true
				break
			}
		}
	case Keyed:
		keys := tv.Keys()
		sort.Strings(keys)
		for _, k := range keys {
			if f.hasKey(k) {
				tv.RemoveValueForKey(k)
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
				if !changed && f.hasN(int64(i)) {
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
					if !changed && f.hasN(int64(i)) {
						changed = true
					} else {
						ns.Index(ni).Set(rv.Index(i))
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
				if f.hasKey(k.String()) {
					rv.SetMapIndex(k, reflect.Value{})
					changed = true
					break
				}
			}
		}
	}
	return
}

func (f Union) remove(value any) (out any, changed bool) {
	out = value
	switch tv := value.(type) {
	case []any:
		ns := make([]any, 0, len(tv))
		for i, v := range tv {
			if f.hasN(int64(i)) {
				changed = true
			} else {
				ns = append(ns, v)
			}
		}
		if changed {
			out = ns
		}
	case map[string]any:
		for k := range tv {
			if f.hasKey(k) {
				delete(tv, k)
				changed = true
			}
		}
	case gen.Array:
		ns := make(gen.Array, 0, len(tv))
		for i, v := range tv {
			if f.hasN(int64(i)) {
				changed = true
			} else {
				ns = append(ns, v)
			}
		}
		if changed {
			out = ns
		}
	case gen.Object:
		for k := range tv {
			if f.hasKey(k) {
				delete(tv, k)
				changed = true
			}
		}
	case RemovableIndexed:
		size := tv.Size()
		for i := (size - 1); i >= 0; i-- {
			if f.hasN(int64(i)) {
				tv.RemoveValueAtIndex(i)
				changed = true
			}
		}
	case Keyed:
		keys := tv.Keys()
		for _, k := range keys {
			if f.hasKey(k) {
				tv.RemoveValueForKey(k)
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
				if f.hasN(int64(i)) {
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
					if f.hasN(int64(i)) {
						changed = true
					} else {
						ns.Index(ni).Set(rv.Index(i))
						ni++
					}
				}
				out = ns.Interface()
			}
		case reflect.Map:
			keys := rv.MapKeys()
			for _, k := range keys {
				if f.hasKey(k.String()) {
					rv.SetMapIndex(k, reflect.Value{})
					changed = true
				}
			}
		}
	}
	return
}

func (f Union) locate(pp Expr, data any, rest Expr, max int) (locs []Expr) {
	var (
		v   any
		has bool
		lf  Frag
	)
	for _, u := range f {
		has = false
		switch tu := u.(type) {
		case string:
			switch td := data.(type) {
			case map[string]any:
				v, has = td[tu]
			case Keyed:
				v, has = td.ValueForKey(tu)
			case gen.Object:
				v, has = td[tu]
			default:
				v, has = pp.reflectGetChild(td, tu)
			}
			lf = Child(tu)
		case int64:
			i := int(tu)
			switch td := data.(type) {
			case []any:
				if i < 0 {
					i = len(td) + i
				}
				if 0 <= i && i < len(td) {
					v = td[i]
					has = true
				}
			case Indexed:
				if i < 0 {
					i = td.Size() + i
				}
				if 0 <= i && i < td.Size() {
					v = td.ValueAtIndex(i)
					has = true
				}
			case gen.Array:
				if i < 0 {
					i = len(td) + i
				}
				if 0 <= i && i < len(td) {
					v = td[i]
					has = true
				}
			default:
				v, has = pp.reflectGetNth(td, i)
			}
			lf = Nth(i)
		}
		if has {
			if len(rest) == 0 { // last one
				locs = locateAppendFrag(locs, pp, lf)
			} else {
				locs = locateContinueFrag(locs, append(pp, lf), v, rest, max)
			}
			if 0 < max && max <= len(locs) {
				break
			}
		}
	}
	return
}
