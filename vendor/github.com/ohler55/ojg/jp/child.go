// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"reflect"

	"github.com/ohler55/ojg/gen"
)

// Child is a child operation for a JSON path expression.
type Child string

// Append a fragment string representation of the fragment to the buffer
// then returning the expanded buffer.
func (f Child) Append(buf []byte, bracket, first bool) []byte {
	if bracket || !f.tokenOk() {
		buf = append(buf, '[')
		buf = AppendString(buf, string(f), '\'')
		buf = append(buf, ']')
	} else {
		if !first {
			buf = append(buf, '.')
		}
		buf = append(buf, string(f)...)
	}
	return buf
}

func (f Child) tokenOk() bool {
	for _, b := range []byte(f) {
		if tokenMap[b] == '.' {
			return false
		}
	}
	return len(f) != 0
}

func (f Child) remove(value any) (out any, changed bool) {
	out = value
	key := string(f)
	switch tv := value.(type) {
	case map[string]any:
		if _, changed = tv[key]; changed {
			delete(tv, key)
		}
	case gen.Object:
		if _, changed = tv[key]; changed {
			delete(tv, key)
		}
	case Keyed:
		tv.RemoveValueForKey(key)
	default:
		if rt := reflect.TypeOf(value); rt != nil {
			// Can't remove a field from a struct so only a map can be modified.
			if rt.Kind() == reflect.Map {
				rv := reflect.ValueOf(value)
				rk := reflect.ValueOf(key)
				if rv.MapIndex(rk).IsValid() {
					rv.SetMapIndex(rk, reflect.Value{})
					changed = true
				}
			}
		}
	}
	return
}

func (f Child) locate(pp Expr, data any, rest Expr, max int) (locs []Expr) {
	var (
		v   any
		has bool
	)
	switch td := data.(type) {
	case map[string]any:
		v, has = td[string(f)]
	case gen.Object:
		v, has = td[string(f)]
	case Keyed:
		v, has = td.ValueForKey(string(f))
	default:
		v, has = pp.reflectGetChild(td, string(f))
	}
	if has {
		locs = locateNthChildHas(pp, f, v, rest, max)
	}
	return
}
