// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strconv"
	"unsafe"
)

var uint64ValFuncs = [8]valFunc{
	valUint64,
	valUint64AsString,
	valUint64NotEmpty,
	valUint64NotEmptyAsString,
	ivalUint64,
	ivalUint64AsString,
	ivalUint64NotEmpty,
	ivalUint64NotEmptyAsString,
}

func valUint64(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return *(*uint64)(unsafe.Pointer(addr + fi.offset)), nilValue, false
}

func valUint64AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatUint(*(*uint64)(unsafe.Pointer(addr + fi.offset)), 10), nilValue, false
}

func valUint64NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*uint64)(unsafe.Pointer(addr + fi.offset))
	return v, nilValue, v == 0
}

func valUint64NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*uint64)(unsafe.Pointer(addr + fi.offset))
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatUint(v, 10), nilValue, false
}

func ivalUint64(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).Interface().(uint64), nilValue, false
}

func ivalUint64AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatUint(rv.FieldByIndex(fi.index).Interface().(uint64), 10), nilValue, false
}

func ivalUint64NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(uint64)
	return v, nilValue, v == 0
}

func ivalUint64NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(uint64)
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatUint(v, 10), nilValue, false
}
