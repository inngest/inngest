// Copyright (c) 2023, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strings"
	"time"
)

// Filter is a simple filter for matching against arbitrary date.
type Filter map[string]any

// NewFilter creates a new filter from the spec which should be a map where
// the keys are simple paths of keys delimited by the dot ('.') character. An
// example is "top.child.grandchild". The matching will either match the key
// when the data is traversed directly or in the case of a slice the elements
// of the slice are also traversed. Generally a Filter is created and reused
// as there is some overhead in creating the Filter. An alternate format is a
// nested set of maps.
func NewFilter(spec map[string]any) Filter {
	f := Filter{}
	f.add(spec)
	return f
}

func (f Filter) add(spec map[string]any) {
	for k, v := range spec {
		path := strings.Split(k, ".")
		f2 := f
		for _, k2 := range path[:len(path)-1] {
			sub, _ := f2[k2].(Filter)
			if sub == nil {
				sub = Filter{}
				f2[k2] = sub
			}
			f2 = sub
		}
		k2 := path[len(path)-1]
		switch tv := v.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			f2[k2], _ = asInt(tv)
		case float32, float64:
			f2[k2], _ = asFloat(tv)
		case map[string]any:
			sub, _ := f2[k2].(Filter)
			if sub == nil {
				sub = NewFilter(map[string]any{})
				f2[k2] = sub
			}
			sub.add(tv)
		default:
			f2[k2] = v
		}
	}
}

// Match returns true if the target matches the Filter.
func (f Filter) Match(data any) bool {
	return match(f, data)
}

func match(target, data any) (same bool) {
top:
	switch tv := data.(type) {
	case map[string]any:
		if f, ok := target.(Filter); ok {
			same = true
			for k, fv := range f {
				if !match(fv, tv[k]) {
					return false
				}
			}
		}
	case []any:
		for _, v := range tv {
			if same = match(target, v); same {
				break
			}
		}
	case nil:
		same = target == nil
	case bool:
		b, ok := target.(bool)
		same = ok && tv == b
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		v, _ := asInt(tv)
		i, ok := asInt(target)
		same = ok && v == i
	case float32, float64:
		v, _ := asFloat(tv)
		ff, ok := asFloat(target)
		same = ok && v == ff
	case string:
		fs, ok := target.(string)
		same = ok && fs == tv
	case time.Time:
		ft, ok := target.(time.Time)
		same = ok && ft.Equal(tv)
	case Simplifier:
		data = tv.Simplify()
		goto top
	default:
		data = reflectValue(reflect.ValueOf(tv), tv, &Options{})
		goto top
	}
	return
}

// Simplify returns a simplified representation of the Filter.
func (f Filter) Simplify() any {
	simple := map[string]any{}
	for k, v := range f {
		if f2, ok := v.(Filter); ok {
			simple[k] = f2.Simplify()
		} else {
			simple[k] = v
		}
	}
	return simple
}
