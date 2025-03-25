// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strconv"
	"unsafe"
)

var int8ValFuncs = [8]valFunc{
	valInt8,
	valInt8AsString,
	valInt8NotEmpty,
	valInt8NotEmptyAsString,
	ivalInt8,
	ivalInt8AsString,
	ivalInt8NotEmpty,
	ivalInt8NotEmptyAsString,
}

func valInt8(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return *(*int8)(unsafe.Pointer(addr + fi.offset)), nilValue, false
}

func valInt8AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatInt(int64(*(*int8)(unsafe.Pointer(addr + fi.offset))), 10), nilValue, false
}

func valInt8NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*int8)(unsafe.Pointer(addr + fi.offset))
	return v, nilValue, v == 0
}

func valInt8NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*int8)(unsafe.Pointer(addr + fi.offset))
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatInt(int64(v), 10), nilValue, false
}

func ivalInt8(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).Interface().(int8), nilValue, false
}

func ivalInt8AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatInt(int64(rv.FieldByIndex(fi.index).Interface().(int8)), 10), nilValue, false
}

func ivalInt8NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(int8)
	return v, nilValue, v == 0
}

func ivalInt8NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(int8)
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatInt(int64(v), 10), nilValue, false
}
