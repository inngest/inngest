// Copyright (c) 2023, Peter Ohler, All rights reserved.

package jp

import (
	"reflect"

	"github.com/ohler55/ojg/gen"
)

// Locate the values described by the Expr and return a slice of normalized
// paths to those values in the data. The returned slice is limited to the max
// specified. A max of 0 or less indicates there is no maximum.
func (x Expr) Locate(data any, max int) (locs []Expr) {
	if 0 < len(x) {
		locs = x[0].locate(nil, data, x[1:], max)
	}
	return
}

func locateNthChildHas(pp Expr, f Frag, v any, rest Expr, max int) (locs []Expr) {
	if len(rest) == 0 { // last one
		loc := make(Expr, len(pp)+1)
		copy(loc, pp)
		loc[len(pp)] = f
		locs = []Expr{loc}
	} else {
		switch v.(type) {
		case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
			int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
		case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
			locs = rest[0].locate(append(pp, f), v, rest[1:], max)
		default:
			if rt := reflect.TypeOf(v); rt != nil {
				switch rt.Kind() {
				case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
					locs = rest[0].locate(append(pp, f), v, rest[1:], max)
				}
			}
		}
	}
	return
}

func locateAppendFrag(locs []Expr, pp Expr, f Frag) []Expr {
	loc := make(Expr, len(pp)+1)
	copy(loc, pp)
	loc[len(pp)] = f

	return append(locs, loc)
}

func locateContinueFrag(locs []Expr, cp Expr, v any, rest Expr, max int) []Expr {
	mx := max
	if 0 < max {
		mx = max - len(locs)
	}
	switch v.(type) {
	case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
		int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
	case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
		locs = append(locs, rest[0].locate(cp, v, rest[1:], mx)...)
	default:
		if rt := reflect.TypeOf(v); rt != nil {
			switch rt.Kind() {
			case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
				locs = append(locs, rest[0].locate(cp, v, rest[1:], mx)...)
			}
		}
	}
	return locs
}
