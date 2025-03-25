// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/gen"
)

// DefaultRecomposer provides a shared Recomposer. Note that this should not
// be shared across go routines unless all types that will be used are
// registered first. That can be done explicitly or with a warm up run.
var DefaultRecomposer = Recomposer{
	composers: map[string]*composer{},
}

// RecomposeFunc should build an object from data in a map returning the
// recomposed object or an error.
type RecomposeFunc func(map[string]any) (any, error)

// RecomposeAnyFunc should build an object from data in an any
// returning the recomposed object or an error.
type RecomposeAnyFunc func(any) (any, error)

// Recomposer is used to recompose simple data into structs.
type Recomposer struct {

	// CreateKey identifies the creation key in decomposed objects.
	CreateKey string

	composers map[string]*composer

	// NumConvMethod specifies the json.Number conversion method.
	NumConvMethod ojg.NumConvMethod
}

var jsonUnmarshalerType reflect.Type

func init() {
	jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()

}

// RegisterComposer regsiters a composer function for a value type. A nil
// function will still register the default composer which uses reflection.
func (r *Recomposer) RegisterComposer(val any, fun RecomposeFunc) error {
	_, err := r.registerComposer(reflect.TypeOf(val), fun)

	return err
}

// RegisterAnyComposer regsiters a composer function for a value type. A nil
// function will still register the default composer which uses reflection.
func (r *Recomposer) RegisterAnyComposer(val any, fun RecomposeAnyFunc) error {
	_, err := r.registerAnyComposer(reflect.TypeOf(val), fun)

	return err
}

// RegisterUnmarshalerComposer regsiters a composer function for a named
// value. This is only used to register cross package json.Unmarshaler
// composer which returns []byte.
func (r *Recomposer) RegisterUnmarshalerComposer(fun RecomposeAnyFunc) {
	name := "json.Unmarshaler"
	r.composers[name] = &composer{
		any:   fun,
		short: name,
		full:  name,
	}
}

func (r *Recomposer) registerComposer(rt reflect.Type, fun RecomposeFunc) (*composer, error) {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	full := rt.PkgPath() + "/" + rt.Name()
	// TBD could loosen this up and allow any type as long as a function is provided.
	if rt.Kind() != reflect.Struct {
		return nil, fmt.Errorf("only structs can be recomposed. %s is not a struct type", rt)
	}
	c := r.composers[full]
	if c == nil {
		c = &composer{
			fun:   fun,
			short: rt.Name(),
			full:  full,
			rtype: rt,
		}
		c.indexes = indexType(c.rtype)
		r.composers[c.short] = c
		r.composers[c.full] = c
	} else {
		if fun != nil {
			c.fun = fun
		}
		// If already registered then there is no reason to walk the fields again.
		return c, nil
	}
	for i := rt.NumField() - 1; 0 <= i; i-- {
		f := rt.Field(i)
		// Private fields should be skipped.
		if len(f.Name) == 0 || ([]byte(f.Name)[0]&0x20) != 0 {
			continue
		}
		ft := f.Type
		switch ft.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Ptr:
			ft = ft.Elem()
		}
		if _, has := r.composers[ft.Name()]; has {
			continue
		}
		_, _ = r.registerComposer(ft, nil)
	}
	return c, nil
}

func (r *Recomposer) registerAnyComposer(rt reflect.Type, fun RecomposeAnyFunc) (*composer, error) {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	full := rt.PkgPath() + "/" + rt.Name()
	if rt.Kind() != reflect.Struct {
		return nil, fmt.Errorf("only structs can be recomposed. %s is not a struct type", rt)
	}
	c := r.composers[full]
	if c == nil {
		c = &composer{
			any:   fun,
			short: rt.Name(),
			full:  full,
			rtype: rt,
		}
		c.indexes = indexType(c.rtype)
		r.composers[c.short] = c
		r.composers[c.full] = c
	} else {
		c.any = fun
	}
	return c, nil
}

// Recompose simple data into more complex go types.
func (r *Recomposer) Recompose(v any, tv ...any) (out any, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = ojg.NewError(rec)
			out = nil
		}
	}()
	out = r.MustRecompose(v, tv...)
	return
}

// MustRecompose simple data into more complex go types.
func (r *Recomposer) MustRecompose(v any, tv ...any) (out any) {
	if 0 < len(tv) {
		if um, ok := tv[0].(json.Unmarshaler); ok {
			if comp := r.composers["json.Unmarshaler"]; comp != nil {
				b, _ := comp.any(v) // Special case. Must return []byte.
				if err := um.UnmarshalJSON(b.([]byte)); err != nil {
					panic(err)
				}
				return um
			}
		}
		out = tv[0]
		rv := reflect.ValueOf(tv[0])
		switch rv.Kind() {
		case reflect.Array, reflect.Slice:
			rv = reflect.New(rv.Type())
			r.recomp(v, rv)
			out = rv.Elem().Interface()
		case reflect.Map:
			r.recomp(v, rv)
		case reflect.Ptr:
			r.recomp(v, rv)
			switch rv.Elem().Kind() {
			case reflect.Slice, reflect.Array, reflect.Map, reflect.Interface:
				out = rv.Elem().Interface()
			}
		default:
			panic(fmt.Errorf("only a slice, map, or pointer is allowed as an optional argument"))
		}
	} else {
		out = r.recompAny(v)
	}
	return
}

func (r *Recomposer) recompAny(v any) any {
	switch tv := v.(type) {
	case nil, bool, int64, float64, string, time.Time:
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
			a[i] = r.recompAny(m)
		}
		v = a
	case map[string]any:
		if cv := tv[r.CreateKey]; cv != nil {
			tn, _ := cv.(string)
			if c := r.composers[tn]; c != nil {
				if c.fun != nil {
					val, err := c.fun(tv)
					if err != nil {
						panic(err)
					}
					return val
				}
				rv := reflect.New(c.rtype)
				r.recomp(v, rv)
				return rv.Interface()
			}
		}
		o := map[string]any{}
		for k, m := range tv {
			o[k] = r.recompAny(m)
		}
		v = o

	case gen.Bool:
		v = bool(tv)
	case gen.Int:
		v = int64(tv)
	case gen.Float:
		v = float64(tv)
	case gen.String:
		v = string(tv)
	case gen.Time:
		v = time.Time(tv)
	case gen.Big:
		v = string(tv)
	case gen.Array:
		a := make([]any, len(tv))
		for i, m := range tv {
			a[i] = r.recompAny(m)
		}
		v = a
	case gen.Object:
		if cv := tv[r.CreateKey]; cv != nil {
			gn, _ := cv.(gen.String)
			tn := string(gn)
			if c := r.composers[tn]; c != nil {
				simple, _ := tv.Simplify().(map[string]any)
				if c.fun != nil {
					val, err := c.fun(simple)
					if err != nil {
						panic(err)
					}
					return val
				}
				rv := reflect.New(c.rtype)
				r.recomp(simple, rv)
				return rv.Interface()
			}
		}
		o := map[string]any{}
		for k, m := range tv {
			o[k] = r.recompAny(m)
		}
		v = o

	case json.Number:
		switch r.NumConvMethod {
		case ojg.NumConvFloat64:
			var err error
			if v, err = tv.Float64(); err != nil {
				panic(err)
			}
		case ojg.NumConvString:
			v = tv.String()
		}
	default:
		panic(fmt.Errorf("can not recompose a %T", v))
	}
	return v
}

func (r *Recomposer) recomp(v any, rv reflect.Value) {
	as, _ := rv.Interface().(AttrSetter)
	if rv.Kind() == reflect.Ptr {
		if v == nil {
			return
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice:
		va, ok := (v).([]any)
		if !ok {
			vv := reflect.ValueOf(v)
			if vv.Kind() != reflect.Slice {
				panic(fmt.Errorf("can only recompose a %s from a []any, not a %T", rv.Type(), v))
			}
			va = make([]any, vv.Len())
			for i := len(va) - 1; 0 <= i; i-- {
				va[i] = vv.Index(i).Interface()
			}
		}
		size := len(va)
		av := reflect.MakeSlice(rv.Type(), size, size)
		et := av.Type().Elem()
		if et.Kind() == reflect.Ptr {
			et = et.Elem()
			for i := 0; i < size; i++ {
				ev := reflect.New(et)
				r.recomp(va[i], ev)
				av.Index(i).Set(ev)
			}
		} else {
			for i := 0; i < size; i++ {
				r.setValue(va[i], av.Index(i), nil)
			}
		}
		rv.Set(av)
	case reflect.Array:
		vv := reflect.ValueOf(v)
		if vv.Kind() != reflect.Slice {
			panic(fmt.Errorf("can only recompose a %s from a []any, not a %T", rv.Type(), v))
		}
		inSize := vv.Len()
		size := rv.Len()
		for i := 0; i < inSize; i++ {
			if size <= i {
				break
			}
			// Kind of awkward but the double reflect is needed to get the
			// actual type of the element value if the slice input is []any.
			ev := vv.Index(i).Interface()
			ri := rv.Index(i)
			r.setValue(ev, ri, nil)
		}
	case reflect.Map:
		if v == nil {
			return
		}
		et := rv.Type().Elem()
		vm, ok := (v).(map[string]any)
		if !ok {
			vv := reflect.ValueOf(v)
			if vv.Kind() != reflect.Map {
				panic(fmt.Errorf("can only recompose a map from a map[string]any, not a %T", v))
			}
			vm = map[string]any{}
			iter := vv.MapRange()
			for iter.Next() {
				k := iter.Key().Interface().(string)
				vm[k] = iter.Value().Interface()
			}
		}
		if rv.IsNil() {
			rv.Set(reflect.MakeMapWithSize(rv.Type(), len(vm)))
		}
		switch {
		case et.Kind() == reflect.Interface:
			for k, m := range vm {
				rv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(r.recompAny(m)))
			}
		case et.Kind() == reflect.Ptr:
			et = et.Elem()
			for k, m := range vm {
				ev := reflect.New(et)
				r.recomp(m, ev)
				rv.SetMapIndex(reflect.ValueOf(k), ev)
			}
		default:
			for k, m := range vm {
				ev := reflect.New(et)
				r.recomp(m, ev)
				rv.SetMapIndex(reflect.ValueOf(k), ev.Elem())
			}
		}
	case reflect.Struct:
		vm, ok := (v).(map[string]any)
		if !ok {
			if c := r.composers[rv.Type().Name()]; c != nil && c.any != nil {
				if val, err := c.any(v); err == nil {
					if val == nil {
						break
					}
					vv := reflect.ValueOf(val)
					if vv.Type().Kind() == reflect.Ptr {
						vv = vv.Elem()
					}
					rv.Set(vv)
				} else {
					panic(err)
				}
				break
			}
			vv := reflect.ValueOf(v)
			if vv.Kind() != reflect.Map {
				panic(fmt.Errorf("can only recompose a %s from a map[string]any, not a %T", rv.Type(), v))
			}
			vm = map[string]any{}
			iter := vv.MapRange()
			for iter.Next() {
				k := iter.Key().Interface().(string)
				vm[k] = iter.Value().Interface()
			}
		}
		if as != nil {
			for k, m := range vm {
				if r.CreateKey == k {
					continue
				}
				if err := as.SetAttr(k, m); err != nil {
					panic(err)
				}
			}
			return
		}
		var im map[string]reflect.StructField
		if c := r.composers[rv.Type().Name()]; c != nil {
			if c.fun != nil {
				if val, err := c.fun(vm); err == nil {
					vv := reflect.ValueOf(val)
					if vv.Type().Kind() == reflect.Ptr {
						vv = vv.Elem()
					}
					rv.Set(vv)
				} else {
					panic(err)
				}
				break
			}
			im = c.indexes
		} else {
			c, _ = r.registerComposer(rv.Type(), nil)
			im = c.indexes
		}
		for k := range im {
			sf := im[k]
			f := rv.FieldByIndex(sf.Index)
			var m any
			var has bool
			if m, has = vm[k]; !has {
				if m, has = vm[sf.Name]; !has {
					name := []byte(sf.Name)
					name[0] |= 0x20
					if m, has = vm[string(name)]; !has {
						m, has = vm[strings.ToLower(string(name))]
					}
				}
			}
			if has && m != nil {
				r.setValue(m, f, &sf)
			}
		}
	case reflect.Interface:
		v = r.recompAny(v)
		rv.Set(reflect.ValueOf(v))

	case reflect.Bool:
		rv.Set(reflect.ValueOf(v))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		rv.Set(reflect.ValueOf(v).Convert(rv.Type()))

	default:
		panic(fmt.Errorf("can not convert (%T)%v to a %s", v, v, rv.Type()))
	}
}

func (r *Recomposer) setValue(v any, rv reflect.Value, sf *reflect.StructField) {
	switch rv.Kind() {
	case reflect.Bool:
		if s, ok := v.(string); ok && sf != nil && strings.Contains(sf.Tag.Get("json"), ",string") {
			if b, err := strconv.ParseBool(s); err == nil {
				rv.Set(reflect.ValueOf(b))
			} else {
				panic(err)
			}
		} else {
			rv.Set(reflect.ValueOf(v))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if s, ok := v.(string); ok && sf != nil && strings.Contains(sf.Tag.Get("json"), ",string") {
			if i, err := strconv.Atoi(s); err == nil {
				rv.Set(reflect.ValueOf(i).Convert(rv.Type()))
			} else {
				panic(err)
			}
		} else if jn, jok := v.(json.Number); jok {
			if i, err := jn.Int64(); err == nil {
				rv.Set(reflect.ValueOf(i).Convert(rv.Type()))
			} else {
				panic(err)
			}
		} else {
			rv.Set(reflect.ValueOf(v).Convert(rv.Type()))
		}
	case reflect.Float32, reflect.Float64:
		if s, ok := v.(string); ok && sf != nil && strings.Contains(sf.Tag.Get("json"), ",string") {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				rv.Set(reflect.ValueOf(f).Convert(rv.Type()))
			} else {
				panic(err)
			}
		} else if jn, jok := v.(json.Number); jok {
			if f, err := jn.Float64(); err == nil {
				rv.Set(reflect.ValueOf(f).Convert(rv.Type()))
			} else {
				panic(err)
			}
		} else {
			rv.Set(reflect.ValueOf(v).Convert(rv.Type()))
		}
	case reflect.String:
		rv.Set(reflect.ValueOf(v).Convert(rv.Type()))
	case reflect.Interface:
		v = r.recompAny(v)
		rv.Set(reflect.ValueOf(v))
	case reflect.Ptr:
		ev := reflect.New(rv.Type().Elem())
		r.recomp(v, ev)
		rv.Set(ev)
	default:
		if reflect.PtrTo(rv.Type()).Implements(jsonUnmarshalerType) {
			ev := rv.Addr().Interface().(json.Unmarshaler)
			if comp := r.composers["json.Unmarshaler"]; comp != nil {
				b, _ := comp.any(v) // Special case. Must return []byte.
				if err := ev.UnmarshalJSON(b.([]byte)); err != nil {
					panic(err)
				}
				return
			}
		}
		r.recomp(v, rv)
	}
}
