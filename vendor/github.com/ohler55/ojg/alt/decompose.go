// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

import (
	"encoding/base64"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/ohler55/ojg"
)

// 23 for fraction in IEEE 754 which amounts to 7 significant digits. Use base
// 10 so that numbers look correct when displayed in base 10.
const fracMax = 10000000.0

func decompose(v any, opt *Options) any {
	switch tv := v.(type) {
	case nil, bool, int64, float64, string:
	case int:
		v = int64(tv)
	case int8:
		v = int64(tv)
	case int16:
		v = int64(tv)
	case int32:
		v = int64(tv)
	case uint:
		v = int64(tv)
	case uint8:
		v = int64(tv)
	case uint16:
		v = int64(tv)
	case uint32:
		v = int64(tv)
	case uint64:
		v = int64(tv)
	case float32:
		// This small rounding makes the conversion from 32 bit to 64 bit
		// display nicer.
		f, i := math.Frexp(float64(tv))
		f = float64(int64(f*fracMax)) / fracMax
		v = math.Ldexp(f, i)
	case []any:
		a := make([]any, len(tv))
		for i, m := range tv {
			a[i] = decompose(m, opt)
		}
		v = a
	case map[string]any:
		o := map[string]any{}
		for k, m := range tv {
			condMapSet(o, k, decompose(m, opt), opt)
		}
		v = o
	case []byte:
		switch opt.BytesAs {
		case ojg.BytesAsBase64:
			v = base64.StdEncoding.EncodeToString(tv)
		case ojg.BytesAsArray:
			a := make([]any, len(tv))
			for i, m := range tv {
				a[i] = decompose(m, opt)
			}
			v = a
		default:
			v = string(tv)
		}
	case time.Time:
		v = opt.DecomposeTime(tv)
	default:
		if simp, _ := v.(Simplifier); simp != nil {
			return decompose(simp.Simplify(), opt)
		}
		return reflectValue(reflect.ValueOf(v), v, opt)
	}
	return v
}

func alter(v any, opt *Options) any {
	switch tv := v.(type) {
	case bool, nil, int64, float64, string, time.Time:
	case int:
		v = int64(tv)
	case int8:
		v = int64(tv)
	case int16:
		v = int64(tv)
	case int32:
		v = int64(tv)
	case uint:
		v = int64(tv)
	case uint8:
		v = int64(tv)
	case uint16:
		v = int64(tv)
	case uint32:
		v = int64(tv)
	case uint64:
		v = int64(tv)
	case float32:
		// This small rounding makes the conversion from 32 bit to 64 bit
		// display nicer.
		f, i := math.Frexp(float64(tv))
		f = float64(int64(f*fracMax)) / fracMax
		v = math.Ldexp(f, i)
	case []any:
		for i, m := range tv {
			tv[i] = alter(m, opt)
		}
	case map[string]any:
		for k, m := range tv {
			mv := alter(m, opt)
			switch tmv := mv.(type) {
			case nil:
				if opt.OmitNil || opt.OmitEmpty {
					delete(tv, k)
					continue
				}
			case string:
				if opt.OmitEmpty && len(tmv) == 0 {
					delete(tv, k)
					continue
				}
			case []any:
				if opt.OmitEmpty && len(tmv) == 0 {
					delete(tv, k)
					continue
				}
			case map[string]any:
				if opt.OmitEmpty && len(tmv) == 0 {
					delete(tv, k)
					continue
				}
			case bool:
				if opt.OmitEmpty && !tmv {
					delete(tv, k)
					continue
				}
			case int64:
				if opt.OmitEmpty && tmv == 0 {
					delete(tv, k)
					continue
				}
			}
			tv[k] = mv
		}
	case []byte:
		switch opt.BytesAs {
		case ojg.BytesAsBase64:
			v = base64.StdEncoding.EncodeToString(tv)
		case ojg.BytesAsArray:
			a := make([]any, len(tv))
			for i, m := range tv {
				a[i] = decompose(m, opt)
			}
			v = a
		default:
			v = string(tv)
		}
	default:
		if simp, _ := v.(Simplifier); simp != nil {
			return alter(simp.Simplify(), opt)
		}
		return reflectValue(reflect.ValueOf(v), v, opt)
	}
	return v
}

func reflectValue(rv reflect.Value, val any, opt *Options) (v any) {
	switch rv.Kind() {
	case reflect.Invalid, reflect.Uintptr, reflect.UnsafePointer, reflect.Chan, reflect.Func, reflect.Interface:
		v = nil
	case reflect.Complex64, reflect.Complex128:
		v = reflectComplex(rv, opt)
	case reflect.Map:
		v = reflectMap(rv, opt)
	case reflect.Ptr:
		elem := rv.Elem()
		if elem.IsValid() && elem.CanInterface() {
			v = reflectValue(elem, elem.Interface(), opt)
		} else {
			v = nil
		}
	case reflect.Slice, reflect.Array:
		v = reflectArray(rv, opt)
	case reflect.Struct:
		v = reflectStruct(rv, val, opt)
	case reflect.String:
		v = rv.String()
	case reflect.Bool:
		v = rv.Bool()
	case reflect.Float32, reflect.Float64:
		v = rv.Float()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v = rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v = rv.Uint()
	}
	return
}

func reflectStruct(rv reflect.Value, val any, opt *Options) any {
	if !rv.CanAddr() {
		return reflectEmbed(rv, val, opt)
	}
	obj := map[string]any{}
	si := getSinfo(val, opt.OmitEmpty)
	t := si.rt
	if 0 < len(opt.CreateKey) {
		if opt.FullTypePath {
			obj[opt.CreateKey] = t.PkgPath() + "/" + t.Name()
		} else {
			obj[opt.CreateKey] = t.Name()
		}
	}
	fields := si.getFields(opt)
	addr := rv.UnsafeAddr()
	for _, fi := range fields {
		if v, fv, omit := fi.value(fi, rv, addr); !omit {
			if fv.IsValid() {
				if opt.NestEmbed && fv.Kind() == reflect.Struct {
					v = reflectEmbed(fv, v, opt)
				} else {
					v = decompose(v, opt)
				}
			}
			condMapSet(obj, fi.key, v, opt)
		}
	}
	return obj
}

func reflectEmbed(rv reflect.Value, val any, opt *Options) any {
	obj := map[string]any{}
	si := getSinfo(val, opt.OmitEmpty)
	t := si.rt
	if 0 < len(opt.CreateKey) {
		if opt.FullTypePath {
			obj[opt.CreateKey] = t.PkgPath() + "/" + t.Name()
		} else {
			obj[opt.CreateKey] = t.Name()
		}
	}
	fields := si.getFields(opt)
	for _, fi := range fields {
		if v, fv, omit := fi.ivalue(fi, rv, 0); !omit {
			if fv.IsValid() {
				if opt.NestEmbed && fv.Kind() == reflect.Struct {
					v = reflectEmbed(fv, v, opt)
				} else {
					v = decompose(v, opt)
				}
			}
			condMapSet(obj, fi.key, v, opt)
		}
	}
	return obj
}

func reflectComplex(rv reflect.Value, opt *Options) any {
	c := rv.Complex()
	obj := map[string]any{
		"real": real(c),
		"imag": imag(c),
	}
	if 0 < len(opt.CreateKey) {
		obj[opt.CreateKey] = "complex"
	}
	return obj
}

func reflectMap(rv reflect.Value, opt *Options) any {
	obj := map[string]any{}
	it := rv.MapRange()
	for it.Next() {
		k := it.Key().Interface()
		var g any
		vv := it.Value()
		if !isNil(vv) {
			g = decompose(vv.Interface(), opt)
		}
		var (
			ks string
			ok bool
		)
		if ks, ok = k.(string); !ok {
			ks = fmt.Sprint(k)
		}
		condMapSet(obj, ks, g, opt)
	}
	return obj
}

func reflectArray(rv reflect.Value, opt *Options) any {
	size := rv.Len()
	a := make([]any, size)
	for i := size - 1; 0 <= i; i-- {
		a[i] = decompose(rv.Index(i).Interface(), opt)
	}
	return a
}

func isNil(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

func condMapSet(m map[string]any, key string, value any, opt *Options) {
	switch tv := value.(type) {
	case nil:
		if opt.OmitNil || opt.OmitEmpty {
			return
		}
	case string:
		if opt.OmitEmpty && len(tv) == 0 {
			return
		}
	case []any:
		if opt.OmitEmpty && len(tv) == 0 {
			return
		}
	case map[string]any:
		if opt.OmitEmpty && len(tv) == 0 {
			return
		}
	case bool:
		if opt.OmitEmpty && !tv {
			return
		}
	case int64:
		if opt.OmitEmpty && tv == 0 {
			return
		}
	}
	m[key] = value
}
