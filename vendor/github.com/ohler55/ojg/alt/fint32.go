// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strconv"
	"unsafe"
)

var int32ValFuncs = [8]valFunc{
	valInt32,
	valInt32AsString,
	valInt32NotEmpty,
	valInt32NotEmptyAsString,
	ivalInt32,
	ivalInt32AsString,
	ivalInt32NotEmpty,
	ivalInt32NotEmptyAsString,
}

func valInt32(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return *(*int32)(unsafe.Pointer(addr + fi.offset)), nilValue, false
}

func valInt32AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatInt(int64(*(*int32)(unsafe.Pointer(addr + fi.offset))), 10), nilValue, false
}

func valInt32NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*int32)(unsafe.Pointer(addr + fi.offset))
	return v, nilValue, v == 0
}

func valInt32NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*int32)(unsafe.Pointer(addr + fi.offset))
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatInt(int64(v), 10), nilValue, false
}

func ivalInt32(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).Interface().(int32), nilValue, false
}

func ivalInt32AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatInt(int64(rv.FieldByIndex(fi.index).Interface().(int32)), 10), nilValue, false
}

func ivalInt32NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(int32)
	return v, nilValue, v == 0
}

func ivalInt32NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(int32)
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatInt(int64(v), 10), nilValue, false
}
