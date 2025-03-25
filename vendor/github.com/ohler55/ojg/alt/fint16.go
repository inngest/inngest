// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strconv"
	"unsafe"
)

var int16ValFuncs = [8]valFunc{
	valInt16,
	valInt16AsString,
	valInt16NotEmpty,
	valInt16NotEmptyAsString,
	ivalInt16,
	ivalInt16AsString,
	ivalInt16NotEmpty,
	ivalInt16NotEmptyAsString,
}

func valInt16(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return *(*int16)(unsafe.Pointer(addr + fi.offset)), nilValue, false
}

func valInt16AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatInt(int64(*(*int16)(unsafe.Pointer(addr + fi.offset))), 10), nilValue, false
}

func valInt16NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*int16)(unsafe.Pointer(addr + fi.offset))
	return v, nilValue, v == 0
}

func valInt16NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*int16)(unsafe.Pointer(addr + fi.offset))
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatInt(int64(v), 10), nilValue, false
}

func ivalInt16(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).Interface().(int16), nilValue, false
}

func ivalInt16AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatInt(int64(rv.FieldByIndex(fi.index).Interface().(int16)), 10), nilValue, false
}

func ivalInt16NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(int16)
	return v, nilValue, v == 0
}

func ivalInt16NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(int16)
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatInt(int64(v), 10), nilValue, false
}
