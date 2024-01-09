// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"unsafe"
)

const (
	strMask   = byte(0x01)
	omitMask  = byte(0x02)
	embedMask = byte(0x04)
)

var nilValue reflect.Value

type valFunc func(fi *finfo, rv reflect.Value, addr uintptr) (v any, fv reflect.Value, omit bool)

type finfo struct {
	rt     reflect.Type
	key    string
	value  valFunc
	ivalue valFunc
	index  []int
	offset uintptr
}

func valString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).String(), nilValue, false
}

func valStringNotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	s := rv.FieldByIndex(fi.index).String()
	if len(s) == 0 {
		return s, nilValue, true
	}
	return s, nilValue, false
}

func valJustVal(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	fv := rv.FieldByIndex(fi.index)
	return fv.Interface(), fv, false
}

func valPtrNotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	fv := rv.FieldByIndex(fi.index)
	v := fv.Interface()
	return v, fv, (*[2]uintptr)(unsafe.Pointer(&v))[1] == 0
}

func valSliceNotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	fv := rv.FieldByIndex(fi.index)
	if fv.Len() == 0 {
		return nil, nilValue, true
	}
	return fv.Interface(), fv, false
}

func valSimplifier(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface()
	if (*[2]uintptr)(unsafe.Pointer(&v))[1] == 0 {
		return nil, nilValue, false
	}
	return v.(Simplifier).Simplify(), nilValue, false
}

func valSimplifierAddr(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Addr().Interface()
	return v.(Simplifier).Simplify(), nilValue, false
}

func valGenericer(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface()
	if (*[2]uintptr)(unsafe.Pointer(&v))[1] == 0 {
		return nil, nilValue, false
	}
	if g, _ := v.(Genericer); g != nil {
		if n := g.Generic(); n != nil {
			return n.Simplify(), nilValue, false
		}
	}
	return nil, nilValue, false
}

func valGenericerAddr(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Addr().Interface()
	if g, _ := v.(Genericer); g != nil {
		if n := g.Generic(); n != nil {
			return n.Simplify(), nilValue, false
		}
	}
	return nil, nilValue, false
}

func newFinfo(f *reflect.StructField, key string, fx byte) *finfo {
	fi := finfo{
		rt:     f.Type,
		key:    key,
		index:  f.Index,
		value:  valJustVal, // replace as necessary later
		ivalue: valJustVal, // replace as necessary later
		offset: f.Offset,
	}
	// Check for interfaces first since almost any type can implement one of
	// the supported interfaces.
	vp := reflect.New(fi.rt).Interface()
	v := reflect.New(fi.rt).Elem().Interface()
	if _, ok := v.(Simplifier); ok {
		fi.value = valSimplifier
		fi.ivalue = valSimplifier
		return &fi
	}
	if _, ok := vp.(Simplifier); ok {
		fi.value = valSimplifierAddr
		fi.ivalue = valSimplifierAddr
		return &fi
	}
	if _, ok := v.(Genericer); ok {
		fi.value = valGenericer
		fi.ivalue = valGenericer
		return &fi
	}
	if _, ok := vp.(Genericer); ok {
		fi.value = valGenericerAddr
		fi.ivalue = valGenericerAddr
		return &fi
	}
	switch f.Type.Kind() {
	case reflect.Bool:
		fi.value = boolValFuncs[fx]
		fi.ivalue = boolValFuncs[fx|embedMask]

	case reflect.Int:
		fi.value = intValFuncs[fx]
		fi.ivalue = intValFuncs[fx|embedMask]
	case reflect.Int8:
		fi.value = int8ValFuncs[fx]
		fi.ivalue = int8ValFuncs[fx|embedMask]
	case reflect.Int16:
		fi.value = int16ValFuncs[fx]
		fi.ivalue = int16ValFuncs[fx|embedMask]
	case reflect.Int32:
		fi.value = int32ValFuncs[fx]
		fi.ivalue = int32ValFuncs[fx|embedMask]
	case reflect.Int64:
		fi.value = int64ValFuncs[fx]
		fi.ivalue = int64ValFuncs[fx|embedMask]

	case reflect.Uint:
		fi.value = uintValFuncs[fx]
		fi.ivalue = uintValFuncs[fx|embedMask]
	case reflect.Uint8:
		fi.value = uint8ValFuncs[fx]
		fi.ivalue = uint8ValFuncs[fx|embedMask]
	case reflect.Uint16:
		fi.value = uint16ValFuncs[fx]
		fi.ivalue = uint16ValFuncs[fx|embedMask]
	case reflect.Uint32:
		fi.value = uint32ValFuncs[fx]
		fi.ivalue = uint32ValFuncs[fx|embedMask]
	case reflect.Uint64:
		fi.value = uint64ValFuncs[fx]
		fi.ivalue = uint64ValFuncs[fx|embedMask]

	case reflect.Float32:
		fi.value = float32ValFuncs[fx]
		fi.ivalue = float32ValFuncs[fx|embedMask]
	case reflect.Float64:
		fi.value = float64ValFuncs[fx]
		fi.ivalue = float64ValFuncs[fx|embedMask]

	case reflect.String:
		if (fx & omitMask) != 0 {
			fi.value = valStringNotEmpty
			fi.ivalue = valStringNotEmpty
		} else {
			fi.value = valString
			fi.ivalue = valString
		}
	case reflect.Struct:
		fi.value = valJustVal
		fi.ivalue = valJustVal
	case reflect.Ptr:
		if (fx & omitMask) != 0 {
			fi.value = valPtrNotEmpty
			fi.ivalue = valPtrNotEmpty
		} else {
			fi.value = valJustVal
			fi.ivalue = valJustVal
		}
	case reflect.Interface:
		if (fx & omitMask) != 0 {
			fi.value = valPtrNotEmpty
			fi.ivalue = valPtrNotEmpty
		} else {
			fi.value = valJustVal
			fi.ivalue = valJustVal
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		if (fx & omitMask) != 0 {
			fi.value = valSliceNotEmpty
			fi.ivalue = valSliceNotEmpty
		} else {
			fi.value = valJustVal
			fi.ivalue = valJustVal
		}
	}
	return &fi
}
