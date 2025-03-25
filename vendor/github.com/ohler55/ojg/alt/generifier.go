// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/ohler55/ojg/gen"
)

// Genericer is the interface for the Generic() function that converts types
// to generic types.
type Genericer interface {

	// Generic should return a Node that represents the object. Generally this
	// includes the use of a creation key consistent with call to the
	// reflection based Generic() function.
	Generic() gen.Node
}

// Generify converts a value into Node compliant data. A best effort is made
// to convert values that are not simple into generic Nodes.
func Generify(v any, options ...*Options) (n gen.Node) {
	opt := &DefaultOptions
	if 0 < len(options) {
		opt = options[0]
	}
	if v != nil {
		switch tv := v.(type) {
		case bool:
			n = gen.Bool(tv)
		case gen.Bool:
			n = tv
		case int:
			n = gen.Int(int64(tv))
		case int8:
			n = gen.Int(int64(tv))
		case int16:
			n = gen.Int(int64(tv))
		case int32:
			n = gen.Int(int64(tv))
		case int64:
			n = gen.Int(tv)
		case uint:
			n = gen.Int(int64(tv))
		case uint8:
			n = gen.Int(int64(tv))
		case uint16:
			n = gen.Int(int64(tv))
		case uint32:
			n = gen.Int(int64(tv))
		case uint64:
			n = gen.Int(int64(tv))
		case gen.Int:
			n = tv
		case float32:
			n = gen.Float(float64(tv))
		case float64:
			n = gen.Float(tv)
		case gen.Float:
			n = tv
		case string:
			n = gen.String(tv)
		case gen.String:
			n = tv
		case time.Time:
			n = gen.Time(tv)
		case gen.Time:
			n = tv
		case []any:
			a := make(gen.Array, len(tv))
			for i, m := range tv {
				a[i] = Generify(m, opt)
			}
			n = a
		case map[string]any:
			o := gen.Object{}
			for k, m := range tv {
				g := Generify(m, opt)
				// TBD OmitEmpty
				if g != nil || !opt.OmitNil {
					o[k] = g
				}
			}
			n = o
		default:
			var ok bool
			if n, ok = v.(gen.Node); ok {
				return
			}
			if g, _ := v.(Genericer); g != nil {
				return g.Generic()
			}
			if simp, _ := v.(Simplifier); simp != nil {
				return Generify(simp.Simplify(), opt)
			}
			return reflectGenData(v, opt)
		}
	}
	return
}

// GenAlter converts a simple go data element into Node compliant data. A best
// effort is made to convert values that are not simple into generic Nodes. It
// modifies the values inplace if possible by altering the original.
func GenAlter(v any, options ...*Options) (n gen.Node) {
	opt := &DefaultOptions
	if 0 < len(options) {
		opt = options[0]
	}
	if v != nil {
		switch tv := v.(type) {
		case bool:
			n = gen.Bool(tv)
		case gen.Bool:
			n = tv
		case int:
			n = gen.Int(int64(tv))
		case int8:
			n = gen.Int(int64(tv))
		case int16:
			n = gen.Int(int64(tv))
		case int32:
			n = gen.Int(int64(tv))
		case int64:
			n = gen.Int(tv)
		case uint:
			n = gen.Int(int64(tv))
		case uint8:
			n = gen.Int(int64(tv))
		case uint16:
			n = gen.Int(int64(tv))
		case uint32:
			n = gen.Int(int64(tv))
		case uint64:
			n = gen.Int(int64(tv))
		case gen.Int:
			n = tv
		case float32:
			n = gen.Float(float64(tv))
		case float64:
			n = gen.Float(tv)
		case gen.Float:
			n = tv
		case string:
			n = gen.String(tv)
		case gen.String:
			n = tv
		case time.Time:
			n = gen.Time(tv)
		case []any:
			a := *(*gen.Array)(unsafe.Pointer(&tv))
			for i, m := range tv {
				a[i] = GenAlter(m)
			}
			n = a
		case map[string]any:
			o := *(*gen.Object)(unsafe.Pointer(&tv))
			var delKeys []string
			// TBD OmitEmpty
			for k, m := range tv {
				g := GenAlter(m, opt)
				if g != nil || !opt.OmitNil {
					o[k] = g
				} else {
					// TBD delete in place
					delKeys = append(delKeys, k)
				}
			}
			for _, k := range delKeys {
				delete(o, k)
			}
			n = o
		default:
			var ok bool
			if n, ok = v.(gen.Node); ok {
				return
			}
			if g, _ := v.(Genericer); g != nil {
				return g.Generic()
			}
			if simp, _ := v.(Simplifier); simp != nil {
				return GenAlter(simp.Simplify(), opt)
			}
			return reflectGenData(v, opt)
		}
	}
	return
}

func reflectGenData(data any, opt *Options) gen.Node {
	return reflectGenValue(reflect.ValueOf(data), opt)
}

func reflectGenValue(rv reflect.Value, opt *Options) (v gen.Node) {
	switch rv.Kind() {
	case reflect.Invalid, reflect.Uintptr, reflect.UnsafePointer, reflect.Chan, reflect.Func, reflect.Interface:
		v = nil
	case reflect.Complex64, reflect.Complex128:
		v = reflectGenComplex(rv, opt)
	case reflect.Map:
		v = reflectGenMap(rv, opt)
	case reflect.Ptr:
		v = reflectGenValue(rv.Elem(), opt)
	case reflect.Slice, reflect.Array:
		v = reflectGenArray(rv, opt)
	case reflect.Struct:
		v = reflectGenStruct(rv, opt)
	}
	return
}

func reflectGenStruct(rv reflect.Value, opt *Options) gen.Node {
	obj := gen.Object{}
	t := rv.Type()
	if 0 < len(opt.CreateKey) {
		if opt.FullTypePath {
			obj[opt.CreateKey] = gen.String(t.PkgPath() + "/" + t.Name())
		} else {
			obj[opt.CreateKey] = gen.String(t.Name())
		}
	}
	for i := rv.NumField() - 1; 0 <= i; i-- {
		name := []byte(t.Field(i).Name)
		if len(name) == 0 || 'a' <= name[0] {
			// not a public field
			continue
		}
		name[0] |= 0x20
		g := Generify(rv.Field(i).Interface(), opt)
		// TBD OmitEmpty
		if g != nil || !opt.OmitNil {
			obj[string(name)] = g
		}
	}
	return obj
}

func reflectGenComplex(rv reflect.Value, opt *Options) gen.Node {
	c := rv.Complex()
	obj := gen.Object{
		"real": gen.Float(real(c)),
		"imag": gen.Float(imag(c)),
	}
	if 0 < len(opt.CreateKey) {
		obj[opt.CreateKey] = gen.String("complex")
	}
	return obj
}

func reflectGenMap(rv reflect.Value, opt *Options) gen.Node {
	obj := gen.Object{}
	it := rv.MapRange()
	for it.Next() {
		k := it.Key().Interface()
		g := Generify(it.Value().Interface(), opt)
		// TBD OmitEmpty
		if g != nil || !opt.OmitNil {
			if ks, ok := k.(string); ok {
				obj[ks] = g
			} else {
				obj[fmt.Sprint(k)] = g
			}
		}
	}
	return obj
}

func reflectGenArray(rv reflect.Value, opt *Options) gen.Node {
	size := rv.Len()
	a := make(gen.Array, size)
	for i := size - 1; 0 <= i; i-- {
		a[i] = Generify(rv.Index(i).Interface(), opt)
	}
	return a
}
