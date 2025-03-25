// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"reflect"

	"github.com/ohler55/ojg/gen"
)

// Descent is used as a flag to indicate the path should be displayed in a
// recursive descent representation.
type Descent byte

// Append a fragment string representation of the fragment to the buffer
// then returning the expanded buffer.
func (f Descent) Append(buf []byte, bracket, first bool) []byte {
	if bracket {
		buf = append(buf, "[..]"...)
	} else {
		buf = append(buf, '.')
	}
	return buf
}

func (f Descent) locate(pp Expr, data any, rest Expr, max int) (locs []Expr) {
	if len(rest) == 0 { // last one
		loc := make(Expr, len(pp))
		copy(loc, pp)
		locs = append(locs, loc)
	} else {
		locs = locateContinueFrag(locs, pp, data, rest, max)
	}
	cp := append(pp, nil) // place holder
	mx := max
	switch td := data.(type) {
	case map[string]any:
		for k, v := range td {
			cp[len(pp)] = Child(k)
			if 0 < max {
				mx = max - len(locs)
				if mx <= 0 {
					break
				}
			}
			locs = append(locs, f.locate(cp, v, rest, mx)...)
		}
	case []any:
		for i, v := range td {
			cp[len(pp)] = Nth(i)
			if 0 < max {
				mx = max - len(locs)
				if mx <= 0 {
					break
				}
			}
			locs = append(locs, f.locate(cp, v, rest, mx)...)
		}
	case gen.Object:
		for k, v := range td {
			cp[len(pp)] = Child(k)
			if 0 < max {
				mx = max - len(locs)
				if mx <= 0 {
					break
				}
			}
			locs = append(locs, f.locate(cp, v, rest, mx)...)
		}
	case gen.Array:
		for i, v := range td {
			cp[len(pp)] = Nth(i)
			if 0 < max {
				mx = max - len(locs)
				if mx <= 0 {
					break
				}
			}
			locs = append(locs, f.locate(cp, v, rest, mx)...)
		}
	case Keyed:
		keys := td.Keys()
		for _, k := range keys {
			v, _ := td.ValueForKey(k)
			cp[len(pp)] = Child(k)
			if 0 < max {
				mx = max - len(locs)
				if mx <= 0 {
					break
				}
			}
			locs = append(locs, f.locate(cp, v, rest, mx)...)
		}
	case Indexed:
		size := td.Size()
		for i := 0; i < size; i++ {
			v := td.ValueAtIndex(i)
			cp[len(pp)] = Nth(i)
			if 0 < max {
				mx = max - len(locs)
				if mx <= 0 {
					break
				}
			}
			locs = append(locs, f.locate(cp, v, rest, mx)...)
		}
	case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
		int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
	default:
		rd := reflect.ValueOf(data)
		rt := rd.Type()
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
			rd = rd.Elem()
		}
		cp := append(pp, nil) // place holder
		switch rt.Kind() {
		case reflect.Struct:
			for i := rd.NumField() - 1; 0 <= i; i-- {
				rv := rd.Field(i)
				if rv.CanInterface() {
					cp[len(pp)] = Child(rt.Field(i).Name)
					if 0 < max {
						mx = max - len(locs)
						if mx <= 0 {
							break
						}
					}
					locs = append(locs, f.locate(cp, rv.Interface(), rest, mx)...)
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < rd.Len(); i++ {
				rv := rd.Index(i)
				if rv.CanInterface() {
					cp[len(pp)] = Nth(i)
					if 0 < max {
						mx = max - len(locs)
						if mx <= 0 {
							break
						}
					}
					locs = append(locs, f.locate(cp, rv.Interface(), rest, mx)...)
				}
			}
		}
	}
	return
}
